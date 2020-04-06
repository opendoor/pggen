package test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/cmd/pggen/test/db_shims"
	"github.com/opendoor-labs/pggen/cmd/pggen/test/global_ts_shims"
)

func TestTimestampsBoth(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := txClient.InsertTimestampsBoth(ctx, &db_shims.TimestampsBoth{
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
	mask.Set(db_shims.TimestampsBothIdFieldIndex, true)
	mask.Set(db_shims.TimestampsBothPayloadFieldIndex, true)
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
	id, err := txClient.InsertTimestampsJustCreated(ctx, &db_shims.TimestampsJustCreated{
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
	id, err := txClient.InsertTimestampsJustUpdated(ctx, &db_shims.TimestampsJustUpdated{
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
	mask.Set(db_shims.TimestampsJustUpdatedPayloadFieldIndex, true)
	mask.Set(db_shims.TimestampsJustUpdatedIdFieldIndex, true)
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
	dbClient := global_ts_shims.NewPGClient(pgClient.Handle().(*sql.DB))
	txClient, err := dbClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := txClient.InsertTimestampsGlobal(ctx, &global_ts_shims.TimestampsGlobal{
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
	dbClient := global_ts_shims.NewPGClient(pgClient.Handle().(*sql.DB))
	txClient, err := dbClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	now := time.Now()
	blah := "blah"
	id, err := txClient.UpsertTimestampsGlobal(ctx, &global_ts_shims.TimestampsGlobal{
		Payload: &blah,
	}, nil, global_ts_shims.TimestampsGlobalAllFields)
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
	dbClient := global_ts_shims.NewPGClient(pgClient.Handle().(*sql.DB))
	txClient, err := dbClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	blah := "blah"
	id, err := txClient.UpsertTimestampsGlobal(ctx, &global_ts_shims.TimestampsGlobal{
		Payload: &blah,
	}, nil, global_ts_shims.TimestampsGlobalAllFields)
	chkErr(t, err)

	fetched, err := txClient.GetTimestampsGlobal(ctx, id)
	chkErr(t, err)

	updateMask := pggen.NewFieldSet(global_ts_shims.TimestampsGlobalMaxFieldIndex)
	updateMask.Set(global_ts_shims.TimestampsGlobalPayloadFieldIndex, true)
	updateMask.Set(global_ts_shims.TimestampsGlobalIdFieldIndex, true)

	time.Sleep(time.Millisecond * 50)

	dip := "dip"
	fetched.Payload = &dip
	_, err = txClient.UpsertTimestampsGlobal(ctx, fetched, nil, updateMask)
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
