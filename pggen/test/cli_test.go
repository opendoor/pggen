package test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"testing"
	"text/template"
)

// cli_test.go contains tests that execute pggen as a subprocess and perform
// assertions about the outputs. For the most part, these tests are used
// to verifty that appropriate error messages are generated for various
// situations.

type cliCase struct {
	// The contents of the toml file to use
	toml string
	// A template string to use to create the the command line flags.
	// Defaults to "{{ .Exe }} -o {{ .Output }} {{ .Toml }}".
	cmd string
	// If non-empty, a regex that must match the stdout of the process.
	stdoutRE string
	// If non-empty, a regex that must match the stderr of the procees.
	stderrRE string
	// The expected exit code of the process.
	exitCode int
}

// The context struct that the `cmd` template is instantiated in
type cmdCtx struct {
	// The name of the exe to invoke
	Exe string
	// The name of the output file that the command must produce
	Output string
	// The name of the toml file
	Toml string
}

var cliCases = []cliCase{
	// help text
	{
		cmd:      "{{ .Exe }} -h",
		exitCode: 0,
		stdoutRE: "(?s)Usage:.*Args:.*Options:",
	},
	{
		cmd:      "{{ .Exe }} --help",
		exitCode: 0,
		stdoutRE: "(?s)Usage:.*Args:.*Options:",
	},

	// malformed args
	{
		cmd:      "{{ .Exe }}",
		exitCode: 1,
		stderrRE: "(?s)Usage:.*Args:.*Options:",
	},
	{
		cmd:      "{{ .Exe }} --bad-arg {{ .Toml }}",
		exitCode: 1,
		stderrRE: "(?s)Usage:.*Args:.*Options:",
	},

	// specific error messages
	{
		toml: `
[[table]]
name = "small_entities"

[[query]]
    name = "GetSmallEntityByID"
    body = '''
    SELECT * FROM small_entities WHERE id = $1
    '''
    return_type = "SmallEntity"
	null_flags = "--"
		`,
		exitCode: 1,
		stderrRE: "don't set null flags.*returning table struct",
		stdoutRE: "generating 1 queries",
	},
	{
		toml: `
[[stored_function]]
    name = "returns_text"
	return_type = "Bad"
		`,
		exitCode: 1,
		stderrRE: "return_type cannot be provided.*primitive",
		stdoutRE: "(?s)stored functions.*generating query 'returns_text'",
	},
	{
		// malformed toml
		toml: `
[[stored_function]]
    name "returns_text"
	return_type = "Bad"
		`,
		exitCode: 1,
		stderrRE: "while parsing config file",
	},
}

func TestCLI(t *testing.T) {
	testDir, err := ioutil.TempDir("", "pggen_cli_test")
	chkErr(t, err)
	defer os.RemoveAll(testDir)

	repoRoot, err := getRepoRoot()
	chkErr(t, err)

	exe := path.Join(testDir, "pggen")
	mainSrc := path.Join(repoRoot, "pggen", "main.go")

	// build the executable we are going to be testing
	cmd := exec.Command("go", "build", "-o", exe, mainSrc)
	err = cmd.Run()
	chkErr(t, err)

	for i, test := range cliCases {
		err = runCLITest(i, exe, testDir, &test)
		if err != nil {
			t.Fatalf("While running cli test case %d:\n%s\n", i, err)
		}
	}
}

func runCLITest(
	testNo int,
	exe string,
	testDir string,
	test *cliCase,
) (err error) {
	testDirStem := fmt.Sprintf("test%d", testNo)
	caseDir, err := ioutil.TempDir(testDir, testDirStem)
	if err != nil {
		return err
	}

	tomlFile := path.Join(caseDir, "test.toml")
	err = ioutil.WriteFile(tomlFile, []byte(test.toml), 0755)
	if err != nil {
		return err
	}

	outModule := path.Join(caseDir, "models")
	err = os.Mkdir(outModule, 0755)
	if err != nil {
		return err
	}
	outPath := path.Join(outModule, "pggen.gen.go")

	cmd := test.cmd
	if len(cmd) == 0 {
		cmd = "{{ .Exe }} -o {{ .Output }} {{ .Toml }}"
	}

	// execute the command template
	cmdTmplContext := cmdCtx{
		Exe:    exe,
		Output: outPath,
		Toml:   tomlFile,
	}
	cmdTmpl, err := template.New("cmd-tmpl").Parse(cmd)
	if err != nil {
		return err
	}
	var cmdTxtBuilder strings.Builder
	err = cmdTmpl.Execute(&cmdTxtBuilder, cmdTmplContext)
	if err != nil {
		return err
	}
	cmdTxt := cmdTxtBuilder.String()

	defer func() {
		if err != nil {
			err = fmt.Errorf("CMD: %s\n%s", cmdTxt, err.Error())
		}
	}()

	// convert the command text into an `exec.Cmd`
	cmdBits := strings.Split(cmdTxt, " ")
	executableCmd := exec.Command(cmdBits[0], cmdBits[1:]...) // nolint: gosec

	// get ready to capture the output
	cmdOut, err := executableCmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmdErr, err := executableCmd.StderrPipe()
	if err != nil {
		return err
	}
	var outBuf strings.Builder
	var errBuf strings.Builder

	// kick off the command
	err = executableCmd.Start()
	if err != nil {
		return err
	}

	// capture all the output
	_, err = io.Copy(&outBuf, cmdOut)
	if err != nil {
		return err
	}
	_, err = io.Copy(&errBuf, cmdErr)
	if err != nil {
		return err
	}

	matchedNonZeroExitCode := false
	err = executableCmd.Wait()
	if err != nil {
		ee, isEE := err.(*exec.ExitError)
		if !isEE {
			return err
		}

		if ee.ExitCode() != test.exitCode {
			return fmt.Errorf(
				"expected exit code %d, got %d (cmd err = %s)",
				test.exitCode,
				ee.ExitCode(),
				err.Error(),
			)
		}
		matchedNonZeroExitCode = true
	}
	if test.exitCode != 0 && !matchedNonZeroExitCode {
		return fmt.Errorf(
			"expected exit code %d, got 0",
			test.exitCode,
		)
	}

	outTxt := outBuf.String()
	errTxt := errBuf.String()

	var err1, err2 error
	if len(test.stdoutRE) > 0 {
		matched, err := regexp.Match(test.stdoutRE, []byte(outTxt))
		if err != nil {
			return err
		}
		if !matched {
			err1 = fmt.Errorf(
				"/%s/ failed to match stdout.\nSTDOUT:\n%s\n",
				test.stdoutRE,
				outTxt,
			)
		}
	}
	if len(test.stderrRE) > 0 {
		matched, err := regexp.Match(test.stderrRE, []byte(errTxt))
		if err != nil {
			return err
		}
		if !matched {
			err2 = fmt.Errorf(
				"/%s/ failed to match stderr.\nSTDERR:\n%s\n",
				test.stderrRE,
				errTxt,
			)
		}
	}
	if err1 != nil && err2 != nil {
		return fmt.Errorf("%s\n%s", err1, err2)
	}
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}
