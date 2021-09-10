package names

import (
	"reflect"
	"regexp"
	"testing"
)

func TestPgNameRoundTrip(t *testing.T) {
	type testCase struct {
		input string
		// regex matching error string if non-blank
		err string
		// expected output if non-nil
		out *PgName
		// expected output of calling .String() if non-blank
		outString string
	}

	cases := []testCase{
		{
			input:     "foo",
			out:       &PgName{Schema: "public", Name: "foo"},
			outString: `foo`,
		},
		{
			input: `foo.bar.baz`,
			err:   ".*nested schemas are not supported.*",
		},
		{
			input:     `"foo"`,
			out:       &PgName{Schema: "public", Name: "foo"},
			outString: `foo`,
		},
		{
			input: `.foo`,
			err:   "empty identifier",
		},
		{
			input: `foo.`,
			err:   "empty identifier",
		},
		{
			input: `"".foo`,
			err:   "empty identifier",
		},
		{
			input: `foo.""`,
			err:   "empty identifier",
		},
		{
			input: `foo."`,
			err:   "unmatched quote",
		},
		{
			input: `foo."sdfas`,
			err:   "unmatched quote",
		},
		{
			input: `"foo.sdfas`,
			err:   "unmatched quote",
		},
		{
			input: `"foo"."sdfas`,
			err:   "unmatched quote",
		},
		{
			input: `foo"asdfadsf`,
			err:   "cannot begin quoting in the middle",
		},
		{
			input:     `"foo".bar`,
			out:       &PgName{Schema: "foo", Name: "bar"},
			outString: `foo.bar`,
		},
		{
			input:     `foo.bar`,
			out:       &PgName{Schema: "foo", Name: "bar"},
			outString: `foo.bar`,
		},
		{
			input: `foo."b"ar"`,
			err:   "unmatched quote",
		},
		{
			input:     `foo."b""ar"`,
			out:       &PgName{Schema: "foo", Name: `b"ar`},
			outString: `foo."b""ar"`,
		},
		{
			input:     `foo."b""a""r"`,
			out:       &PgName{Schema: "foo", Name: `b"a"r`},
			outString: `foo."b""a""r"`,
		},
		{
			input:     `foo."b ar"`,
			out:       &PgName{Schema: "foo", Name: `b ar`},
			outString: `foo."b ar"`,
		},
		{
			// Can't have escaped quotes in an unquoted identifier. This is not the best error message,
			// but it is good enough for government work.
			input: `f""oo.bar`,
			err:   "cannot begin quoting in the middle",
		},
		{
			input: `foo.bar.baz`,
			err:   "nested schemas are not supported",
		},
	}

	for i, c := range cases {
		res, err := ParsePgName(c.input)

		if c.err != "" {
			if err == nil {
				t.Fatalf("%d: expected error but there was none", i)
			} else {
				matches, reErr := regexp.Match(c.err, []byte(err.Error()))
				if reErr != nil {
					t.Fatalf("%d: bad regex: /%s/: %s", i, c.err, reErr.Error())
				}
				if !matches {
					t.Fatalf("%d: /%s/ fails to match err '%s'", i, c.err, err.Error())
				}
			}
		} else if err != nil {
			t.Fatalf("%d: unexpected error: %s", i, err.Error())
		}

		if c.out != nil {
			if !reflect.DeepEqual(c.out, &res) {
				t.Fatalf("%d: cmp: actual = %#v, expected = %#v", i, res, c.out)
			}
		}

		if c.outString != "" {
			roundTripTxt := res.String()
			if roundTripTxt != c.outString {
				t.Fatalf("%d: rt: actual = '%s', expected = '%s'", i, roundTripTxt, c.outString)
			}
		}
	}
}
