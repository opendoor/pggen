// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package meta

import (
	"reflect"
	"regexp"
	"testing"
)

func TestArgNamesToSlice(t *testing.T) {
	type testCase struct {
		// inputs
		argNamesSpec string
		targetNargs  int
		// outputs
		args []string
		err  string
	}

	cases := []testCase{
		{
			argNamesSpec: "1:foo",
			targetNargs:  1,
			args:         []string{"foo"},
		},
		{
			argNamesSpec: "2:bar 1:foo",
			targetNargs:  2,
			args:         []string{"foo", "bar"},
		},
		{
			argNamesSpec: "2:bar 1:foo",
			targetNargs:  1,
			err:          "2 out of range",
		},
		{
			argNamesSpec: "-1:foo",
			targetNargs:  1,
			err:          "start at 1 not -1",
		},
		{
			argNamesSpec: "1:foo",
			targetNargs:  2,
			args:         []string{"foo", "arg2"},
		},
		{
			argNamesSpec: "1:foo 3:baz",
			targetNargs:  3,
			args:         []string{"foo", "arg2", "baz"},
		},
		{
			argNamesSpec: "",
			targetNargs:  2,
			args:         []string{"arg1", "arg2"},
		},
	}

	for i, c := range cases {
		actualArgs, err := argNamesToSlice(c.argNamesSpec, c.targetNargs)

		if !reflect.DeepEqual(actualArgs, c.args) {
			t.Fatalf("%d: expected(%v) != actual(%v)", i, c.args, actualArgs)
		}

		if c.err == "" && err != nil {
			t.Fatalf("%d: got err when expecting none: %s", i, err.Error())
		}

		if c.err != "" {
			if err == nil {
				t.Fatalf("%d: got no err when expecting one to match /%s/", i, c.err)
			} else {
				errStr := []byte(err.Error())
				matched, regexErr := regexp.Match(c.err, errStr)
				if regexErr != nil {
					t.Fatalf("%d: bad pattern /%s/", i, c.err)
				}
				if !matched {
					t.Fatalf("%d: expected err '%s' to match pattern /%s/", i, string(errStr), c.err)
				}
			}
		}

	}
}
