package test

import (
	"testing"
)

func TestStmtInsertSmallEntity(t *testing.T) {
	err := pgClient.BeginTx(ctx, nil)
	chkErr(t, err)
	defer func() {
		_ = pgClient.Rollback()
	}()

	res, err := pgClient.StmtInsertSmallEntity(ctx, 719)
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

	smallEntities, err := pgClient.GetSmallEntityByAnint(ctx, 719)
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

// TODO: once #20 is done, test inserting null enum values using the
//       NullEnumType generated type
