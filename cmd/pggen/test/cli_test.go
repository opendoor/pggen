// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
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
	// The name of the test case.
	name string
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
		name:     "HelpText1",
		cmd:      "{{ .Exe }} -h",
		exitCode: 0,
		stdoutRE: "(?s)Usage:.*Args:.*Options:",
	},
	{
		name:     "HelpText2",
		cmd:      "{{ .Exe }} --help",
		exitCode: 0,
		stdoutRE: "(?s)Usage:.*Args:.*Options:",
	},

	// malformed args
	{
		name:     "BadArg1",
		cmd:      "{{ .Exe }}",
		exitCode: 1,
		stderrRE: "(?s)Usage:.*Args:.*Options:",
	},
	{
		name:     "BadArg2",
		cmd:      "{{ .Exe }} --bad-arg {{ .Toml }}",
		exitCode: 1,
		stderrRE: "(?s)Usage:.*Args:.*Options:",
	},

	// specific error messages
	{
		name: "NullFlagsAndReturnTypeError",
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
		name: "MalformedToml",
		toml: `
[[query]]
    name "returns_text"
	return_type = "Bad"
		`,
		exitCode: 1,
		stderrRE: "while parsing config file",
	},
	{
		name: "MissingTable",
		toml: `
[[table]]
    name = "dne"
		`,
		exitCode: 1,
		stderrRE: "could not find table 'dne' in the database",
	},
	{
		name: "UnknownConfigKey",
		toml: `
[[unknown_key]]
    also_unknwon = "dne"
		`,
		exitCode: 0,
		stderrRE: "WARN: unknown config file key: 'unknown_key'",
	},
	{
		name: "BadTypeOverride",
		toml: `
[[type_override]]
	type_name = "bogus"
		`,
		exitCode: 1,
		stderrRE: "type overrides must include a postgres type",
	},
	{
		// ok to omit the package when the go type is a primitive
		name: "TypeOverridePrimitive",
		toml: `
[[type_override]]
	postgres_type_name = "integer"
	type_name = "int"
		`,
	},
	{
		name: "TypeOverrideNonPrimitiveNoPkg",
		toml: `
[[type_override]]
	postgres_type_name = "foo_bar"
	type_name = "foo.Bar"
		`,
		exitCode: 1,
		stderrRE: "type override must include a package unless .*a primitive",
	},
	{
		name: "TypeOverrideNoOverride",
		toml: `
[[type_override]]
	postgres_type_name = "foo_bar"
		`,
		exitCode: 1,
		stderrRE: "type override must override the type or the nullable type",
	},
	{
		name: "TypeOverrideMissingNullName",
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
		name: "EnvVar",
		cmd:  "{{ .Exe }} -o {{ .Output }} -c $DB_URL {{ .Toml }}",
		toml: `
[[query]]
    name = "ConcatsText"
	body = "SELECT $1 || $2"
		`,
		exitCode: 0,
		stdoutRE: "ConcatsText",
	},
	{
		// test that we try the connection strings in order
		name: "ConnectionOrder",
		cmd:  "{{ .Exe }} -o {{ .Output }} -c bad -c $DB_URL {{ .Toml }}",
		toml: `
[[query]]
    name = "ConcatsText"
	body = "SELECT $1 || $2"
		`,
		exitCode: 0,
		stdoutRE: "ConcatsText",
	},
	{
		name: "AllBadConnections",
		cmd:  "{{ .Exe }} -o {{ .Output }} -c bad -c bader -c badest {{ .Toml }}",
		toml: `
[[query]]
    name = "concats_text"
		`,
		exitCode: 1,
		stderrRE: "unable to connect with any",
	},
	{
		name:     "DisableVar1",
		extraEnv: []string{"FOO=b"},
		cmd:      "{{ .Exe }} -o {{ .Output }} -d FOO {{ .Toml }}",
		toml: `
[[query]]
    name = "concats_text"
		`,
		exitCode: 0,
		stdoutRE: "doing nothing because a disable var matched",
	},
	{
		name:     "DisableVar2",
		extraEnv: []string{"FOO=b", "BLIP=baz"},
		cmd:      "{{ .Exe }} -o {{ .Output }} -d FOO --disable-var BLIP=baz {{ .Toml }}",
		toml: `
[[query]]
    name = "concats_text"
		`,
		exitCode: 0,
		stdoutRE: "doing nothing because a disable var matched",
	},
	{
		name:     "DisableVar3",
		extraEnv: []string{"FOO=b"},
		cmd:      "{{ .Exe }} -o {{ .Output }} -d FOO -c bad {{ .Toml }}",
		toml: `
[[query]]
    name = "concats_text"
		`,
		exitCode: 0,
		stdoutRE: "doing nothing because a disable var matched",
	},
	{
		name: "MissingCreatedAtField",
		toml: `
[[table]]
    name = "timestamps_both"
	created_at_field = "does_not_exist"
		`,
		exitCode: 0,
		stderrRE: "WARN.*no.*does_not_exist.*created at",
	},
	{
		name: "MissingUpdatedAtField",
		toml: `
[[table]]
    name = "timestamps_both"
	updated_at_field = "does_not_exist"
		`,
		exitCode: 0,
		stderrRE: "WARN.*no.*does_not_exist.*updated at",
	},
	{
		name:     "EnableVar1",
		extraEnv: []string{"FOO=b"},
		cmd:      "{{ .Exe }} -o {{ .Output }} -e FOO {{ .Toml }}",
		toml: `
[[query]]
    name = "ReturnsText"
	body = "SELECT 'foo'::text AS t"
		`,
		exitCode: 0,
		stdoutRE: "query 'ReturnsText'",
	},
	{
		name: "EnableVar2",
		cmd:  "{{ .Exe }} -o {{ .Output }} --enable-var UNSET=missing_value {{ .Toml }}",
		toml: `
[[query]]
    name = "returns_text"
		`,
		exitCode: 0,
		stdoutRE: "pggen: doing nothing because an enable var failed to match",
	},
	{
		name: "EmptyQueryBody",
		toml: `
[[query]]
    name = "EmptyQueryBody"
	body = ""
		`,
		exitCode: 1,
		stderrRE: "generating query 'EmptyQueryBody': empty query body",
	},
	{
		name: "MissingDeletedAt",
		toml: `
[[table]]
	name = "small_entities"
	deleted_at_field = "deleted_at"
		`,
		exitCode: 0,
		stderrRE: "WARN: table 'small_entities' has no nullable 'deleted_at' deleted at timestamp",
	},
	{
		name: "UnmatchedQuote",
		toml: `
[[table]]
	name = 'badschema."name'
		`,
		exitCode: 1,
		stderrRE: `parsing 'badschema."name': unmatched quote`,
	},
	{
		name: "MissingComment",
		toml: `
require_query_comments = true
[[query]]
	name = 'SomeQuery'
	body = "SELECT * FROM small_entities"
		`,
		exitCode: 1,
		stderrRE: `query 'SomeQuery' is missing a comment but require_query_comments is set`,
	},
	{
		name: "BadJsonColType",
		toml: `
[[table]]
	name = 'small_entities'
	[[table.json_type]]
		column_name = 'anint'
		type_name = 'DoesntMatter'
		pkg = '"github.com/doesnt/matter"'
		`,
		exitCode: 1,
		stderrRE: `cannot have a json type`,
	},
	{
		name: "BadImportPath",
		toml: `
[[type_override]]
	pkg = "github.com/opendoor-labs/pggen/examples/query/models" # note lack of quotes
		`,
		exitCode: 1,
		stderrRE: `import paths without spaces in them should be quoted strings`,
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

	for i := range cliCases {
		t.Run(cliCases[i].name, func(t *testing.T) {
			err = runCLITest(i, exe, testDir, &cliCases[i])
			if err != nil {
				t.Fatalf("While running cli test case %d:\n%s\n", i, err)
			}
		})
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
	err = ioutil.WriteFile(tomlFile, []byte(test.toml), 0600)
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
		envTxt := strings.Join(test.extraEnv, " ")
		if err != nil {
			err = fmt.Errorf("CMD: %s %s\n%s", envTxt, cmdTxt, err.Error())
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

		// can't use ee.ExitCode() because it is from go 1.12 and our msgv is 1.11
		if ee.String() != fmt.Sprintf("exit status %d", test.exitCode) {
			return fmt.Errorf(
				"expected exit code %d, got %s (cmd err = %s)",
				test.exitCode,
				ee.String(),
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
