package meta

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitType(t *testing.T) {
	type testVec struct {
		input       string
		expected    []string
		expectedErr string
	}

	testVecs := []testVec{
		{
			input:       "foo",
			expectedErr: "malformed data 'foo'",
		},
		{
			input:    "{foo}",
			expected: []string{"foo"},
		},
		{
			input:    `{"foo bar"}`,
			expected: []string{"foo bar"},
		},
		{
			input:    `{foo,bar}`,
			expected: []string{"foo", "bar"},
		},
		{
			input:       `{foo,}`,
			expectedErr: "trailing comma",
		},
		{
			input:    `{"string with \" quote"}`,
			expected: []string{`string with \" quote`},
		},
		{
			input:    `{"foo","bar"}`,
			expected: []string{"foo", "bar"},
		},
		{
			input:    `{foo,"bar"}`,
			expected: []string{"foo", "bar"},
		},
		{
			input:    `{"foo",bar}`,
			expected: []string{"foo", "bar"},
		},
		{
			input: `{"this one",is,a,"big chungus",with,many,"bits"}`,
			expected: []string{"this one", "is", "a", "big chungus",
				"with", "many", "bits"},
		},
	}

	for i, v := range testVecs {
		inputBytes := []byte(v.input)

		var actual RegTypeArray
		err := actual.Scan(inputBytes)
		if err != nil &&
			(!strings.Contains(err.Error(), v.expectedErr) ||
				len(v.expectedErr) == 0) {
			t.Errorf(
				"\n(case %d) Error: %s\n       Expected Error: %s\n",
				i,
				err.Error(),
				v.expectedErr,
			)
		}

		if !reflect.DeepEqual(actual.pgTypes, v.expected) {
			t.Errorf(
				"\n(case %d) Actual: %#v\n       Expected: %#v\n",
				i,
				actual.pgTypes,
				v.expected,
			)
		}
	}
}
