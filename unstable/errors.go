// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package unstable

// DO NOT USE. Use pggen.IsNotFoundError instead of directly referencing this type.
type NotFoundError struct {
	Msg string
}

func (e *NotFoundError) Error() string {
	return e.Msg
}
