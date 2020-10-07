package test

import (
	"testing"

	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/cmd/pggen/test/models"
)

// file: otherschema_test.go
// This file contains tests to ensure that `pggen` behaves reasonably when pointed at
// objects not in the 'public' schema.

// TestTestOtherschemaFoos is a smoke test for making sure that a table in a
// non-public schema works reasonably.
func TestOtherschemaFoos(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	// insert
	id, err := txClient.InsertOtherschema_Foo(ctx, &models.Otherschema_Foo{
		Value:  "a value",
		MyEnum: models.Otherschema_EnumTypeOpt3,
	})
	chkErr(t, err)

	// get (and list transitivly as well)
	foo, err := txClient.GetOtherschema_Foo(ctx, id)
	chkErr(t, err)
	if foo.Value != "a value" {
		t.Fatal("expected a value")
	}
	if foo.MyEnum.String() != "opt3" {
		t.Fatal("expected opt3")
	}

	// update
	_, err = txClient.UpdateOtherschema_Foo(ctx, &models.Otherschema_Foo{
		Id:    id,
		Value: "blah",
	}, models.Otherschema_FooAllFields)
	chkErr(t, err)
	foo, err = txClient.GetOtherschema_Foo(ctx, id)
	chkErr(t, err)
	if foo.Value != "blah" {
		t.Fatal("expected blah (1)")
	}

	// query
	foos, err := txClient.GetAllOtherschemaFoos(ctx)
	chkErr(t, err)
	if len(foos) != 1 {
		t.Fatal("expected 1 result")
	}
	if foos[0].Value != "blah" {
		t.Fatal("expected blah (2)")
	}

	// stmt
	_, err = txClient.ClobberAllOtherschemaFooValues(ctx)
	chkErr(t, err)
	foo, err = txClient.GetOtherschema_Foo(ctx, id)
	chkErr(t, err)
	if foo.Value != "" {
		t.Fatal("expected blank")
	}

	// delete
	err = txClient.DeleteOtherschema_Foo(ctx, id)
	chkErr(t, err)
	_, err = txClient.GetOtherschema_Foo(ctx, id)
	if !pggen.IsNotFoundError(err) {
		t.Fatal("expected not found")
	}
}

// this tests association support for associations between tables within the same schema
func TestOtherschemaInternalAssociations(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	parentID, err := txClient.InsertOtherschema_Parent(ctx, &models.Otherschema_Parent{
		Value: "parent",
	})
	chkErr(t, err)

	childID, err := txClient.InsertOtherschema_Child(ctx, &models.Otherschema_Child{
		Value:    "child",
		ParentId: parentID,
	})
	chkErr(t, err)

	unconstrainedChildID, err := txClient.InsertOtherschema_UnconstrainedChild(ctx, &models.Otherschema_UnconstrainedChild{
		Value:    "unconstrained_child",
		ParentId: parentID,
	})
	chkErr(t, err)

	// fill from parent to child
	parent, err := txClient.GetOtherschema_Parent(ctx, parentID)
	chkErr(t, err)
	err = txClient.Otherschema_ParentFillIncludes(ctx, parent, models.Otherschema_ParentAllIncludes)
	chkErr(t, err)
	if parent.Otherschema_Children[0].Value != "child" {
		t.Fatal("expected child")
	}
	if parent.Otherschema_UnconstrainedChild.Value != "unconstrained_child" {
		t.Fatal("expected unconstrained_child")
	}

	// fill in the child
	child, err := txClient.GetOtherschema_Child(ctx, childID)
	chkErr(t, err)
	err = txClient.Otherschema_ChildFillIncludes(ctx, child, models.Otherschema_ChildAllIncludes)
	chkErr(t, err)
	if child.Otherschema_Parent.Value != "parent" {
		t.Fatal("expected parent (child)")
	}

	// fill in the unconstrained_child
	unconstrainedChild, err := txClient.GetOtherschema_UnconstrainedChild(ctx, unconstrainedChildID)
	chkErr(t, err)
	err = txClient.Otherschema_UnconstrainedChildFillIncludes(ctx, unconstrainedChild, models.Otherschema_UnconstrainedChildAllIncludes)
	chkErr(t, err)
	if unconstrainedChild.Otherschema_Parent.Value != "parent" {
		t.Fatal("expected parent (unconstrained_child)")
	}
}

func TestOtherschemaCrossSchemaAssociations(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	// child is in non-public schema
	smallEntityID, err := txClient.InsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 9,
	})
	chkErr(t, err)
	childID, err := txClient.InsertOtherschema_SmallEntityChild(ctx, &models.Otherschema_SmallEntityChild{
		Value:         "small_entity_child",
		SmallEntityId: smallEntityID,
	})
	chkErr(t, err)

	child, err := txClient.GetOtherschema_SmallEntityChild(ctx, childID)
	chkErr(t, err)
	err = txClient.Otherschema_SmallEntityChildFillIncludes(ctx, child, models.Otherschema_SmallEntityChildAllIncludes)
	chkErr(t, err)
	if child.SmallEntity.Anint != 9 {
		t.Fatal("expected 9")
	}

	// child is in public schema
	parentID, err := txClient.InsertOtherschema_Parent(ctx, &models.Otherschema_Parent{
		Value: "parent",
	})
	chkErr(t, err)
	childOfOtherschemaID, err := txClient.InsertChildrenOfOtherschema(ctx, &models.ChildrenOfOtherschema{
		Value:               "children_of_otherschema",
		OtherschemaParentId: parentID,
	})
	chkErr(t, err)

	childOfOtherschema, err := txClient.GetChildrenOfOtherschema(ctx, childOfOtherschemaID)
	chkErr(t, err)
	err = txClient.ChildrenOfOtherschemaFillIncludes(ctx, childOfOtherschema, models.ChildrenOfOtherschemaAllIncludes)
	chkErr(t, err)
	if childOfOtherschema.Otherschema_Parent.Value != "parent" {
		t.Fatal("expected parent")
	}
}

func TestOtherschemaFunkyName(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	// insert
	id, err := txClient.InsertOtherschema_Funkyname(ctx, &models.Otherschema_Funkyname{
		Value: "a value",
	})
	chkErr(t, err)

	// get (and list transitivly as well)
	funky, err := txClient.GetOtherschema_Funkyname(ctx, id)
	chkErr(t, err)
	if funky.Value != "a value" {
		t.Fatal("expected a value")
	}
}
