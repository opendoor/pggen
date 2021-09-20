package test

import (
	"testing"

	"github.com/opendoor/pggen/cmd/pggen/test/models"
)

func TestConnSmoke(t *testing.T) {
	connClient, err := pgClient.Conn(ctx)
	chkErr(t, err)

	id, err := connClient.InsertSmallEntity(ctx, &models.SmallEntity{
		Anint: 9735,
	})
	chkErr(t, err)
	entity, err := connClient.GetSmallEntity(ctx, id)
	chkErr(t, err)
	if entity.Anint != 9735 {
		t.Fatal("bad value")
	}
}
