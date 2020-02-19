package gen

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

	for i, c := range cases {
		actual := pgToGoName(c.src)
		if actual != c.expected {
			t.Fatalf("case %d: expected '%s', got '%s'", i, c.expected, actual)
		}
	}
}
