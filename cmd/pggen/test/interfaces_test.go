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
