package test

import (
	"testing"

	"github.com/opendoor-labs/pggen/cmd/pggen/test/db_shims"
)

func TestTxRollback(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	seID, err := txClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 19,
	})
	chkErr(t, err)

	fetched, err := txClient.GetSmallEntity(ctx, seID)
	chkErr(t, err)
	if fetched == nil {
		t.Fatalf("expected to fetch small entity")
	}

	err = txClient.Rollback()
	chkErr(t, err)

	_, err = pgClient.GetSmallEntity(ctx, seID)
	if err == nil {
		t.Fatalf("expected not to fetch small entity")
	}
}

func TestTxCommit(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	doRollback := true
	defer func() {
		if doRollback {
			_ = txClient.Rollback()
		}
	}()

	seID, err := txClient.InsertSmallEntity(ctx, &db_shims.SmallEntity{
		Anint: 19,
	})
	chkErr(t, err)

	fetched, err := txClient.GetSmallEntity(ctx, seID)
	chkErr(t, err)
	if fetched == nil {
		t.Fatalf("expected to fetch small entity")
	}

	err = txClient.Commit()
	chkErr(t, err)
	doRollback = false

	fetched, err = pgClient.GetSmallEntity(ctx, seID)
	chkErr(t, err)
	if fetched == nil {
		t.Fatalf("expected to fetch small entity")
	}

	err = pgClient.DeleteSmallEntity(ctx, seID)
	chkErr(t, err)
}
