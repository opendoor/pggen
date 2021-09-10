package meta

import (
	"reflect"
	"regexp"
	"testing"
)

func TestMergeTags(t *testing.T) {
	type testCase struct {
		t1  string
		t2  string
		out string
	}

	cases := []testCase{
		{
			t1:  `foo:"bar"`,
			t2:  `bim:"baz"`,
			out: `foo:"bar" bim:"baz"`,
		},
		{
			t1:  `foo:"bar"`,
			t2:  `foo:"baz"`,
			out: `foo:"bar"`,
		},
		{
			t1:  `gorm:"bar"`,
			t2:  `gorm:"baz" gorm:"blip"`,
			out: `gorm:"bar;baz;blip"`,
		},
		{
			t1:  `gorm:"bar"`,
			t2:  `gorm:"baz" gorm:"`, // malformed
			out: `gorm:"bar" gorm:"baz" gorm:"`,
		},
		{
			t1:  `foo"bar"`, // malformed
			t2:  `bar:"baz"`,
			out: `bar:"baz" foo"bar"`,
		},
		{
			t1:  `foo"bar"`, // malformed
			t2:  `baraz"`,   // malformed
			out: `foo"bar" baraz"`,
		},
		{
			t1:  `foo:"bar"`,
			t2:  `slim:"j\"im"`,
			out: `foo:"bar" slim:"j\"im"`,
		},
		{
			t1:  `foo:"bar"`,
			t2:  ``,
			out: `foo:"bar"`,
		},
	}

	for i, c := range cases {
		actual := mergeTags(c.t1, c.t2)
		if actual != c.out {
			t.Fatalf("case %d: actual '%s' != expected '%s'", i, actual, c.out)
		}
	}
}

func TestParseTags(t *testing.T) {
	type testCase struct {
		input    string
		outPairs []tagPair
		outErrRE string
	}

	cases := []testCase{
		{
			input:    `foo:"bar"`,
			outPairs: []tagPair{{key: "foo", value: "bar"}},
		},
		{
			input:    `foo:"bar`,
			outErrRE: "unclosed quoted value",
		},
		{
			input:    `foo:"`,
			outErrRE: "unclosed quoted value",
		},
		{
			input:    `foo:`,
			outErrRE: "incomplete tag",
		},
		{
			input:    `foo`,
			outErrRE: "incomplete tag",
		},
		{
			input: `foo:"bar" bar:"ba\"z"  foo:"bop"`,
			outPairs: []tagPair{
				{key: "foo", value: "bar"},
				{key: "bar", value: "ba\"z"},
				{key: "foo", value: "bop"},
			},
		},
	}

	for i, c := range cases {
		pairs, err := parseTags(c.input)

		if c.outPairs != nil {
			if !reflect.DeepEqual(c.outPairs, pairs) {
				t.Fatalf("case %d: expected %v, actual %v", i, c.outPairs, pairs)
			}
		}

		if c.outErrRE != "" {
			if err == nil {
				t.Fatalf("case %d: expected err but there was none", i)
			}

			re := regexp.MustCompile(c.outErrRE)
			if !re.Match([]byte(err.Error())) {
				t.Fatalf("case %d: /%s/ failed to match: %s", i, c.outErrRE, err.Error())
			}
		}

		if c.outPairs == nil && c.outErrRE == "" {
			t.Fatalf("case %d: no output assertions", i)
		}
	}
}
