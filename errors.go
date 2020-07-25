package pggen

import (
	"github.com/opendoor-labs/pggen/unstable"
)

// file: errors.go
// This file defines error utility functions used by pggen.

func IsNotFoundError(err error) bool {
	_, is := err.(*unstable.NotFoundError)
	return is
}
