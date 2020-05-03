package utils

import (
	"testing"
)

func TestNullOutArgs(t *testing.T) {
	type testVec struct {
		input    string
		expected string
	}
	vecs := []testVec{
		{
			input:    "there is nothing to replace here",
			expected: "there is nothing to replace here",
		},
		{
			input:    "a null $1",
			expected: "a null NULL",
		},
		{
			input:    "some nulls $1 $2",
			expected: "some nulls NULL NULL",
		},
		{
			input:    "some adjacent nulls $1$2$3349 $23$12",
			expected: "some adjacent nulls NULLNULLNULL NULLNULL",
		},
		{
			input:    "quoted '$1' not quoted $2",
			expected: "quoted '$1' not quoted NULL",
		},
		{
			input:    "quoted adjacent '$1'$2",
			expected: "quoted adjacent '$1'NULL",
		},
		{
			input:    `fake quoted \'$1\'$2`,
			expected: `fake quoted \'NULL\'NULL`,
		},
		{
			input:    "$19",
			expected: "NULL",
		},
		{
			input:    `double quoted "$1" `,
			expected: `double quoted "$1" `,
		},
		{
			input:    `escaped quote terminator "$1\"$3" `,
			expected: `escaped quote terminator "$1\"$3" `,
		},
	}

	for _, v := range vecs {
		actual := NullOutArgs(v.input)
		if actual != v.expected {
			t.Errorf("\nExpected: %s\nActual: %s\n", v.expected, actual)
		}
	}
}
