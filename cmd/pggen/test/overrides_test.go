package test

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/opendoor/pggen/cmd/pggen/test/overridden_models"
)

func TestOverriddenGetTimes(t *testing.T) {
	overriddenPgClient := overridden_models.NewPGClient(pgClient.Handle().(*sql.DB))

	res, err := overriddenPgClient.GetTimes(ctx)
	chkErr(t, err)

	tsStr := res[0].TsFieldNotNull.String()
	if !(tsStr == "1999-01-08 04:05:06 +0000 +0000" || tsStr == "1999-01-08 04:05:06 +0000 UTC") {
		t.Fatalf("bad ts field: '%s'", res[0].TsFieldNotNull.String())
	}

	timeTy := reflect.TypeOf(&time.Time{})
	tsFieldTy := reflect.TypeOf(res[0].TsField)
	if tsFieldTy != timeTy {
		t.Fatalf("type mismatch")
	}
}

func TestOverriddenSelectUUID(t *testing.T) {
	overriddenPgClient := overridden_models.NewPGClient(pgClient.Handle().(*sql.DB))

	res, err := overriddenPgClient.SelectUuid(ctx)
	chkErr(t, err)

	uuidTy := reflect.TypeOf(&uuid.UUID{})
	selectFieldTy := reflect.TypeOf(res[0].UuidField)

	if uuidTy != selectFieldTy {
		t.Fatalf("type mismatch")
	}
}
