package test

import (
	"testing"

	"github.com/opendoor/pggen/cmd/pggen/test/models"
)

func TestStmtInsertSmallEntity(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	res, err := txClient.StmtInsertSmallEntity(ctx, 719)
	if err != nil {
		t.Fatal(err)
	}

	nrows, err := res.RowsAffected()
	if err != nil {
		t.Fatal(err)
	}
	if nrows != 1 {
		t.Fatalf("expected 1 row to be affected (actually %d)", nrows)
	}

	smallEntities, err := txClient.GetSmallEntityByAnint(ctx, 719)
	if err != nil {
		t.Fatal(err)
	}
	if len(smallEntities) != 1 {
		t.Fatalf("Expected 1 result (actually %d)", len(smallEntities))
	}
	if smallEntities[0].Anint != 719 {
		t.Fatalf("Unexpected entity (Anint = %d)", smallEntities[0].Anint)
	}
}

func TestEnumInsertStatement(t *testing.T) {
	txClient, err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = txClient.Rollback()
	}()

	_, err = txClient.EnumInsertStmt(ctx, models.FunkyNameEnumFoo)
	chkErr(t, err)
}

// TODO: once #20 is done, test inserting null enum values using the
//       NullEnumType generated type
