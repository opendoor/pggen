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
	"github.com/opendoor-labs/pggen/cmd/pggen/test/global_ts_shims"
	"github.com/opendoor-labs/pggen/include"
)

func TestInsertSmallEntity(t *testing.T) {
	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	entity := db_shims.SmallEntity{
		Anint: 129,
	}

	id, err := pgClient.InsertSmallEntity(ctx, &entity)
	chkErr(t, err)

	entity.Id = id

	fetched, err := pgClient.GetSmallEntityByAnint(ctx, entity.Anint)
	chkErr(t, err)

	if !reflect.DeepEqual(entity, fetched[0]) {
		t.Fatalf("%#v != %#v", entity, fetched[0])
	}
}

func TestSmallEntityBulk(t *testing.T) {
	// tests BulkInsert and List

	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
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

	_, err = pgClient.BulkInsertSmallEntity(ctx, entities)
	chkErr(t, err)

	fetched, err := pgClient.GetSmallEntityByAnint(ctx, 1232)
	chkErr(t, err)

	// the ids won't match up, so just length check for now
	if len(fetched) != len(entities) {
		t.Fatalf("not %v ~= %v", entities, fetched)
	}

	ids := make([]int64, len(fetched))[:0]
	for _, ent := range fetched {
		ids = append(ids, ent.Id)
	}

	fetched2, err := pgClient.ListSmallEntity(ctx, ids)
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

	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
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

	_, err = pgClient.BulkInsertSmallEntity(ctx, entities)
	chkErr(t, err)

	fetched, err := pgClient.GetSmallEntityByAnint(ctx, 1232)
	chkErr(t, err)

	noOpBitset := pggen.NewFieldSet(2)
	noOpBitset.Set(db_shims.SmallEntityIdFieldIndex, true)

	fetched[0].Anint = 34
	id, err := pgClient.UpdateSmallEntity(ctx, &fetched[0], noOpBitset)
	if err != nil {
		t.Fatal(err)
	}
	if id != fetched[0].Id {
		t.Fatalf("update id mismatch")
	}

	e0, err := pgClient.GetSmallEntityByID(ctx, fetched[0].Id)
	chkErr(t, err)
	if e0[0].Anint == 34 {
		t.Fatalf("unexpected update")
	}

	fetched[1].Anint = 42
	id, err = pgClient.UpdateSmallEntity(ctx, &fetched[1], db_shims.SmallEntityAllFields)
	chkErr(t, err)
	if id != fetched[1].Id {
		t.Fatalf("id mismatch (passed in %d, got back %d)", fetched[1].Id, id)
	}
	e1, err := pgClient.GetSmallEntity(ctx, fetched[1].Id)
	chkErr(t, err)
	if e1.Anint != 42 {
		t.Fatalf("update failed e1 = %#v", e1)
	}
}

func TestSmallEntityCreateDelete(t *testing.T) {
	// tests BulkInsert, BulkDelete and Delete

	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
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

	ids, err := pgClient.BulkInsertSmallEntity(ctx, entities)
	chkErr(t, err)

	chkErr(t, pgClient.BulkDeleteSmallEntity(ctx, ids[:2]))

	fetched, err := pgClient.GetSmallEntityByAnint(ctx, 232)
	chkErr(t, err)

	if len(fetched) != 4 {
		t.Fatalf("expected 2 entities to be deleted")
	}

	chkErr(t, pgClient.DeleteSmallEntity(ctx, ids[4]))
	fetched, err = pgClient.GetSmallEntityByAnint(ctx, 232)
	chkErr(t, err)

	if len(fetched) != 3 {
		t.Fatalf(
			"expected 3 entities to be deleted (%d present)",
			len(fetched),
		)
	}
}

func TestFillAll(t *testing.T) {
	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	entityID, err := pgClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 129,
	})
	chkErr(t, err)

	foo := "foo"
	attachmentID1, err := pgClient.InsertAttachment(ctx, &db_shims.Attachment{
		SmallEntityId: entityID,
		Value:         &foo,
	})
	chkErr(t, err)

	bar := "bar"
	attachmentID2, err := pgClient.InsertAttachment(ctx, &db_shims.Attachment{
		SmallEntityId: entityID,
		Value:         &bar,
	})
	chkErr(t, err)

	aTime := time.Unix(5432553, 0)
	_, err = pgClient.InsertSingleAttachment(ctx, &db_shims.SingleAttachment{
		SmallEntityId: entityID,
		CreatedAt:     aTime,
	})
	chkErr(t, err)

	e, err := pgClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = pgClient.SmallEntityFillIncludes(ctx, e, db_shims.SmallEntityAllIncludes)
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
	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	entityID, err := pgClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 129,
	})
	chkErr(t, err)

	foo := "foo"
	attachmentID1, err := pgClient.InsertAttachment(ctx, &db_shims.Attachment{
		SmallEntityId: entityID,
		Value:         &foo,
	})
	chkErr(t, err)

	aTime := time.Unix(5432553, 0)
	singleAttachmentID, err := pgClient.InsertSingleAttachment(ctx, &db_shims.SingleAttachment{
		SmallEntityId: entityID,
		CreatedAt:     aTime,
	})
	chkErr(t, err)

	// we are going to use include specs to load the attachment, but no the SingleAttachment
	includes := include.Must(include.Parse("small_entities.attachments"))
	smallEntity, err := pgClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = pgClient.SmallEntityFillIncludes(ctx, smallEntity, includes)
	chkErr(t, err)

	if smallEntity.Attachments[0].Id != attachmentID1 {
		t.Fatalf("failed to fetch attachment")
	}
	if smallEntity.SingleAttachment != nil {
		t.Fatalf("fetched single attachment when it wasn't in the include set")
	}

	// now do load the single_attachments
	includes = include.Must(include.Parse("small_entities.{attachments, single_attachments}"))
	smallEntity, err = pgClient.GetSmallEntity(ctx, entityID)
	chkErr(t, err)
	err = pgClient.SmallEntityFillIncludes(ctx, smallEntity, includes)
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
	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	var nineteen int64 = 19
	funnyID, err := pgClient.InsertWeirdNaMe(ctx, &db_shims.WeirdNaMe{
		WearetalkingReallyBadstyle: 1923,
		GotWhitespace:              "yes",
		ButWhyTho:                  &nineteen,
	})
	chkErr(t, err)

	funny, err := pgClient.GetWeirdNaMe(ctx, funnyID)
	chkErr(t, err)

	funny.GotWhitespace = "no"

	funnyID, err = pgClient.UpdateWeirdNaMe(
		ctx, funny, db_shims.WeirdNaMeAllFields)
	chkErr(t, err)

	funny, err = pgClient.GetWeirdNaMe(ctx, funnyID)
	chkErr(t, err)

	if funny.GotWhitespace != "no" {
		t.Fatalf("update failed")
	}

	kidID, err := pgClient.InsertWeirdKid(ctx, &db_shims.WeirdKid{
		Daddy: funny.Evenidisweird,
	})
	chkErr(t, err)
	err = pgClient.WeirdNaMeFillIncludes(ctx, funny, db_shims.WeirdNaMeAllIncludes)
	chkErr(t, err)

	err = pgClient.DeleteWeirdKid(ctx, kidID)
	chkErr(t, err)

	err = pgClient.DeleteWeirdNaMe(ctx, funny.Evenidisweird)
	chkErr(t, err)
}

func TestArrayMembers(t *testing.T) {
	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	var nineteen int64 = 19
	id, err := pgClient.InsertArrayMember(ctx, &db_shims.ArrayMember{
		TextArray: []string{"foo", "bar"},
		IntArray:  []*int64{&nineteen, nil},
	})
	chkErr(t, err)

	arrayMember, err := pgClient.GetArrayMember(ctx, id)
	chkErr(t, err)

	_, err = pgClient.UpdateArrayMember(
		ctx, arrayMember, db_shims.ArrayMemberAllFields)
	chkErr(t, err)
}

func TestMaxFieldIndex(t *testing.T) {
	if db_shims.SmallEntityMaxFieldIndex != db_shims.SmallEntityAnintFieldIndex {
		t.Fatalf("max field index mismatch")
	}
}

func TestTimestampsBoth(t *testing.T) {
	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := pgClient.InsertTimestampsBoth(ctx, &db_shims.TimestampsBoth{
		Payload: &blah,
	})
	chkErr(t, err)

	fetched, err := pgClient.GetTimestampsBoth(ctx, id)
	chkErr(t, err)

	if !fetched.UpdatedAt.Equal(*fetched.CreatedAt) {
		t.Fatalf("expeced same updated and created timestamps")
	}

	if now.Add(-time.Second).After(fetched.UpdatedAt) {
		t.Fatalf("1 expected timestamp within about a second")
	}
	if now.Add(time.Second).Before(fetched.UpdatedAt) {
		t.Fatalf("2 expected timestamp within about a second")
	}

	time.Sleep(time.Millisecond * 50)

	oldUpdatedAt := fetched.UpdatedAt

	mask := pggen.NewFieldSet(10)
	mask.Set(db_shims.TimestampsBothIdFieldIndex, true)
	mask.Set(db_shims.TimestampsBothPayloadFieldIndex, true)
	id, err = pgClient.UpdateTimestampsBoth(ctx, fetched, mask)
	chkErr(t, err)

	fetched, err = pgClient.GetTimestampsBoth(ctx, id)
	chkErr(t, err)

	if now.Add(-time.Second).After(fetched.UpdatedAt) {
		t.Fatalf(
			"3 expected timestamp within about a second (%s, %s)",
			now.Add(-time.Second).String(),
			fetched.UpdatedAt.String(),
		)
	}
	if now.Add(time.Second).Before(fetched.UpdatedAt) {
		t.Fatalf("4 expected timestamp within about a second")
	}

	if oldUpdatedAt.Equal(fetched.UpdatedAt) {
		t.Fatalf(
			"expected the timestamp to change (%s, %s)",
			oldUpdatedAt.String(),
			fetched.UpdatedAt.String(),
		)
	}
}

func TestTimestampsJustCreated(t *testing.T) {
	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	now := time.Now().UTC()
	blah := "blah"
	id, err := pgClient.InsertTimestampsJustCreated(ctx, &db_shims.TimestampsJustCreated{
		Payload: &blah,
	})
	chkErr(t, err)

	fetched, err := pgClient.GetTimestampsJustCreated(ctx, id)
	chkErr(t, err)

	if now.Add(-time.Second).After(fetched.MadeAt) {
		t.Fatalf(
			"1 expected timestamp within about a second (now = %s, made at = %s)",
			now,
			fetched.MadeAt,
		)
	}
	if now.Add(time.Second).Before(fetched.MadeAt) {
		t.Fatalf("2 expected timestamp within about a second")
	}
}

func TestTimestampsJustUpdated(t *testing.T) {
	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := pgClient.InsertTimestampsJustUpdated(ctx, &db_shims.TimestampsJustUpdated{
		Payload: &blah,
	})
	chkErr(t, err)

	fetched, err := pgClient.GetTimestampsJustUpdated(ctx, id)
	chkErr(t, err)

	if now.Add(-time.Second).After(*fetched.LastTouched) {
		t.Fatalf("1 expected timestamp within about a second")
	}
	if now.Add(time.Second).Before(*fetched.LastTouched) {
		t.Fatalf("2 expected timestamp within about a second")
	}

	oldUpdatedAt := *fetched.LastTouched

	time.Sleep(time.Millisecond * 1250)

	mask := pggen.NewFieldSet(10)
	mask.Set(db_shims.TimestampsJustUpdatedPayloadFieldIndex, true)
	mask.Set(db_shims.TimestampsJustUpdatedIdFieldIndex, true)
	now = time.Now()
	id, err = pgClient.UpdateTimestampsJustUpdated(ctx, fetched, mask)
	chkErr(t, err)

	fetched, err = pgClient.GetTimestampsJustUpdated(ctx, id)
	chkErr(t, err)

	if now.Add(-time.Second).After(*fetched.LastTouched) {
		t.Fatalf("3 expected timestamp within about a second")
	}
	if now.Add(time.Second).Before(*fetched.LastTouched) {
		t.Fatalf("4 expected timestamp within about a second")
	}

	if oldUpdatedAt.Equal(*fetched.LastTouched) {
		t.Fatalf(
			"expected the timestamp to change (%s, %s)",
			oldUpdatedAt.String(),
			fetched.LastTouched.String(),
		)
	}
}

func TestTimestampsGlobal(t *testing.T) {
	dbClient := global_ts_shims.NewPGClient(pgClient.Handle().(*sql.DB))
	err := dbClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = dbClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := dbClient.InsertTimestampsGlobal(ctx, &global_ts_shims.TimestampsGlobal{
		Payload: &blah,
	})
	chkErr(t, err)

	fetched, err := dbClient.GetTimestampsGlobal(ctx, id)
	chkErr(t, err)

	if !fetched.UpdatedAt.Equal(*fetched.CreatedAt) {
		t.Fatalf("expeced same updated and created timestamps")
	}

	if now.Add(-time.Second).After(fetched.UpdatedAt) {
		t.Fatalf("1 expected timestamp within about a second")
	}
	if now.Add(time.Second).Before(fetched.UpdatedAt) {
		t.Fatalf("2 expected timestamp within about a second")
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

	err = pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	id, err := pgClient.InsertColOrder(ctx, &db_shims.ColOrder{
		Field1: "foo",
		Field2: 1,
		Field3: 2,
	})
	chkErr(t, err)

	rec, err := pgClient.GetColOrder(ctx, id)
	chkErr(t, err)

	if rec.Field1 != "foo" || rec.Field2 != 1 || rec.Field3 != 2 {
		t.Fatalf("rec = %#v", rec)
	}
}
