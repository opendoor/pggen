package names

import (
	"regexp"
	"testing"
)

func TestValidateImportPath(t *testing.T) {
	type testCase struct {
		in    string
		errRE string
	}

	cases := []testCase{
		{
			in: `"foo"`,
		},
		{
			in: `blip "foo"`,
		},
		{
			in:    `"foo`,
			errRE: "import paths without spaces in them should be quoted strings",
		},
		{
			in:    `foo`,
			errRE: "import paths without spaces in them should be quoted strings",
		},
		{
			in:    ``,
			errRE: "import paths without spaces in them should be quoted strings",
		},
		{
			in:    `foo"`,
			errRE: "import paths without spaces in them should be quoted strings",
		},
		{
			in:    `"foo" "bar"`,
			errRE: "import paths containing spaces should be aliased quoted strings",
		},
		{
			in:    `9oo "bar"`,
			errRE: "import paths containing spaces should be aliased quoted strings",
		},
		{
			in:    `oo bar"`,
			errRE: "import paths containing spaces should be aliased quoted strings",
		},
		{
			in:    `oo b"ar"`,
			errRE: "import paths containing spaces should be aliased quoted strings",
		},
	}

	for i, c := range cases {
		err := ValidateImportPath(c.in)
		if len(c.errRE) > 0 {
			if err == nil {
				t.Fatalf("%d: no error, but we were expecting one", i)
			}

			matches, reErr := regexp.Match(c.errRE, []byte(err.Error()))
			if reErr != nil {
				t.Fatalf("%d: regex err: %s", i, reErr.Error())
			}
			if !matches {
				t.Fatalf("%d: /%s/ failed to match error '%s'", i, c.errRE, err.Error())
			}
		} else {
			if err != nil {
				t.Fatalf("%d: unexpected err: %s", i, err.Error())
			}
		}
	}
}
