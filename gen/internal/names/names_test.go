// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package names

import (
	"testing"
)

func TestPgToGoName(t *testing.T) {
	type testCase struct {
		src      string
		expected string
	}

	cases := []testCase{
		{
			src:      "foo_bar",
			expected: "FooBar",
		},
		{
			src:      "foo",
			expected: "Foo",
		},
		{
			src:      "fooBar",
			expected: "FooBar",
		},
		{
			src:      "foo Bar",
			expected: "FooBar",
		},
		{
			src:      "foo?!#_bar",
			expected: "FooBar",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.src, func(t *testing.T) {
			actual := PgToGoName(c.src)
			if actual != c.expected {
				t.Fatalf("case %d: expected '%s', got '%s'", i, c.expected, actual)
			}
		})
	}
}

func TestPgTableToGoModel(t *testing.T) {
	type testCase struct {
		src      string
		expected string
	}

	cases := []testCase{
		{
			src:      "foos",
			expected: "Foo",
		},
		{
			src:      "foo.bars",
			expected: "Foo_Bar",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.src, func(t *testing.T) {
			actual := PgTableToGoModel(c.src)
			if actual != c.expected {
				t.Fatalf("%d: expected '%s', got '%s'", i, c.expected, actual)
			}
		})
	}
}

func TestGoTypeNameToPascal(t *testing.T) {
	type testCase struct {
		src string
		expected string
	}

	cases := []testCase{
		{
			src: "int64",
			expected: "Int64",
		},
		{
			src: "doesntmatter.int64",
			expected: "Int64",
		},
		{
			src: "doesntmatter.myType",
			expected: "MyType",
		},
		{
			src: "myType",
			expected: "MyType",
		},
		{ // this one is weird, but I think this is a somewhat reasonable thing to do
			src: "wut.do.myType",
			expected: "MyType",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.src, func(t *testing.T) {
			actual := GoTypeNameToPascal(c.src)
			if actual != c.expected {
				t.Fatalf("expected '%s', got '%s'", c.expected, actual)
			}
		})
	}
}
