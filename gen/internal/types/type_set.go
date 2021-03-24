// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package types

import (
	"fmt"
	"io"
	"sort"

	"github.com/opendoor-labs/pggen/gen/internal/utils"
)

// set is a set of types that pggen wants to emit. Each type
// in the set consists of a 3-tuple of (<type name>, <type sig>, <type body>).
// Identity is determined by the tuple of (<type name>, <type sig>), and
// <type body> contains the actual definition of the type that will be emitted.
//
// Keeping separate signatures and bodies allows us to decide that some mostly-the-same
// types are good enough to count as the same thing. In particular, tables with child
// entities have some extra slices attached to them for the child entities, but we still
// want people to be able to return the table's generated type from custom queries.
type set struct {
	// A set mapping return type names to type definitions and signatures.
	set map[string]typeDecl
}

type typeDecl struct {
	// A string which uniquely identifies this type
	sig string
	// The body of this type
	body string
}

func newSet() set {
	return set{
		set: map[string]typeDecl{},
	}
}

// Probe the type set for a specific type defintion. Can allow us to skip
// some work for types generated purely off of database objects such as
// user-defined postgres types.
func (t *set) probe(name string) bool {
	_, inSet := t.set[name]
	return inSet
}

func (t *set) emitType(name string, sig string, body string) error {
	existingDecl, inSet := t.set[name]
	if inSet {
		if existingDecl.sig != sig {
			return fmt.Errorf(
				`field mismatch for type '%s'.
one query has a return with fields
'''
%s
'''
but another has a return type with fields
'''
%s
'''`, name, existingDecl.sig, sig)
		}
	} else {
		t.set[name] = typeDecl{
			sig:  sig,
			body: body,
		}
	}

	return nil
}

func (t *set) gen(into io.Writer) error {
	decls := make([]typeDecl, len(t.set))
	for _, decl := range t.set {
		decls = append(decls, decl)
	}

	sort.Slice(decls, func(i, j int) bool {
		return decls[i].sig < decls[j].sig
	})

	for _, decl := range decls {
		err := utils.WriteCompletely(into, []byte(decl.body))
		if err != nil {
			return err
		}
	}

	return nil
}
