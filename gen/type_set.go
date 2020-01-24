package gen

import (
	"fmt"
	"io"
)

// typeSet is a set of types that pggen wants to emit. Each type
// in the set consists of a 3-tuple of (<type name>, <type sig>, <type body>).
// Identity is determined by the tuple of (<type name>, <type sig>), and
// <type body> contains the actual definition of the type that will be emitted.
//
// Keeping separate signatures and bodies allows us to decide that some mostly-the-same
// types are good enough to count as the same thing. In particular, tables with child
// entities have some extra slices attached to them for the child entities, but we still
// want people to be able to return the table's generated type from custom queries.
type typeSet struct {
	// A set mapping return type names to type definitions and signatures.
	set map[string]typeDecl
}

type typeDecl struct {
	// A string which uniquely identifies this type
	sig string
	// The body of this type
	body string
}

func newTypeSet() typeSet {
	return typeSet{
		set: map[string]typeDecl{},
	}
}

// Probe the type set for a specific type defintion. Can allow us to skip
// some work for types generated purely off of database objects such as
// user-defined postgres types.
func (t *typeSet) probe(name string) bool {
	_, inSet := t.set[name]
	return inSet
}

func (t *typeSet) emitType(name string, sig string, body string) error {
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

func (t *typeSet) gen(into io.Writer) error {
	for _, decl := range t.set {
		err := writeCompletely(into, []byte(decl.body))
		if err != nil {
			return err
		}
	}

	return nil
}
