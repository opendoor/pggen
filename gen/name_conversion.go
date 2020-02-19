package gen

import (
	"strings"
	"unicode"
)

//
// Name Conversion Helpers
//
// These utility functions are used for munging the names associated
// with a given stored function or prepared statement.
//

// Convert a postgres name (assumed to be snake_case)
// to a PascalCaseName
func pgToGoName(snakeName string) string {
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
