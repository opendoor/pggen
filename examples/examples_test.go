// package examples_test runs each example as a test.
//
// In order to be tested by this driver, examples must follow a special pattern. The
// schema for an example must be defined in a file called `db.sql` in the root of the
// example directory. This schema file should include the preamble:
//
// ```sql
// DROP SCHEMA public CASCADE;
// CREATE SCHEMA public;
// ```
//
// This is because the database schema is not garenteed to be empty when the schema file
// is run.
//
// The example must have a file called `main.go` in its root. This serves as the entrypoint
// to the example.
//
// The example must have a file caled `output.txt` in its root. The stdout of the running `main.go`
// will be checked against this file.
//
// The example must have a subdirectory called `models` which contains its generated code. This
// code will be regenerated and diffed against.
//
// In order to focus on just a single example, run with PGGEN_TEST_EXAMPLE=<examples>
// where <examples> is a comma seperated list of directory names containing the examples
// to run.
//
// By default, this package will test all of the examples found in the examples directory.
//
// System Requirements: The `go` tool must be installed as this package shells out to `go run`.
package examples_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"testing"

	ensureSchema "github.com/opendoor/pggen/tools/ensure-schema/lib"
)

// for examples with output that can vary with time, we need to fudge our output
// matching a little. This table of regular expressions configures that fudging.
var expectedDiffs = map[string]string{
	"timestamps": "(?s).*CreatedAt.*UpdatedAt.*CreatedAt.*UpdatedAt.*DeletedAt.*",
}

func TestExamples(t *testing.T) {
	examples, err := ioutil.ReadDir(".")
	chkErr(t, err)

	exampleAllowed := getExampleAllowed()

	for _, e := range examples {
		if !e.IsDir() {
			continue
		}

		if !exampleAllowed(e.Name()) {
			continue
		}

		err = runExample(e.Name())
		chkErr(t, err)
	}

	// restore standard schema so that unit tests will continue to work if run
	// afterward
	err = ensureSchema.PopulateDB(path.Join("..", "cmd", "pggen", "test", "db.sql"))
	chkErr(t, err)
}

// runExample runs and tests an example
func runExample(exampleName string) error {
	err := generateCode(exampleName)
	if err != nil {
		return fmt.Errorf("generating code for '%s': %s", exampleName, err.Error())
	}

	outputFile, err := os.Open(path.Join(exampleName, "output.txt"))
	if err != nil {
		return fmt.Errorf("%s: opening output file: %s", exampleName, err.Error())
	}
	defer outputFile.Close()
	outputReader := bufio.NewReader(outputFile)
	expectedOutput, err := ioutil.ReadAll(outputReader)
	if err != nil {
		return fmt.Errorf("%s: reading output: %s", exampleName, err.Error())
	}

	runCmd := exec.Command("go", "run", path.Join(exampleName, "main.go"))
	actualOutput, err := runCmd.Output()
	if err != nil {
		return fmt.Errorf("running '%s': %s", exampleName, err.Error())
	}

	if !bytes.Equal(expectedOutput, actualOutput) {
		diff, err := displayDiff(expectedOutput, actualOutput)
		if err != nil {
			return fmt.Errorf(
				"%s: error diffing output: %s\nEXPECTED:\n%s\nACTUAL:\n%s\n",
				exampleName,
				err.Error(),
				string(expectedOutput),
				string(actualOutput),
			)
		}

		expectedDiffRE, inMap := expectedDiffs[exampleName]
		if inMap {
			re := regexp.MustCompile(expectedDiffRE)
			if !re.Match(diff) {
				return fmt.Errorf(
					"%s: expected diff re /%s/ failed to match diff:\n%s",
					exampleName,
					expectedDiffRE,
					string(diff),
				)
			}
		} else {
			return fmt.Errorf("%s: different output:\n%s", exampleName, diff)
		}
	}

	return nil
}

var genFileRE = regexp.MustCompile(".*\\.gen\\.go$")

// generateCode generates code for the given example and checks to make sure the result
// matches up with the code that is checked in.
func generateCode(exampleName string) error {
	tmpDir, err := ioutil.TempDir("", "example_test_gen")
	if err != nil {
		return fmt.Errorf("creating tmp dir: %s", err.Error())
	}
	defer os.RemoveAll(tmpDir)

	// copy the generated files into our scratch space for later diffing
	modelsDir := path.Join(exampleName, "models")
	modelsFiles, err := ioutil.ReadDir(modelsDir)
	if err != nil {
		return fmt.Errorf("reading models dir: %s", err.Error())
	}
	for _, file := range modelsFiles {
		if !genFileRE.MatchString(file.Name()) {
			continue
		}
		err = copyFile(path.Join(modelsDir, file.Name()), path.Join(tmpDir, file.Name()))
		if err != nil {
			return fmt.Errorf("failed to move '%s': %s", path.Join(modelsDir, file.Name()), err.Error())
		}
	}

	// actually generate the code
	var errCollector strings.Builder
	genCmd := exec.Command("go", "generate", "./"+exampleName+"/...")
	genCmd.Stderr = &errCollector
	err = genCmd.Run()
	if err != nil {
		return fmt.Errorf("generating code: %s: %s", err.Error(), errCollector.String())
	}

	// diff the generated files against the expected generated code
	expectedModelsFiles, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		return fmt.Errorf("reading tmp dir: %s", err.Error())
	}
	for _, file := range expectedModelsFiles {
		diff, err := displayFileDiff(path.Join(tmpDir, file.Name()), path.Join(modelsDir, file.Name()))
		if err != nil {
			return fmt.Errorf("diffing generated code: %s", err.Error())
		}

		if len(strings.TrimSpace(string(diff))) > 0 {
			return fmt.Errorf("generated code different than what is checked in:\n%s", string(diff))
		}
	}

	return nil
}

// getAllowedExamples returns a closure which answers if a given example should be run
func getExampleAllowed() func(exampleName string) bool {
	focusedExamples, inEnv := os.LookupEnv("PGGEN_TEST_EXAMPLE")
	if !inEnv {
		return func(exampleName string) bool {
			return true
		}
	}

	examples := strings.Split(focusedExamples, ",")
	allowSet := map[string]struct{}{}
	for _, e := range examples {
		allowSet[e] = struct{}{}
	}
	return func(exampleName string) bool {
		_, inSet := allowSet[exampleName]
		return inSet
	}
}

// displayDiff dumps the args to files and shells out to `diff` to determine their differences
func displayDiff(lhs []byte, rhs []byte) ([]byte, error) {
	tmpDir, err := ioutil.TempDir("", "example_test_diff")
	if err != nil {
		return nil, fmt.Errorf("creating tmp dir: %s", err.Error())
	}
	defer os.RemoveAll(tmpDir)

	lhsFile := path.Join(tmpDir, "lhs.txt")
	rhsFile := path.Join(tmpDir, "rhs.txt")

	err = ioutil.WriteFile(lhsFile, lhs, 0400)
	if err != nil {
		return nil, fmt.Errorf("writing lhs: %s", err.Error())
	}
	err = ioutil.WriteFile(rhsFile, rhs, 0400)
	if err != nil {
		return nil, fmt.Errorf("writing lhs: %s", err.Error())
	}

	return displayFileDiff(lhsFile, rhsFile)
}

// displayFileDiff shells out to `diff` to diff the two given files
func displayFileDiff(lhsFile string, rhsFile string) ([]byte, error) {
	diffCmd := exec.Command("diff", lhsFile, rhsFile)
	out, err := diffCmd.Output()
	if err != nil {
		_, isExitErr := err.(*exec.ExitError)
		if !isExitErr {
			return nil, fmt.Errorf("diffing files: %s", err.Error())
		}
	}

	return out, nil
}

func copyFile(src string, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

func chkErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
