package meta

import (
	"testing"
)

func TestConfigCommentToGoComment(t *testing.T) {
	type testCase struct {
		in  string
		out string
	}

	cases := []testCase{
		{
			in:  "",
			out: "",
		},
		{
			in:  "foo",
			out: "// foo",
		},
		{
			in: `
			foo
			 bar
			`,
			out: `// foo
//  bar`,
		},
		{
			in: `foo
 bar`,
			out: `// foo
//  bar`,
		},
		{
			in: `
this comment has

some blank lines in it
`,
			out: `// this comment has
//
// some blank lines in it`,
		},
	}

	for i, c := range cases {
		actual := configCommentToGoComment(c.in)
		if actual != c.out {
			t.Fatalf("%d: expected '%s', got '%s'", i, c.out, actual)
		}
	}
}
