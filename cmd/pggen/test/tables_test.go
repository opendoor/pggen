package test

import (
	"database/sql"
	"reflect"
	"sort"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/cmd/pggen/test/models"
	"github.com/opendoor-labs/pggen/include"
)

func TestInsertSmallEntity(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	entity := models.SmallEntity{
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

	entities := []models.SmallEntity{
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

	entities := []models.SmallEntity{
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
	noOpBitset.Set(models.SmallEntityIdFieldIndex, true)

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
	id, err = txClient.UpdateSmallEntity(ctx, &fetched[1], models.SmallEntityAllFields)
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

	entities := []models.SmallEntity{
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

	entityID, err := txClient.InsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 129,
	})
	chkErr(t, err)

	foo := "foo"
	attachmentID1, err := txClient.InsertAttachment(ctx, &models.Attachment{
		SmallEntityId: entityID,
		Value:         &foo,
	})
	chkErr(t, err)

	bar := "bar"
	attachmentID2, err := txClient.InsertAttachment(ctx, &models.Attachment{
		SmallEntityId: entityID,
		Value:         &bar,
	})
	chkErr(t, err)

	aTime := time.Unix(5432553, 0)
	_, err = txClient.InsertSingleAttachment(ctx, &models.SingleAttachment{
		SmallEntityId: entityID,
		CreatedAt:     aTime,
	})
	chkErr(t, err)

	e, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = txClient.SmallEntityFillIncludes(ctx, e, models.SmallEntityAllIncludes)
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

	entityID, err := txClient.InsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 129,
	})
	chkErr(t, err)

	foo := "foo"
	attachmentID1, err := txClient.InsertAttachment(ctx, &models.Attachment{
		SmallEntityId: entityID,
		Value:         &foo,
	})
	chkErr(t, err)

	aTime := time.Unix(5432553, 0)
	singleAttachmentID, err := txClient.InsertSingleAttachment(ctx, &models.SingleAttachment{
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

	entityID, err := txClient.InsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 3,
	})
	chkErr(t, err)

	// one that isn't attached
	_, err = txClient.InsertNullableAttachment(ctx, &models.NullableAttachment{
		Value: "not attached",
	})
	chkErr(t, err)

	// and one that is
	_, err = txClient.InsertNullableAttachment(ctx, &models.NullableAttachment{
		SmallEntityId: &entityID,
		Value:         "attached",
	})
	chkErr(t, err)

	smallEntity, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)

	err = txClient.SmallEntityFillIncludes(ctx, smallEntity, models.SmallEntityAllIncludes)
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

	entityID, err := txClient.InsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 3,
	})
	chkErr(t, err)

	// one that isn't attached
	_, err = txClient.InsertNullableSingleAttachment(ctx, &models.NullableSingleAttachment{
		Value: "not attached",
	})
	chkErr(t, err)

	// and one that is
	_, err = txClient.InsertNullableSingleAttachment(ctx, &models.NullableSingleAttachment{
		SmallEntityId: &entityID,
		Value:         "attached",
	})
	chkErr(t, err)

	smallEntity, err := txClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)

	err = txClient.SmallEntityFillIncludes(ctx, smallEntity, models.SmallEntityAllIncludes)
	chkErr(t, err)

	if smallEntity.NullableSingleAttachment.Value != "attached" {
		t.Fatalf("should be attached")
	}
}

func TestNoInfer(t *testing.T) {
	smallEntityType := reflect.TypeOf(models.SmallEntity{})

	_, has := smallEntityType.FieldByName("NoInfer")
	if has {
		t.Fatalf("pggen generated a NoInfer field when we asked it not to")
	}
}

func TestExplicitBelongsTo(t *testing.T) {
	smallEntityType := reflect.TypeOf(models.SmallEntity{})

	f, has := smallEntityType.FieldByName("ExplicitBelongsTo")
	if !has {
		t.Fatalf("pggen generated failed to generate ExplicitBelongsTo")
	}

	if f.Type.Kind() == reflect.Array {
		t.Fatalf("pggen generated a 1-many instead of a 1-1")
	}
}

func TestExplicitBelongsToMany(t *testing.T) {
	smallEntityType := reflect.TypeOf(models.SmallEntity{})

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
	funnyID, err := txClient.InsertWeirdNaMe(ctx, &models.WeirdNaMe{
		WearetalkingReallyBadstyle: 1923,
		GotWhitespace:              "yes",
		ButWhyTho:                  &nineteen,
	})
	chkErr(t, err)

	funny, err := txClient.GetWeirdNaMe(ctx, funnyID)
	chkErr(t, err)

	funny.GotWhitespace = "no"

	funnyID, err = txClient.UpdateWeirdNaMe(
		ctx, funny, models.WeirdNaMeAllFields)
	chkErr(t, err)

	funny, err = txClient.GetWeirdNaMe(ctx, funnyID)
	chkErr(t, err)

	if funny.GotWhitespace != "no" {
		t.Fatalf("update failed")
	}

	kidID, err := txClient.InsertWeirdKid(ctx, &models.WeirdKid{
		Daddy: funny.Evenidisweird,
	})
	chkErr(t, err)
	err = txClient.WeirdNaMeFillIncludes(ctx, funny, models.WeirdNaMeAllIncludes)
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
	id, err := txClient.InsertArrayMember(ctx, &models.ArrayMember{
		TextArray: []string{"foo", "bar"},
		IntArray:  []*int64{&nineteen, nil},
	})
	chkErr(t, err)

	arrayMember, err := txClient.GetArrayMember(ctx, id)
	chkErr(t, err)

	_, err = txClient.UpdateArrayMember(
		ctx, arrayMember, models.ArrayMemberAllFields)
	chkErr(t, err)
}

func TestMaxFieldIndex(t *testing.T) {
	if models.SmallEntityMaxFieldIndex != models.SmallEntityAnintFieldIndex {
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

	id, err := txClient.InsertColOrder(ctx, &models.ColOrder{
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

	allButID := models.SmallEntityAllFields.Clone()
	allButID.Set(models.SmallEntityIdFieldIndex, false)

	id1, err := txClient.UpsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 19,
	}, nil, allButID)
	chkErr(t, err)

	fetched, err := txClient.GetSmallEntity(ctx, id1)
	chkErr(t, err)
	if fetched.Anint != 19 {
		t.Fatal("expected 19")
	}

	id2, err := txClient.UpsertSmallEntity(ctx, &models.SmallEntity{
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

	id, err := txClient.InsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 19,
	})
	chkErr(t, err)

	id, err = txClient.UpsertSmallEntity(ctx, &models.SmallEntity{
		Id:    id,
		Anint: 14,
	}, nil, models.SmallEntityAllFields)
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

	empty := pggen.NewFieldSet(models.SmallEntityMaxFieldIndex)

	id, err := txClient.InsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 19,
	})
	chkErr(t, err)

	id, err = txClient.UpsertSmallEntity(ctx, &models.SmallEntity{
		Id:    id,
		Anint: 14,
	}, []string{}, empty, pggen.UpsertUsePkey)
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

	id, err := txClient.InsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 19,
	})
	chkErr(t, err)

	ids, err := txClient.BulkUpsertSmallEntity(ctx, []models.SmallEntity{
		{
			Id:    id,
			Anint: 9,
		},
		{
			Anint: 57,
		},
	}, nil, models.SmallEntityAllFields)
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

	otherMask := pggen.NewFieldSet(models.ConstraintMaxFieldIndex)
	otherMask.Set(models.ConstraintOtherFieldIndex, true)

	id1, err := txClient.UpsertConstraint(ctx, &models.Constraint{
		Snowflake: 2,
		Other:     19,
	}, []string{"snowflake"}, otherMask)
	chkErr(t, err)

	id2, err := txClient.UpsertConstraint(ctx, &models.Constraint{
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

	allButID := models.SmallEntityAllFields.Clone()
	allButID.Set(models.SmallEntityIdFieldIndex, false)

	ids, err := txClient.BulkUpsertSmallEntity(ctx, []models.SmallEntity{}, nil, allButID)
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

	_, err = txClient.UpsertTextArray(ctx, &models.TextArray{
		Value: []*string{&[]string{"foo"}[0], nil},
	}, nil, models.TextArrayAllFields)
	chkErr(t, err)
}

func TestEnumBlanks(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	_, err = txClient.InsertEnumBlank(ctx, &models.EnumBlank{
		Value: models.EnumTypeWithBlankBlank0,
	})
	chkErr(t, err)

	id, err := txClient.InsertEnumBlank(ctx, &models.EnumBlank{
		Value: models.EnumTypeWithBlankBlank,
	})
	chkErr(t, err)

	_, err = txClient.GetEnumBlank(ctx, id)
	chkErr(t, err)

	if models.EnumTypeWithBlankBlank.String() != "blank" {
		t.Fatal("should not actually be blank")
	}

	if models.EnumTypeWithBlankBlank0.String() != "" {
		t.Fatal("should be blank")
	}
}

func TestBasicCycle(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	cycle1ID, err := txClient.InsertCycle1(ctx, &models.Cycle1{
		Value: "foo",
	})
	chkErr(t, err)

	cycle2ID, err := txClient.InsertCycle2(ctx, &models.Cycle2{
		Cycle1Id: cycle1ID,
		Value:    9,
	})
	chkErr(t, err)

	cycle2IDMask := pggen.NewFieldSet(models.Cycle1MaxFieldIndex)
	cycle2IDMask.Set(models.Cycle1IdFieldIndex, true)
	cycle2IDMask.Set(models.Cycle1Cycle2IdFieldIndex, true)
	_, err = txClient.UpdateCycle1(ctx, &models.Cycle1{
		Id:       cycle1ID,
		Cycle2Id: &cycle2ID,
	}, cycle2IDMask)
	chkErr(t, err)

	cycle1, err := txClient.GetCycle1(ctx, cycle1ID)
	chkErr(t, err)

	err = txClient.Cycle1FillIncludes(ctx, cycle1, models.Cycle1AllIncludes)
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

	if cycle1 != cycle1.Cycle2[0].Cycle1Parent {
		t.Fatal("parent pointer mishap 1")
	}
	if cycle2 != cycle2.Cycle1[0].Cycle2Parent {
		t.Fatal("parent pointer mishap 2")
	}
}

func TestCycleTree(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	rootID, err := txClient.InsertCycleTreeRoot(ctx, &models.CycleTreeRoot{
		Value: "root",
	})
	chkErr(t, err)

	branch1ID, err := txClient.InsertCycleTreeBranch1(ctx, &models.CycleTreeBranch1{
		Value:           "branch-1",
		CycleTreeRootId: rootID,
	})
	chkErr(t, err)

	branch2ID, err := txClient.InsertCycleTreeBranch2(ctx, &models.CycleTreeBranch2{
		Value:           "branch-2",
		CycleTreeRootId: rootID,
	})
	chkErr(t, err)

	cycle1ID, err := txClient.InsertCycleTreeCycle1(ctx, &models.CycleTreeCycle1{
		Value:              "cycle-1",
		CycleTreeBranch1Id: branch1ID,
	})
	chkErr(t, err)

	cycle2ID, err := txClient.InsertCycleTreeCycle2(ctx, &models.CycleTreeCycle2{
		Value:              "cycle-2",
		CycleTreeBranch2Id: branch2ID,
		CycleTreeCycle1Id:  cycle1ID,
	})
	chkErr(t, err)

	cycle3ID, err := txClient.InsertCycleTreeCycle3(ctx, &models.CycleTreeCycle3{
		Value:             "cycle-3",
		CycleTreeCycle2Id: cycle2ID,
	})
	chkErr(t, err)

	cycle3IDMask := pggen.NewFieldSet(models.CycleTreeCycle1MaxFieldIndex)
	cycle3IDMask.Set(models.CycleTreeCycle1IdFieldIndex, true)
	cycle3IDMask.Set(models.CycleTreeCycle1CycleTreeCycle3IdFieldIndex, true)
	_, err = txClient.UpdateCycleTreeCycle1(ctx, &models.CycleTreeCycle1{
		Id:                cycle1ID,
		CycleTreeCycle3Id: &cycle3ID,
	}, cycle3IDMask)
	chkErr(t, err)

	root, err := txClient.GetCycleTreeRoot(ctx, rootID)
	chkErr(t, err)

	err = txClient.CycleTreeRootFillIncludes(ctx, root, models.CycleTreeRootAllIncludes)
	chkErr(t, err)

	c1 := root.CycleTreeBranch1[0].CycleTreeCycle1
	c2 := root.CycleTreeBranch2.CycleTreeCycle2
	cycle1s := []*models.CycleTreeCycle1{
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
	willGetNewColumnType := reflect.TypeOf(models.WillGetNewColumn{})
	_, has := willGetNewColumnType.FieldByName("F2")
	if has {
		t.Fatalf("pggen generated an F2 field. There is something wrong with the db state.")
	}

	_, err := pgClient.Handle().ExecContext(ctx, "ALTER TABLE will_get_new_column ADD COLUMN f2 integer")
	chkErr(t, err)
	defer func() {
		_, err = pgClient.Handle().ExecContext(ctx, "ALTER TABLE will_get_new_column DROP COLUMN f2")
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

	id, err := txClient.InsertWillGetNewColumn(ctx, &models.WillGetNewColumn{
		F1: "foo",
	})
	chkErr(t, err)

	fetched, err := txClient.GetWillGetNewColumn(ctx, id)
	chkErr(t, err)

	if fetched.F1 != "foo" {
		t.Fatalf("expected foo")
	}
}

func TestInsertPkey(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	one := int64(1)
	_, err = txClient.InsertNonDefaultPkey(ctx, &models.NonDefaultPkey{
		Id:  "foo",
		Val: &one,
	}, pggen.InsertUsePkey)
	chkErr(t, err)
}

func TestCustomReferenceNames(t *testing.T) {
	smallEntityType := reflect.TypeOf(models.SmallEntity{})

	names := []string{"CustomReferenceName", "Custom1to1ReferenceName"}
	for _, n := range names {
		_, has := smallEntityType.FieldByName(n)
		if !has {
			t.Fatalf("pggen failed to generate '%s'", n)
		}
	}

	shouldBeMissingNames := []string{"AlternativeReferenceName", "AlternativeReferenceName1to1"}
	for _, n := range shouldBeMissingNames {
		_, has := smallEntityType.FieldByName(n)
		if has {
			t.Fatalf("pggen generated '%s'", n)
		}
	}
}

func TestIncludeCustomNames(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	entity := models.SmallEntity{
		Anint: 1892,
	}
	smallEntityID, err := txClient.InsertSmallEntity(ctx, &entity)
	chkErr(t, err)

	altRef1 := models.AlternativeReferenceName{
		SmallEntityId: smallEntityID,
	}
	_, err = txClient.InsertAlternativeReferenceName(ctx, &altRef1)
	chkErr(t, err)

	altRef2 := models.AlternativeReferenceName{
		SmallEntityId: smallEntityID,
	}
	_, err = txClient.InsertAlternativeReferenceName(ctx, &altRef2)
	chkErr(t, err)

	alt1to1Ref := models.AlternativeReferenceName1to1{
		SmallEntityId: smallEntityID,
	}
	_, err = txClient.InsertAlternativeReferenceName1to1(ctx, &alt1to1Ref)
	chkErr(t, err)

	smallE, err := txClient.GetSmallEntity(ctx, smallEntityID)
	chkErr(t, err)

	// We can construct a spec with our custom name, not the name of the table.
	// Because they are custom names, we need to to use a rename expressions in the spec.
	spec := include.Must(include.Parse(`
		small_entities.{
			custom_reference_name -> alternative_reference_name,
			custom_1to1_reference_name -> alternative_reference_name_1to1
		}
	`))

	err = txClient.SmallEntityFillIncludes(ctx, smallE, spec)
	chkErr(t, err)

	if len(smallE.CustomReferenceName) != 2 {
		t.Fatal("custom entities not attached")
	}

	if smallE.Custom1to1ReferenceName == nil {
		t.Fatal("custom 1to1 entity not attached")
	}
}

func TestParentPointers(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	entity := models.SmallEntity{
		Anint: 1892,
	}
	smallEntityID, err := txClient.InsertSmallEntity(ctx, &entity)
	chkErr(t, err)
	entity.Id = smallEntityID

	foo := "foo"
	_, err = txClient.InsertAttachment(ctx, &models.Attachment{
		SmallEntityId: smallEntityID,
		Value:         &foo,
	})
	chkErr(t, err)

	bar := "bar"
	_, err = txClient.InsertAttachment(ctx, &models.Attachment{
		SmallEntityId: smallEntityID,
		Value:         &bar,
	})
	chkErr(t, err)

	err = txClient.SmallEntityFillIncludes(ctx, &entity, models.SmallEntityAllIncludes)
	chkErr(t, err)

	if &entity != entity.Attachments[0].SmallEntity {
		t.Fatal("Attachment(0): bad parent pointer")
	}

	if &entity != entity.Attachments[1].SmallEntity {
		t.Fatal("Attachment(1): bad parent pointer")
	}
}

func TestNotFoundGet(t *testing.T) {
	_, err := pgClient.GetSmallEntity(ctx, 23423)
	if err == nil {
		t.Fatal("expected err")
	}
	if !pggen.IsNotFoundError(err) {
		t.Fatal("expected NotFoundError")
	}
}

func TestNotFoundList(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	entity := models.SmallEntity{
		Anint: 1892,
	}
	smallEntityID, err := txClient.InsertSmallEntity(ctx, &entity)
	chkErr(t, err)

	params := [][]int64{
		{23423, smallEntityID}, // partial match
		{23423},                // just completely missing
	}

	for _, p := range params {
		// a partial match will be tagged with NotFoundError
		_, err = pgClient.ListSmallEntity(ctx, p)
		if err == nil {
			t.Fatal("expected err")
		}
		if !pggen.IsNotFoundError(err) {
			t.Fatal("expected NotFoundError")
		}
	}
}

func TestDroppingColumnOnTheFly(t *testing.T) {
	// make sure we always start in a consistant state
	_, err := pgClient.Handle().ExecContext(ctx, `
		DROP TABLE drop_cols;
		CREATE TABLE drop_cols (
			id SERIAL PRIMARY KEY NOT NULL,
			f1 int NOT NULL,
			f2 int NOT NULL
		);
	`)
	chkErr(t, err)

	// force the `Scan` method to be called to populate the caches
	id, err := pgClient.InsertDropCol(ctx, &models.DropCol{
		F1: 1,
		F2: 2,
	})
	chkErr(t, err)
	_, err = pgClient.GetDropCol(ctx, id)
	chkErr(t, err)

	// pull the rug out from under us
	_, err = pgClient.Handle().ExecContext(ctx, `ALTER TABLE drop_cols DROP COLUMN f1`)
	chkErr(t, err)

	// load the record again
	dc, err := pgClient.GetDropCol(ctx, id)
	chkErr(t, err)

	if dc.F1 != 0 {
		t.Fatalf("expected F1 to be the zero value, was %d", dc.F1)
	}

	if dc.F2 != 2 {
		t.Fatalf("expected F2 to be 2, was %d", dc.F2)
	}

	// pull the rug out from under us
	_, err = pgClient.Handle().ExecContext(ctx, `ALTER TABLE drop_cols ADD COLUMN f1 int NOT NULL DEFAULT 1`)
	chkErr(t, err)
}
