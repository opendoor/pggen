// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package pggen

import (
	"github.com/opendoor-labs/pggen/unstable"
)

// file: errors.go
// This file defines error utility functions used by pggen.

// IsNotFoundError returns true if the given error, or any of its causes
// is a not found error returned from pggen. The causal chain is determined
// by repeatedly calling Unwrap on the error.
func IsNotFoundError(err error) bool {
	for {
		if err == nil {
			return false
		}

		_, is := err.(*unstable.NotFoundError)
		if is {
			return is
		}

		// we don't use errors.Unwrap in order to maintain our msgv
		u, ok := err.(interface {
			Unwrap() error
		})
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
}
