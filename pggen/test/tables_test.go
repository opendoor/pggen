package test

import (
	"database/sql"
	"reflect"
	"sort"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/willf/bitset"

	"github.com/opendoor-labs/pggen/pggen/test/db_shims"
)

func TestInsertSmallEntity(t *testing.T) {
	txClient := newTx(t)
	defer func() {
		_ = txClient.DB.(*sql.Tx).Rollback()
	}()

	entity := db_shims.SmallEntity{
		Anint: 129,
	}

	id, err := txClient.InsertSmallEntity(ctx, entity)
	chkErr(t, err)

	entity.Id = id

	fetched, err := txClient.GetSmallEntityByAnint(ctx, entity.Anint)
	chkErr(t, err)

	if !reflect.DeepEqual(entity, fetched[0]) {
		t.Fatalf("%#v != %#v", entity, fetched[0])
	}
}

func TestSmallEntityBulk(t *testing.T) {
	// tests BulkInsert and List

	txClient := newTx(t)
	defer func() {
		_ = txClient.DB.(*sql.Tx).Rollback()
	}()

	entities := []db_shims.SmallEntity{
		{
			Anint: 1232,
		},
		{
			Anint: 1232,
		},
		{
			Anint: 1232,
		},
	}

	_, err := txClient.BulkInsertSmallEntity(ctx, entities)
	chkErr(t, err)

	fetched, err := txClient.GetSmallEntityByAnint(ctx, 1232)
	chkErr(t, err)

	// the ids won't match up, so just length check for now
	if len(fetched) != len(entities) {
		t.Fatalf("not %v ~= %v", entities, fetched)
	}

	ids := make([]int64, len(fetched))[:0]
	for _, ent := range fetched {
		ids = append(ids, ent.Id)
	}

	fetched2, err := txClient.ListSmallEntity(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(fetched, func(i, j int) bool {
		return fetched[i].Id < fetched[j].Id
	})
	sort.Slice(fetched2, func(i, j int) bool {
		return fetched2[i].Id < fetched[j].Id
	})

	if !reflect.DeepEqual(fetched, fetched2) {
		t.Fatalf("deep cmp: %v != %v", fetched, fetched2)
	}
}

func TestSmallEntityUpdate(t *testing.T) {
	// tests Update and Get

	txClient := newTx(t)
	defer func() {
		_ = txClient.DB.(*sql.Tx).Rollback()
	}()

	entities := []db_shims.SmallEntity{
		{
			Anint: 1232,
		},
		{
			Anint: 1232,
		},
		{
			Anint: 1232,
		},
	}

	_, err := txClient.BulkInsertSmallEntity(ctx, entities)
	chkErr(t, err)

	fetched, err := txClient.GetSmallEntityByAnint(ctx, 1232)
	chkErr(t, err)

	noOpBitset := bitset.New(2)
	noOpBitset.Set(db_shims.SmallEntityIdFieldIndex)

	fetched[0].Anint = 34
	id, err := txClient.UpdateSmallEntity(ctx, fetched[0], noOpBitset)
	if err != nil {
		t.Fatal(err)
	}
	if id != fetched[0].Id {
		t.Fatalf("update id mismatch")
	}

	e0, err := txClient.GetSmallEntityByID(ctx, fetched[0].Id)
	chkErr(t, err)
	if e0[0].Anint == 34 {
		t.Fatalf("unexpected update")
	}

	fetched[1].Anint = 42
	id, err = txClient.UpdateSmallEntity(ctx, fetched[1], db_shims.SmallEntityAllFields)
	chkErr(t, err)
	if id != fetched[1].Id {
		t.Fatalf("id mismatch (passed in %d, got back %d)", fetched[1].Id, id)
	}
	e1, err := txClient.GetSmallEntity(ctx, fetched[1].Id)
	chkErr(t, err)
	if e1.Anint != 42 {
		t.Fatalf("update failed e1 = %#v", e1)
	}
}

func TestSmallEntityCreateDelete(t *testing.T) {
	// tests BulkInsert, BulkDelete and Delete

	txClient := newTx(t)
	defer func() {
		_ = txClient.DB.(*sql.Tx).Rollback()
	}()

	entities := []db_shims.SmallEntity{
		{
			Anint: 232,
		},
		{
			Anint: 232,
		},
		{
			Anint: 232,
		},
		{
			Anint: 232,
		},
		{
			Anint: 232,
		},
		{
			Anint: 232,
		},
	}

	ids, err := txClient.BulkInsertSmallEntity(ctx, entities)
	chkErr(t, err)

	chkErr(t, txClient.BulkDeleteSmallEntity(ctx, ids[:2]))

	fetched, err := txClient.GetSmallEntityByAnint(ctx, 232)
	chkErr(t, err)

	if len(fetched) != 4 {
		t.Fatalf("expected 2 entities to be deleted")
	}

	chkErr(t, txClient.DeleteSmallEntity(ctx, ids[4]))
	fetched, err = txClient.GetSmallEntityByAnint(ctx, 232)
	chkErr(t, err)

	if len(fetched) != 3 {
		t.Fatalf(
			"expected 3 entities to be deleted (%d present)",
			len(fetched),
		)
	}
}

// A test of the "FillAll" functionality
func TestFill(t *testing.T) {
	txClient := newTx(t)
	defer func() {
		_ = txClient.DB.(*sql.Tx).Rollback()
	}()

	entityID, err := txClient.InsertSmallEntity(ctx, db_shims.SmallEntity{
		Anint: 129,
	})
	chkErr(t, err)

	foo := "foo"
	attachmentID1, err := txClient.InsertAttachment(ctx, db_shims.Attachment{
		SmallEntityId: entityID,
		Value:         &foo,
	})
	chkErr(t, err)

	bar := "bar"
	attachmentID2, err := txClient.InsertAttachment(ctx, db_shims.Attachment{
		SmallEntityId: entityID,
		Value:         &bar,
	})
	chkErr(t, err)

	e, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = txClient.SmallEntityFillAll(ctx, &e)
	chkErr(t, err)

	if len(e.Attachments) != 2 {
		t.Fatalf("len attachments = %d, expected 2", len(e.Attachments))
	}
	for _, a := range e.Attachments {
		if !(uuid.Equal(a.Id, attachmentID1) || uuid.Equal(a.Id, attachmentID2)) {
			t.Fatalf(
				"a.Id = %s, expected %s or %s",
				a.Id.String(),
				attachmentID1,
				attachmentID2,
			)
		}
	}
}
