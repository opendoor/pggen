package test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/opendoor/pggen"
	"github.com/opendoor/pggen/cmd/pggen/test/global_ts_models"
	"github.com/opendoor/pggen/cmd/pggen/test/models"
)

func TestTimestampsBoth(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := txClient.InsertTimestampsBoth(ctx, &models.TimestampsBoth{
		Payload: &blah,
	})
	chkErr(t, err)

	fetched, err := txClient.GetTimestampsBoth(ctx, id)
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
	mask.Set(models.TimestampsBothIdFieldIndex, true)
	mask.Set(models.TimestampsBothPayloadFieldIndex, true)
	id, err = txClient.UpdateTimestampsBoth(ctx, fetched, mask)
	chkErr(t, err)

	fetched, err = txClient.GetTimestampsBoth(ctx, id)
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
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	now := time.Now().UTC()
	blah := "blah"
	id, err := txClient.InsertTimestampsJustCreated(ctx, &models.TimestampsJustCreated{
		Payload: &blah,
	})
	chkErr(t, err)

	fetched, err := txClient.GetTimestampsJustCreated(ctx, id)
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
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := txClient.InsertTimestampsJustUpdated(ctx, &models.TimestampsJustUpdated{
		Payload: &blah,
	})
	chkErr(t, err)

	fetched, err := txClient.GetTimestampsJustUpdated(ctx, id)
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
	mask.Set(models.TimestampsJustUpdatedPayloadFieldIndex, true)
	mask.Set(models.TimestampsJustUpdatedIdFieldIndex, true)
	now = time.Now()
	id, err = txClient.UpdateTimestampsJustUpdated(ctx, fetched, mask)
	chkErr(t, err)

	fetched, err = txClient.GetTimestampsJustUpdated(ctx, id)
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
	dbClient := global_ts_models.NewPGClient(pgClient.Handle().(*sql.DB))
	txClient, err := dbClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := txClient.InsertTimestampsGlobal(ctx, &global_ts_models.TimestampsGlobal{
		Payload: &blah,
	})
	chkErr(t, err)

	fetched, err := txClient.GetTimestampsGlobal(ctx, id)
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

func TestUpsertCreateTimestamps(t *testing.T) {
	dbClient := global_ts_models.NewPGClient(pgClient.Handle().(*sql.DB))
	txClient, err := dbClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := txClient.UpsertTimestampsGlobal(ctx, &global_ts_models.TimestampsGlobal{
		Payload: &blah,
	}, nil, global_ts_models.TimestampsGlobalAllFields)
	chkErr(t, err)

	fetched, err := txClient.GetTimestampsGlobal(ctx, id)
	chkErr(t, err)

	if now.Add(-time.Second).After(fetched.UpdatedAt) {
		t.Fatalf("1 expected timestamp within about a second")
	}
	if now.Add(time.Second).Before(fetched.UpdatedAt) {
		t.Fatalf("2 expected timestamp within about a second")
	}
}

func TestUpsertUpdateTimestamps(t *testing.T) {
	dbClient := global_ts_models.NewPGClient(pgClient.Handle().(*sql.DB))
	txClient, err := dbClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	blah := "blah"
	id, err := txClient.UpsertTimestampsGlobal(ctx, &global_ts_models.TimestampsGlobal{
		Payload: &blah,
	}, nil, global_ts_models.TimestampsGlobalAllFields)
	chkErr(t, err)

	fetched, err := txClient.GetTimestampsGlobal(ctx, id)
	chkErr(t, err)

	updateMask := pggen.NewFieldSet(global_ts_models.TimestampsGlobalMaxFieldIndex)
	updateMask.Set(global_ts_models.TimestampsGlobalPayloadFieldIndex, true)
	updateMask.Set(global_ts_models.TimestampsGlobalIdFieldIndex, true)

	time.Sleep(time.Millisecond * 50)

	dip := "dip"
	fetched.Payload = &dip
	_, err = txClient.UpsertTimestampsGlobal(ctx, fetched, nil, updateMask, pggen.UpsertUsePkey)
	chkErr(t, err)

	refetched, err := txClient.GetTimestampsGlobal(ctx, id)
	chkErr(t, err)

	if fetched.UpdatedAt.Equal(refetched.UpdatedAt) {
		t.Fatal("expected some change in timestamp")
	}
	if fetched.UpdatedAt.After(refetched.UpdatedAt) {
		t.Fatal("refetched should be later")
	}
}

func TestSoftDeleteCRUD(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	id, err := txClient.InsertSoftDeletable(ctx, &models.SoftDeletable{
		Value: "Some Ping", // if Charolotte was a programmer
	})
	chkErr(t, err)

	// should be fine to fetch it now
	_, err = txClient.GetSoftDeletable(ctx, id)
	chkErr(t, err)

	err = txClient.DeleteSoftDeletable(ctx, id)
	chkErr(t, err)

	_, err = txClient.GetSoftDeletable(ctx, id)
	if err == nil || !pggen.IsNotFoundError(err) {
		t.Fatal("expected the record not to be found (get)")
	}

	_, err = txClient.ListSoftDeletable(ctx, []int64{id})
	if err == nil || !pggen.IsNotFoundError(err) {
		t.Fatal("expected the record not to be found (list)")
	}

	sneakyFetched, err := txClient.GetSoftDeletableAnyway(ctx, id)
	chkErr(t, err)
	if len(sneakyFetched) != 1 || sneakyFetched[0].Value != "Some Ping" || sneakyFetched[0].DeletedTs == nil {
		t.Fatalf("expected the record to still be there and have the right data: %v\n", sneakyFetched)
	}

	err = txClient.DeleteSoftDeletable(ctx, id, pggen.DeleteDoHardDelete)
	chkErr(t, err)

	sneakyFetched, err = txClient.GetSoftDeletableAnyway(ctx, id)
	chkErr(t, err)
	if len(sneakyFetched) != 0 {
		t.Fatal("expected the data to be proper gone at this point")
	}
}

func TestSoftDeleteIncludes(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	//
	// setup some data
	//

	rootID, err := txClient.InsertSoftDeletable(ctx, &models.SoftDeletable{
		Value: "root",
	})
	chkErr(t, err)

	leaf1ID, err := txClient.InsertDeletableLeaf(ctx, &models.DeletableLeaf{
		Value:           "leaf-1",
		SoftDeletableId: rootID,
	})
	chkErr(t, err)

	leaf2ID, err := txClient.InsertDeletableLeaf(ctx, &models.DeletableLeaf{
		Value:           "leaf-2",
		SoftDeletableId: rootID,
	})
	chkErr(t, err)

	//
	// soft delete part of the tree and then fill it in via includes
	//

	err = txClient.DeleteDeletableLeaf(ctx, leaf2ID)
	chkErr(t, err)

	root, err := txClient.GetSoftDeletable(ctx, rootID)
	chkErr(t, err)

	err = txClient.SoftDeletableFillIncludes(ctx, root, models.SoftDeletableAllIncludes)
	chkErr(t, err)

	if len(root.DeletableLeafs) != 1 {
		t.Fatalf("expected one child, got: %v\n", root.DeletableLeafs)
	}

	err = txClient.DeleteSoftDeletable(ctx, rootID)
	chkErr(t, err)

	leaf1, err := txClient.GetDeletableLeaf(ctx, leaf1ID)
	chkErr(t, err)

	err = txClient.DeletableLeafFillIncludes(ctx, leaf1, models.DeletableLeafAllIncludes)
	chkErr(t, err)

	if leaf1.SoftDeletable != nil {
		t.Fatal("parent should be deleted")
	}
}

func TestGlobalDeletedAt(t *testing.T) {
	dbClient := global_ts_models.NewPGClient(pgClient.Handle().(*sql.DB))
	txClient, err := dbClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	id, err := txClient.InsertSoftDeletable(ctx, &global_ts_models.SoftDeletable{
		Value: "Some Ping", // if Charolotte was a programmer
	})
	chkErr(t, err)

	err = txClient.DeleteSoftDeletable(ctx, id)
	chkErr(t, err)

	_, err = txClient.GetSoftDeletable(ctx, id)
	if err == nil || !pggen.IsNotFoundError(err) {
		t.Fatal("expected the record not to be found (get)")
	}
}
