// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package test

import (
	"github.com/opendoor/pggen/cmd/pggen/test/models"
)

// type assertions that the PGClient and TxPGClient types satisfy the DBQueries
// interface.
var (
	_ models.DBQueries = &models.PGClient{}
	_ models.DBQueries = &models.TxPGClient{}
)
