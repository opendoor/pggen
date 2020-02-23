package gen

import (
	"fmt"
	"strings"
)

//
// Parsing routines for postgres types.
//

type pgType interface {
	fmt.Stringer
}

type pgPrimType struct {
	name string
}

func (pp *pgPrimType) String() string {
	return pp.name
}

type pgArrayType struct {
	// The type that the array contains
	inner pgType
}

func (pa *pgArrayType) String() string {
	var out strings.Builder

	out.WriteString(pa.inner.String())
	out.WriteString("[]")

	return out.String()
}

//
// Parsers
//

func parsePgArray(src string) (array *pgArrayType, err error) {
	// The full type grammar for postgres is rather long and complicated,
	// so we are going to punt on parsing it completely. Doing it properly
	// likely requires pulling in the actual postgres parser and hooking
	// in with cgo.

	nestLevel := 0

	for strings.HasSuffix(src, "[]") {
		src = src[:len(src)-2]
		nestLevel++
	}

	if nestLevel == 0 {
		return nil, fmt.Errorf("tried to parse an array, but failed to")
	}

	var ty pgType = &pgPrimType{name: src}
	for i := 0; i < nestLevel; i++ {
		ty = &pgArrayType{inner: ty}
	}

	return ty.(*pgArrayType), nil
}
