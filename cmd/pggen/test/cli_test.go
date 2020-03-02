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
	// A list of VAR=value environment variables to inject into the environment
	// the command executes in.
	extraEnv []string
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
	{
		// missing table
		toml: `
[[table]]
    name = "dne"
		`,
		exitCode: 1,
		stderrRE: "could not find table 'dne' in the database",
	},
	{
		toml: `
[[unknown_key]]
    also_unknwon = "dne"
		`,
		exitCode: 0,
		stderrRE: "WARN: unknown config file key: 'unknown_key'",
	},
	{
		toml: `
[[type_override]]
	type_name = "bogus"
		`,
		exitCode: 1,
		stderrRE: "type overrides must include a postgres type",
	},
	{
		// ok to omit the package when the go type is a primitive
		toml: `
[[type_override]]
	postgres_type_name = "integer"
	type_name = "int"
		`,
	},
	{
		toml: `
[[type_override]]
	postgres_type_name = "foo_bar"
	type_name = "foo.Bar"
		`,
		exitCode: 1,
		stderrRE: "type override must include a package unless .*a primitive",
	},
	{
		toml: `
[[type_override]]
	postgres_type_name = "foo_bar"
		`,
		exitCode: 1,
		stderrRE: "type override must override the type or the nullable type",
	},
	{
		toml: `
[[type_override]]
	postgres_type_name = "foo_bar"
	pkg = "\"fake.com/foo\""
	type_name = "foo.Bar"
		`,
		exitCode: 1,
		stderrRE: "`type_name` and `nullable_type_name` must both be provided",
	},
	{
		// test that we expand $ENV_VAR connection strings
		cmd: "{{ .Exe }} -o {{ .Output }} -c $DB_URL {{ .Toml }}",
		toml: `
[[stored_function]]
    name = "concats_text"
		`,
		exitCode: 0,
		stdoutRE: "concats_text",
	},
	{
		// test that we try the connection strings in order
		cmd: "{{ .Exe }} -o {{ .Output }} -c bad -c $DB_URL {{ .Toml }}",
		toml: `
[[stored_function]]
    name = "concats_text"
		`,
		exitCode: 0,
		stdoutRE: "concats_text",
	},
	{
		cmd: "{{ .Exe }} -o {{ .Output }} -c bad -c bader -c badest {{ .Toml }}",
		toml: `
[[stored_function]]
    name = "concats_text"
		`,
		exitCode: 1,
		stderrRE: "unable to connect with any",
	},
	{
		extraEnv: []string{"FOO=b"},
		cmd:      "{{ .Exe }} -o {{ .Output }} -d FOO {{ .Toml }}",
		toml: `
[[stored_function]]
    name = "concats_text"
		`,
		exitCode: 0,
		stdoutRE: "doing nothing because a disable var matched",
	},
	{
		extraEnv: []string{"FOO=b", "BLIP=baz"},
		cmd:      "{{ .Exe }} -o {{ .Output }} -d FOO --disable-var BLIP=baz {{ .Toml }}",
		toml: `
[[stored_function]]
    name = "concats_text"
		`,
		exitCode: 0,
		stdoutRE: "doing nothing because a disable var matched",
	},
	{
		extraEnv: []string{"FOO=b"},
		cmd:      "{{ .Exe }} -o {{ .Output }} -d FOO -c bad {{ .Toml }}",
		toml: `
[[stored_function]]
    name = "concats_text"
		`,
		exitCode: 0,
		stdoutRE: "doing nothing because a disable var matched",
	},
}

func TestCLI(t *testing.T) {
	debug := false
	debugEnvVar := os.Getenv("PGGEN_DEBUG_CLI")
	if debugEnvVar == "1" || debugEnvVar == "true" {
		debug = true
	}

	testDir, err := ioutil.TempDir("", "pggen_cli_test")
	chkErr(t, err)
	if !debug {
		defer os.RemoveAll(testDir)
	}

	repoRoot, err := getRepoRoot()
	chkErr(t, err)

	exe := path.Join(testDir, "pggen")
	mainSrc := path.Join(repoRoot, "cmd", "pggen", "main.go")

	// build the executable we are going to be testing
	cmd := exec.Command("go", "build", "-o", exe, mainSrc)
	if debug {
		cmd = exec.Command(
			"go", "build", "-gcflags", "all=-N -l", "-o", exe, mainSrc)
	}
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

	// set up the environment
	for _, setting := range test.extraEnv {
		eqIdx := strings.Index(setting, "=")
		if eqIdx == -1 {
			return fmt.Errorf("expected = in env setting")
		}

		key := setting[:eqIdx]
		os.Setenv(key, setting[eqIdx+1:])
		defer func() {
			os.Setenv(key, "") // close enough for government work
		}()
	}

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

	errs := []error{}

	// check the exit code
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
		errs = append(errs, fmt.Errorf("expecting non-zero exit code"))
	}

	// execute regex assertions on the output
	outTxt := outBuf.String()
	errTxt := errBuf.String()
	if len(test.stdoutRE) > 0 {
		matched, err := regexp.Match(test.stdoutRE, []byte(outTxt))
		if err != nil {
			return err
		}
		if !matched {
			errs = append(errs, fmt.Errorf(
				"/%s/ failed to match stdout.\nSTDOUT:\n%s\n",
				test.stdoutRE,
				outTxt,
			))
		}
	}
	if len(test.stderrRE) > 0 {
		matched, err := regexp.Match(test.stderrRE, []byte(errTxt))
		if err != nil {
			return err
		}
		if !matched {
			errs = append(errs, fmt.Errorf(
				"/%s/ failed to match stderr.\nSTDERR:\n%s\n",
				test.stderrRE,
				errTxt,
			))
		}
	}

	if len(errs) > 0 {
		var errTxt strings.Builder

		for _, e := range errs {
			errTxt.WriteString(e.Error())
			errTxt.WriteByte('\n')
		}

		return fmt.Errorf(errTxt.String())
	}

	return nil
}
