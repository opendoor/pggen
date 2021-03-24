// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package names

import (
	"strings"
	"unicode"

	"github.com/jinzhu/inflection"
)

//
// Name Conversion Helpers
//
// These utility functions are used for munging the names associated
// with a given stored function or prepared statement.
//

func PgTableToGoModel(tableName string) string {
	parsed, err := ParsePgName(tableName)
	if err != nil {
		// fall back to treating the name as a single name
		// not idea, but we want to keep this infallable
		return PgToGoName(inflection.Singular(tableName))
	}

	if parsed.Schema == "public" {
		return PgToGoName(inflection.Singular(parsed.Name))
	}

	return PgToGoName(parsed.Schema) + "_" + PgToGoName(inflection.Singular(parsed.Name))
}

// Convert a postgres name (assumed to be snake_case)
// to a PascalCaseName
func PgToGoName(snakeName string) string {
	needsUpper := true

	var res strings.Builder
	for _, r := range snakeName {
		if unicode.IsSpace(r) {
			continue
		} else if r == '_' {
			needsUpper = true
		} else if unicode.IsPunct(r) {
			continue
		} else if needsUpper {
			res.WriteRune(unicode.ToUpper(r))
			needsUpper = false
		} else {
			res.WriteRune(r)
		}
	}

	return res.String()
}
