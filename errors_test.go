package pggen

import (
	"fmt"
	"testing"

	"github.com/opendoor/pggen/unstable"
)

func TestIsNotFoundError(t *testing.T) {
	type testCase struct {
		err error
		is  bool
	}
	cases := []testCase{
		{
			err: fmt.Errorf("NonNotFound1"),
			is:  false,
		},
		{
			err: &unstable.NotFoundError{Msg: "NotFound1"},
			is:  true,
		},
		{
			err: &causedErr{cause: &unstable.NotFoundError{Msg: "NotFound2"}},
			is:  true,
		},
		{
			err: &causedErr{cause: fmt.Errorf("NonNotFound2")},
			is:  false,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.err.Error(), func(t *testing.T) {
			if IsNotFoundError(c.err) != c.is {
				t.Fatalf("expected %t, got %t", c.is, !c.is)
			}
		})
	}
}

// we define this manually rather than using %w to maintain our msgv
type causedErr struct {
	cause error
}

func (ce *causedErr) Error() string {
	return ce.cause.Error()
}
func (ce *causedErr) Unwrap() error {
	return ce.cause
}
