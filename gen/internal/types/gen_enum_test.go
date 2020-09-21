package types

import (
	"reflect"
	"testing"
)

func TestEnumValuesToGoNames(t *testing.T) {
	type testCase struct {
		in  []string
		out []string
	}
	cases := []testCase{
		{
			in:  []string{"foo", "foo_bar"},
			out: []string{"Foo", "FooBar"},
		},
		{
			in:  []string{"foo", "foo+"},
			out: []string{"Foo", "Foo1"},
		},
		{
			in:  []string{"bar___ blip@@foo+"},
			out: []string{"BarBlipfoo"},
		},
	}

	for _, c := range cases {
		actual := enumValuesToGoNames(c.in)
		if !reflect.DeepEqual(actual, c.out) {
			t.Fatalf("expected %v, got %v\n", c.out, actual)
		}
	}
}
