package test

import (
	"database/sql"
	"testing"
)

func TestStmtInsertSmallEntity(t *testing.T) {
	txClient := newTx(t)
	defer func() {
		txClient.DB.(*sql.Tx).Rollback()
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
