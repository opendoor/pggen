// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
// ensure-schema runs the given schema file against $DB_URL
package main

import (
	"log"
	"os"

	"github.com/opendoor/pggen/tools/ensure-schema/lib"
)

func main() {
	err := lib.PopulateDB(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
}
