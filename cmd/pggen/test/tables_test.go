package test

import (
	"database/sql"
	"reflect"
	"sort"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/cmd/pggen/test/db_shims"
	"github.com/opendoor-labs/pggen/include"
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

	noOpBitset := pggen.NewFieldSet(2)
	noOpBitset.Set(db_shims.SmallEntityIdFieldIndex, true)

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

func TestFillAll(t *testing.T) {
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
		Value:         sql.NullString{String: foo, Valid: true},
	})
	chkErr(t, err)

	bar := "bar"
	attachmentID2, err := txClient.InsertAttachment(ctx, db_shims.Attachment{
		SmallEntityId: entityID,
		Value:         sql.NullString{String: bar, Valid: true},
	})
	chkErr(t, err)

	aTime := time.Unix(5432553, 0)
	_, err = txClient.InsertSingleAttachment(ctx, db_shims.SingleAttachment{
		SmallEntityId: entityID,
		CreatedAt:     aTime,
	})
	chkErr(t, err)

	e, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = txClient.SmallEntityFillIncludes(ctx, &e, db_shims.SmallEntityAllIncludes)
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

	if !e.SingleAttachment.CreatedAt.Equal(aTime) {
		t.Fatalf(
			"single attachment time clash: '%s' != '%s'",
			aTime.String(),
			e.SingleAttachment.CreatedAt.String(),
		)
	}
}

func TestFillIncludes(t *testing.T) {
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
		Value:         sql.NullString{String: foo, Valid: true},
	})
	chkErr(t, err)

	aTime := time.Unix(5432553, 0)
	singleAttachmentID, err := txClient.InsertSingleAttachment(ctx, db_shims.SingleAttachment{
		SmallEntityId: entityID,
		CreatedAt:     aTime,
	})
	chkErr(t, err)

	// we are going to use include specs to load the attachment, but no the SingleAttachment
	includes := include.Must(include.Parse("small_entities.attachments"))
	smallEntity, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = txClient.SmallEntityFillIncludes(ctx, &smallEntity, includes)
	chkErr(t, err)

	if smallEntity.Attachments[0].Id != attachmentID1 {
		t.Fatalf("failed to fetch attachment")
	}
	if smallEntity.SingleAttachment != nil {
		t.Fatalf("fetched single attachment when it wasn't in the include set")
	}

	// now do load the single_attachments
	includes = include.Must(include.Parse("small_entities.{attachments, single_attachments}"))
	smallEntity, err = txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = txClient.SmallEntityFillIncludes(ctx, &smallEntity, includes)
	chkErr(t, err)

	if smallEntity.Attachments[0].Id != attachmentID1 {
		t.Fatalf("failed to fetch attachment")
	}
	if smallEntity.SingleAttachment.Id != singleAttachmentID {
		t.Fatalf("failed to fetch single attachment")
	}
	if smallEntity.ExplicitBelongsTo != nil || smallEntity.ExplicitBelongsToMany != nil {
		t.Fatalf("shouldn't load ExplicitBelongsTo or ExplicitBelongsToMany")
	}
}

func TestNoInfer(t *testing.T) {
	smallEntityType := reflect.TypeOf(db_shims.SmallEntity{})

	_, has := smallEntityType.FieldByName("NoInfer")
	if has {
		t.Fatalf("pggen generated a NoInfer field when we asked it not to")
	}
}

func TestExplicitBelongsTo(t *testing.T) {
	smallEntityType := reflect.TypeOf(db_shims.SmallEntity{})

	f, has := smallEntityType.FieldByName("ExplicitBelongsTo")
	if !has {
		t.Fatalf("pggen generated failed to generate ExplicitBelongsTo")
	}

	if f.Type.Kind() == reflect.Array {
		t.Fatalf("pggen generated a 1-many instead of a 1-1")
	}
}

func TestExplicitBelongsToMany(t *testing.T) {
	smallEntityType := reflect.TypeOf(db_shims.SmallEntity{})

	f, has := smallEntityType.FieldByName("ExplicitBelongsToMany")
	if !has {
		t.Fatalf("pggen generated failed to generate ExplicitBelongsToMany")
	}

	if f.Type.Kind() != reflect.Slice {
		t.Fatalf("pggen generated a 1-1 instead of 1-many")
	}
}

func TestFunnyNamesInTableGeneratedFunc(t *testing.T) {
	txClient := newTx(t)
	defer func() {
		_ = txClient.DB.(*sql.Tx).Rollback()
	}()

	funnyID, err := txClient.InsertWeirdNaMe(ctx, db_shims.WeirdNaMe{
		WearetalkingReallyBadstyle: 1923,
		GotWhitespace:              "yes",
		ButWhyTho:                  sql.NullInt64{Int64: 19, Valid: true},
	})
	chkErr(t, err)

	funny, err := txClient.GetWeirdNaMe(ctx, funnyID)
	chkErr(t, err)

	funny.GotWhitespace = "no"

	funnyID, err = txClient.UpdateWeirdNaMe(
		ctx, funny, db_shims.WeirdNaMeAllFields)
	chkErr(t, err)

	funny, err = txClient.GetWeirdNaMe(ctx, funnyID)
	chkErr(t, err)

	if funny.GotWhitespace != "no" {
		t.Fatalf("update failed")
	}

	kidID, err := txClient.InsertWeirdKid(ctx, db_shims.WeirdKid{
		Daddy: funny.Evenidisweird,
	})
	chkErr(t, err)
	err = txClient.WeirdNaMeFillIncludes(ctx, &funny, db_shims.WeirdNaMeAllIncludes)
	chkErr(t, err)

	err = txClient.DeleteWeirdKid(ctx, kidID)
	chkErr(t, err)

	err = txClient.DeleteWeirdNaMe(ctx, funny.Evenidisweird)
	chkErr(t, err)
}
