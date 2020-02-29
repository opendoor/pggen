package gen

import (
	"reflect"
	"regexp"
	"testing"
)

func TestParseCmdLine(t *testing.T) {
	type testCase struct {
		input    string
		expected []string
		errRE    string
	}

	cases := []testCase{
		{
			input: `   `,
			errRE: "blank cmd string",
		},
		{
			input:    "foo bar",
			expected: []string{"foo", "bar"},
		},
		{
			input:    `"foo bar" baz`,
			expected: []string{"foo bar", "baz"},
		},
		{
			input:    `"foo  bar" baz`,
			expected: []string{"foo  bar", "baz"},
		},
		{
			input:    `"foo bar" baz "bli p al"`,
			expected: []string{"foo bar", "baz", "bli p al"},
		},
		{
			input: `"foo bar`,
			errRE: "unmatched quote char",
		},
		{
			input:    `"bli p 'al"`,
			expected: []string{"bli p 'al"},
		},
		{
			input:    `'foo bar' baz "bli p 'al"`,
			expected: []string{"foo bar", "baz", "bli p 'al"},
		},
		{
			input:    `'foo \' bar'`,
			expected: []string{"foo ' bar"},
		},
		{
			input:    `'foo \" bar'`,
			expected: []string{`foo \" bar`},
		},
		{
			input:    `\"blah "foo bar"`,
			expected: []string{`"blah`, `foo bar`},
		},
		{
			input:    `foo\ bar`,
			expected: []string{`foo bar`},
		},
		{
			input:    `foo \ bar`,
			expected: []string{`foo`, ` bar`},
		},
		{
			input:    `foo \ \ bar`,
			expected: []string{`foo`, `  bar`},
		},
		{
			input:    `foo " \ \ bar"`,
			expected: []string{`foo`, ` \ \ bar`},
		},
		{
			input:    `fo"o`,
			expected: []string{`fo"o`},
		},
		{
			input:    `fo"o, "baz"`,
			expected: []string{`fo"o,`, "baz"},
		},
	}

	for i, c := range cases {
		actual, err := parseCmdLine(c.input)

		if err != nil {
			if len(c.errRE) > 0 {
				matches, err := regexp.Match(c.errRE, []byte(err.Error()))
				chkErr(t, err)

				if !matches {
					t.Fatalf(
						"%d: expected err to match /%s/, actual:\n%s",
						i,
						c.errRE,
						err.Error(),
					)
				}
			} else {
				t.Fatalf("%d: non-nil err, but no errRE", i)
			}
		} else if len(c.errRE) > 0 {
			t.Fatalf("%d: nil err, but errRE", i)
		}

		if !reflect.DeepEqual(actual, c.expected) {
			t.Fatalf(
				"%d: actual = %#v, expected = %#v",
				i,
				actual,
				c.expected,
			)
		}
	}
}
