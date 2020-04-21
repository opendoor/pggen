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
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	entity := db_shims.SmallEntity{
		Anint: 129,
	}

	id, err := txClient.InsertSmallEntity(ctx, &entity)
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

	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
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

	_, err = txClient.BulkInsertSmallEntity(ctx, entities)
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

	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
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

	_, err = txClient.BulkInsertSmallEntity(ctx, entities)
	chkErr(t, err)

	fetched, err := txClient.GetSmallEntityByAnint(ctx, 1232)
	chkErr(t, err)

	noOpBitset := pggen.NewFieldSet(2)
	noOpBitset.Set(db_shims.SmallEntityIdFieldIndex, true)

	fetched[0].Anint = 34
	id, err := txClient.UpdateSmallEntity(ctx, &fetched[0], noOpBitset)
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
	id, err = txClient.UpdateSmallEntity(ctx, &fetched[1], db_shims.SmallEntityAllFields)
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

	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
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
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	entityID, err := txClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 129,
	})
	chkErr(t, err)

	foo := "foo"
	attachmentID1, err := txClient.InsertAttachment(ctx, &db_shims.Attachment{
		SmallEntityId: entityID,
		Value:         &foo,
	})
	chkErr(t, err)

	bar := "bar"
	attachmentID2, err := txClient.InsertAttachment(ctx, &db_shims.Attachment{
		SmallEntityId: entityID,
		Value:         &bar,
	})
	chkErr(t, err)

	aTime := time.Unix(5432553, 0)
	_, err = txClient.InsertSingleAttachment(ctx, &db_shims.SingleAttachment{
		SmallEntityId: entityID,
		CreatedAt:     aTime,
	})
	chkErr(t, err)

	e, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = txClient.SmallEntityFillIncludes(ctx, e, db_shims.SmallEntityAllIncludes)
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
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	entityID, err := txClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 129,
	})
	chkErr(t, err)

	foo := "foo"
	attachmentID1, err := txClient.InsertAttachment(ctx, &db_shims.Attachment{
		SmallEntityId: entityID,
		Value:         &foo,
	})
	chkErr(t, err)

	aTime := time.Unix(5432553, 0)
	singleAttachmentID, err := txClient.InsertSingleAttachment(ctx, &db_shims.SingleAttachment{
		SmallEntityId: entityID,
		CreatedAt:     aTime,
	})
	chkErr(t, err)

	// we are going to use include specs to load the attachment, but no the SingleAttachment
	includes := include.Must(include.Parse("small_entities.attachments"))
	smallEntity, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = txClient.SmallEntityFillIncludes(ctx, smallEntity, includes)
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
	err = txClient.SmallEntityFillIncludes(ctx, smallEntity, includes)
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

func TestNullableAttachments(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	entityID, err := txClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 3,
	})
	chkErr(t, err)

	// one that isn't attached
	_, err = txClient.InsertNullableAttachment(ctx, &db_shims.NullableAttachment{
		Value: "not attached",
	})
	chkErr(t, err)

	// and one that is
	_, err = txClient.InsertNullableAttachment(ctx, &db_shims.NullableAttachment{
		SmallEntityId: &entityID,
		Value:         "attached",
	})
	chkErr(t, err)

	smallEntity, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)

	err = txClient.SmallEntityFillIncludes(ctx, smallEntity, db_shims.SmallEntityAllIncludes)
	chkErr(t, err)

	if len(smallEntity.NullableAttachments) != 1 {
		t.Fatalf("expected exactly 1 attachment")
	}
	if smallEntity.NullableAttachments[0].Value != "attached" {
		t.Fatalf("should be attached")
	}
}

func TestNullableSingleAttachments(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	entityID, err := txClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 3,
	})
	chkErr(t, err)

	// one that isn't attached
	_, err = txClient.InsertNullableSingleAttachment(ctx, &db_shims.NullableSingleAttachment{
		Value: "not attached",
	})
	chkErr(t, err)

	// and one that is
	_, err = txClient.InsertNullableSingleAttachment(ctx, &db_shims.NullableSingleAttachment{
		SmallEntityId: &entityID,
		Value:         "attached",
	})
	chkErr(t, err)

	smallEntity, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)

	err = txClient.SmallEntityFillIncludes(ctx, smallEntity, db_shims.SmallEntityAllIncludes)
	chkErr(t, err)

	if smallEntity.NullableSingleAttachment.Value != "attached" {
		t.Fatalf("should be attached")
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
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	var nineteen int64 = 19
	funnyID, err := txClient.InsertWeirdNaMe(ctx, &db_shims.WeirdNaMe{
		WearetalkingReallyBadstyle: 1923,
		GotWhitespace:              "yes",
		ButWhyTho:                  &nineteen,
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

	kidID, err := txClient.InsertWeirdKid(ctx, &db_shims.WeirdKid{
		Daddy: funny.Evenidisweird,
	})
	chkErr(t, err)
	err = txClient.WeirdNaMeFillIncludes(ctx, funny, db_shims.WeirdNaMeAllIncludes)
	chkErr(t, err)

	err = txClient.DeleteWeirdKid(ctx, kidID)
	chkErr(t, err)

	err = txClient.DeleteWeirdNaMe(ctx, funny.Evenidisweird)
	chkErr(t, err)
}

func TestArrayMembers(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	var nineteen int64 = 19
	id, err := txClient.InsertArrayMember(ctx, &db_shims.ArrayMember{
		TextArray: []string{"foo", "bar"},
		IntArray:  []*int64{&nineteen, nil},
	})
	chkErr(t, err)

	arrayMember, err := txClient.GetArrayMember(ctx, id)
	chkErr(t, err)

	_, err = txClient.UpdateArrayMember(
		ctx, arrayMember, db_shims.ArrayMemberAllFields)
	chkErr(t, err)
}

func TestMaxFieldIndex(t *testing.T) {
	if db_shims.SmallEntityMaxFieldIndex != db_shims.SmallEntityAnintFieldIndex {
		t.Fatalf("max field index mismatch")
	}
}

func TestColOrdering(t *testing.T) {
	// change the table so that it is the same, but has a different column ordering
	_, err := pgClient.Handle().(*sql.DB).Exec(`
		DROP TABLE col_order;
		CREATE TABLE col_order (
			field3 int NOT NULL,
			id SERIAL PRIMARY KEY,
			field2 int NOT NULL,
			dropped text,
			field1 text NOT NULL
		);
		ALTER TABLE col_order DROP COLUMN dropped;
	`)
	chkErr(t, err)
	defer func() {
		_, err := pgClient.Handle().(*sql.DB).Exec(`
			DROP TABLE col_order;
			CREATE TABLE col_order (
				id SERIAL PRIMARY KEY,
				field1 text NOT NULL,
				dropped text,
				field2 int NOT NULL,
				field3 int NOT NULL
			);
			ALTER TABLE col_order DROP COLUMN dropped;
		`)
		chkErr(t, err)
	}()

	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	id, err := txClient.InsertColOrder(ctx, &db_shims.ColOrder{
		Field1: "foo",
		Field2: 1,
		Field3: 2,
	})
	chkErr(t, err)

	rec, err := txClient.GetColOrder(ctx, id)
	chkErr(t, err)

	if rec.Field1 != "foo" || rec.Field2 != 1 || rec.Field3 != 2 {
		t.Fatalf("rec = %#v", rec)
	}
}

func TestUpsertInserts(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	allButID := db_shims.SmallEntityAllFields.Clone()
	allButID.Set(db_shims.SmallEntityIdFieldIndex, false)

	id1, err := txClient.UpsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 19,
	}, nil, allButID)
	chkErr(t, err)

	fetched, err := txClient.GetSmallEntity(ctx, id1)
	chkErr(t, err)
	if fetched.Anint != 19 {
		t.Fatal("expected 19")
	}

	id2, err := txClient.UpsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 20,
	}, nil, allButID)
	chkErr(t, err)

	fetched, err = txClient.GetSmallEntity(ctx, id2)
	chkErr(t, err)
	if fetched.Anint != 20 {
		t.Fatal("expected 20")
	}

	if id1 == id2 {
		t.Fatal("should not have overwritten")
	}
}

func TestUpsertUpdates(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	id, err := txClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 19,
	})
	chkErr(t, err)

	id, err = txClient.UpsertSmallEntity(ctx, &db_shims.SmallEntity{
		Id:    id,
		Anint: 14,
	}, nil, db_shims.SmallEntityAllFields)
	chkErr(t, err)

	fetched, err := txClient.GetSmallEntity(ctx, id)
	chkErr(t, err)
	if fetched.Anint != 14 {
		t.Fatalf("expected 14")
	}
}

func TestUpsertDoesntUpdateThingsNotInFieldSet(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	justID := pggen.NewFieldSet(db_shims.SmallEntityMaxFieldIndex)
	justID.Set(db_shims.SmallEntityIdFieldIndex, true)

	id, err := txClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 19,
	})
	chkErr(t, err)

	id, err = txClient.UpsertSmallEntity(ctx, &db_shims.SmallEntity{
		Id:    id,
		Anint: 14,
	}, []string{}, justID)
	chkErr(t, err)

	fetched, err := txClient.GetSmallEntity(ctx, id)
	chkErr(t, err)
	if fetched.Anint != 19 {
		t.Fatalf("expected 19")
	}
}

func TestBulkUpsert(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	id, err := txClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 19,
	})
	chkErr(t, err)

	ids, err := txClient.BulkUpsertSmallEntity(ctx, []db_shims.SmallEntity{
		{
			Id:    id,
			Anint: 9,
		},
		{
			Anint: 57,
		},
	}, nil, db_shims.SmallEntityAllFields)
	chkErr(t, err)

	entities, err := txClient.ListSmallEntity(ctx, ids)
	chkErr(t, err)
	for _, e := range entities {
		if !(e.Anint == 9 || e.Anint == 57) {
			t.Fatal("expected only 9 and 57")
		}
	}
}

func TestUpsertWithExplicitConstraints(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	otherMask := pggen.NewFieldSet(db_shims.ConstraintMaxFieldIndex)
	otherMask.Set(db_shims.ConstraintOtherFieldIndex, true)

	id1, err := txClient.UpsertConstraint(ctx, &db_shims.Constraint{
		Snowflake: 2,
		Other:     19,
	}, []string{"snowflake"}, otherMask)
	chkErr(t, err)

	id2, err := txClient.UpsertConstraint(ctx, &db_shims.Constraint{
		Snowflake: 2,
		Other:     4,
	}, []string{"snowflake"}, otherMask)
	chkErr(t, err)

	fetched, err := txClient.GetConstraint(ctx, id1)
	chkErr(t, err)
	if fetched.Snowflake != 2 || id1 != id2 || fetched.Other != 4 {
		t.Fatal("expected update")
	}
}

func TestBulkUpsertEmptyList(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	allButID := db_shims.SmallEntityAllFields.Clone()
	allButID.Set(db_shims.SmallEntityIdFieldIndex, false)

	ids, err := txClient.BulkUpsertSmallEntity(ctx, []db_shims.SmallEntity{}, nil, allButID)
	chkErr(t, err)

	if len(ids) != 0 {
		t.Fatal("expected no ids")
	}
}

func TestUpsertNullableArray(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	_, err = txClient.UpsertTextArray(ctx, &db_shims.TextArray{
		Value: []*string{&[]string{"foo"}[0], nil},
	}, nil, db_shims.TextArrayAllFields)
	chkErr(t, err)
}

func TestEnumBlanks(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	_, err = txClient.InsertEnumBlank(ctx, &db_shims.EnumBlank{
		Value: db_shims.EnumTypeWithBlankBlank0,
	})
	chkErr(t, err)

	_, err = txClient.InsertEnumBlank(ctx, &db_shims.EnumBlank{
		Value: db_shims.EnumTypeWithBlankBlank,
	})
	chkErr(t, err)

	if db_shims.EnumTypeWithBlankBlank.String() != "blank" {
		t.Fatal("should not actually be blank")
	}

	if db_shims.EnumTypeWithBlankBlank0.String() != "" {
		t.Fatal("should be blank")
	}
}

func TestBasicCycle(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	cycle1ID, err := txClient.InsertCycle1(ctx, &db_shims.Cycle1{
		Value: "foo",
	})
	chkErr(t, err)

	cycle2ID, err := txClient.InsertCycle2(ctx, &db_shims.Cycle2{
		Cycle1Id: cycle1ID,
		Value:    9,
	})
	chkErr(t, err)

	cycle2IDMask := pggen.NewFieldSet(db_shims.Cycle1MaxFieldIndex)
	cycle2IDMask.Set(db_shims.Cycle1IdFieldIndex, true)
	cycle2IDMask.Set(db_shims.Cycle1Cycle2IdFieldIndex, true)
	_, err = txClient.UpdateCycle1(ctx, &db_shims.Cycle1{
		Id:       cycle1ID,
		Cycle2Id: &cycle2ID,
	}, cycle2IDMask)
	chkErr(t, err)

	cycle1, err := txClient.GetCycle1(ctx, cycle1ID)
	chkErr(t, err)

	err = txClient.Cycle1FillIncludes(ctx, cycle1, db_shims.Cycle1AllIncludes)
	chkErr(t, err)

	if len(cycle1.Cycle2) != 1 {
		t.Fatalf("expected exactly 1 cycle2")
	}

	cycle2 := cycle1.Cycle2[0]
	if cycle2.Value != 9 {
		t.Fatalf("expected 9")
	}

	if len(cycle2.Cycle1) != 1 {
		t.Fatalf("expected exactly 1 cycle1")
	}

	roundaboutCycle1 := cycle2.Cycle1[0]
	if roundaboutCycle1.Value != "foo" {
		t.Fatalf("expected foo")
	}

	if cycle1 != roundaboutCycle1 {
		t.Fatalf("expected them to actually be the same object, not just have the same values")
	}
}

func TestCycleTree(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	rootID, err := txClient.InsertCycleTreeRoot(ctx, &db_shims.CycleTreeRoot{
		Value: "root",
	})
	chkErr(t, err)

	branch1ID, err := txClient.InsertCycleTreeBranch1(ctx, &db_shims.CycleTreeBranch1{
		Value:           "branch-1",
		CycleTreeRootId: rootID,
	})
	chkErr(t, err)

	branch2ID, err := txClient.InsertCycleTreeBranch2(ctx, &db_shims.CycleTreeBranch2{
		Value:           "branch-2",
		CycleTreeRootId: rootID,
	})
	chkErr(t, err)

	cycle1ID, err := txClient.InsertCycleTreeCycle1(ctx, &db_shims.CycleTreeCycle1{
		Value:              "cycle-1",
		CycleTreeBranch1Id: branch1ID,
	})
	chkErr(t, err)

	cycle2ID, err := txClient.InsertCycleTreeCycle2(ctx, &db_shims.CycleTreeCycle2{
		Value:              "cycle-2",
		CycleTreeBranch2Id: branch2ID,
		CycleTreeCycle1Id:  cycle1ID,
	})
	chkErr(t, err)

	cycle3ID, err := txClient.InsertCycleTreeCycle3(ctx, &db_shims.CycleTreeCycle3{
		Value:             "cycle-3",
		CycleTreeCycle2Id: cycle2ID,
	})
	chkErr(t, err)

	cycle3IDMask := pggen.NewFieldSet(db_shims.CycleTreeCycle1MaxFieldIndex)
	cycle3IDMask.Set(db_shims.CycleTreeCycle1IdFieldIndex, true)
	cycle3IDMask.Set(db_shims.CycleTreeCycle1CycleTreeCycle3IdFieldIndex, true)
	_, err = txClient.UpdateCycleTreeCycle1(ctx, &db_shims.CycleTreeCycle1{
		Id:                cycle1ID,
		CycleTreeCycle3Id: &cycle3ID,
	}, cycle3IDMask)
	chkErr(t, err)

	root, err := txClient.GetCycleTreeRoot(ctx, rootID)
	chkErr(t, err)

	err = txClient.CycleTreeRootFillIncludes(ctx, root, db_shims.CycleTreeRootAllIncludes)
	chkErr(t, err)

	c1 := root.CycleTreeBranch1[0].CycleTreeCycle1
	c2 := root.CycleTreeBranch2.CycleTreeCycle2
	cycle1s := []*db_shims.CycleTreeCycle1{
		c1,
		c1.CycleTreeCycle2.CycleTreeCycle3.CycleTreeCycle1[0],
		c2.CycleTreeCycle3.CycleTreeCycle1[0],
	}

	for i, node := range cycle1s {
		for j, other := range cycle1s {
			if i == j {
				continue // no need to check this case
			}

			if node != other {
				t.Fatalf("expected all cycle1s to have the same object identity")
			}
		}
	}
}

// We should be able to work with tables that have had an extra column
// added since we generated code.
func TestNewColumn(t *testing.T) {
	willGetNewColumnType := reflect.TypeOf(db_shims.WillGetNewColumn{})
	_, has := willGetNewColumnType.FieldByName("F2")
	if has {
		t.Fatalf("pggen generated an F2 field. There is something wrong with the db state.")
	}

	_, err := pgClient.Handle().Exec("ALTER TABLE will_get_new_column ADD COLUMN f2 integer")
	chkErr(t, err)
	defer func() {
		_, err = pgClient.Handle().Exec("ALTER TABLE will_get_new_column DROP COLUMN f2")
		chkErr(t, err)
	}()

	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	// Force the column index caches to clear so we can be sure that the lazy loading
	// logic triggers.
	txClient.ClearCaches()

	id, err := txClient.InsertWillGetNewColumn(ctx, &db_shims.WillGetNewColumn{
		F1: "foo",
	})
	chkErr(t, err)

	fetched, err := txClient.GetWillGetNewColumn(ctx, id)
	chkErr(t, err)

	if fetched.F1 != "foo" {
		t.Fatalf("expected foo")
	}
}
