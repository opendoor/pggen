// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package test

import (
	"github.com/opendoor-labs/pggen/cmd/pggen/test/models"
	"testing"
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
