package include

import (
	"regexp"
	"testing"
)

func TestParseSuccess(t *testing.T) {
	type testCase struct {
		src    string
		result string
	}

	cases := []testCase{
		{
			src: "foos",
		},
		{
			src: "f234oos",
		},
		{
			src:    "  f_23",
			result: "f_23",
		},
		{
			src:    "FooBar  ",
			result: "FooBar",
		},
		{
			src:    "   foos  ",
			result: "foos",
		},
		{
			src: "foos.bars",
		},
		{
			src:    "foos .bars",
			result: "foos.bars",
		},
		{
			src:    "foos. bars",
			result: "foos.bars",
		},
		{
			src:    "foos . bars",
			result: "foos.bars",
		},
		{
			src:    "foos.{bars}",
			result: "foos.bars",
		},
		{
			src: "foos.{bars,bim}",
		},
		{
			src:    "foos.{bars,}",
			result: "foos.bars",
		},
		{
			src:    "foos.{bars,bim,}",
			result: "foos.{bars,bim}",
		},
		{
			src: "foos.{bars.blip,bim}",
		},
		{
			src: "foos.{bars.blip.flip.dip,bim.{a,b,c.{d,e}}}",
		},
		{
			src:    "  foos.{bars .blip. flip.dip ,bim.{a, b   ,c.{d   ,    e}}}    ",
			result: "foos.{bars.blip.flip.dip,bim.{a,b,c.{d,e}}}",
		},
		{
			// basic rename expression
			src: `f.a->b`,
		},
		{
			// normalize rename expressions
			src:    "f.a->a",
			result: "f.a",
		},
		{
			// longer names and spaces
			src:    " f . longer -> names_can_be_renamed_as_well ",
			result: "f.longer->names_can_be_renamed_as_well",
		},
		{
			// rename expression w/subspec
			src: "f.n1->n2.baz",
		},
		// funny names
		{
			src: "f$",
		},
		{
			src: "_f",
		},
		{
			src:    `"foo"`,
			result: `foo`,
		},
		{
			src: `"123 _f"`,
		},
		{
			src: `"123 "" _f"`,
		},
		{
			src: `"a.b".c`,
		},
	}

	for i, c := range cases {
		s, err := Parse(c.src)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %s", i, err)
		}

		if len(c.result) == 0 {
			c.result = c.src
		}

		after := s.String()
		if after != c.result {
			t.Fatalf(
				"case %d: expected '%s' to become '%s', got '%s'",
				i,
				c.src,
				c.result,
				after,
			)
		}
	}
}

func TestParseErrors(t *testing.T) {
	type testCase struct {
		src string
		re  string
	}

	cases := []testCase{
		{
			src: "=foos",
			re:  "'=' cannot begin",
		},
		{
			src: "",
			re:  "expected an identifier to start a spec",
		},
		{
			src: "foo bar",
			re:  "unexpected extra token begining with 'b'",
		},
		{
			src: "foo.",
			re:  "expected spec or list of specs after '.'",
		},
		{
			src: "foo.{",
			re:  "unexpected end of input while parsing spec list",
		},
		{
			src: "foo.{bar",
			re:  "unexpected end of input while parsing spec list",
		},
		{
			src: "foo . { bar",
			re:  "unexpected end of input while parsing spec list",
		},
		{
			src: "foo.{ bar baz",
			re:  "expected ',' to separate sub specs",
		},
		{
			src: "foos.{}",
			re:  "empty spec list",
		},
		{
			src: `"blah balhjl`,
			re:  "unexpected end of input in quoted identifier",
		},
		{
			src: `top_level->rename.bad`,
			re:  "unexpected extra token begining with '-'",
		},
	}

	for i, c := range cases {
		s, err := Parse(c.src)
		if err == nil {
			t.Fatalf("case %d: err == nil", i)
		}
		if s != nil {
			t.Fatalf("case %d: s != nil", i)
		}

		errTxt := err.Error()
		matches, err := regexp.Match(c.re, []byte(errTxt))
		if err != nil {
			t.Fatal(err)
		}
		if !matches {
			t.Fatalf("case %d: /%s/ failed to match '%s'", i, c.re, errTxt)
		}
	}
}

func TestCyclicIncludeSpec(t *testing.T) {
	cyclic := Spec{
		TableName: "foo",
	}
	cyclic.Includes = map[string]*Spec{
		"bar": {
			TableName: "bar",
			Includes: map[string]*Spec{
				"foo": &cyclic,
			},
		},
	}

	containsCyclic := Spec{
		TableName: "baz",
		Includes: map[string]*Spec{
			"foo": &cyclic,
		},
	}

	txt := cyclic.String()
	if txt != "foo.bar.foo" {
		t.Fatalf("bad txt: %s", txt)
	}

	txt = containsCyclic.String()
	if txt != "baz.foo.bar.foo" {
		t.Fatalf("bad txt: %s", txt)
	}
}
