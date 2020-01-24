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

// Convert a snake_case name to a PascalCaseName
func snakeToPascal(snakeName string) string {
	return snakeToMixed(snakeName, true)
}

func snakeToCamel(snakeName string) string {
	return snakeToMixed(snakeName, false)
}

func snakeToMixed(snakeName string, needsUpper bool) string {
	var res strings.Builder
	for _, r := range snakeName {
		if r == '_' {
			needsUpper = true
		} else if needsUpper {
			res.WriteRune(unicode.ToUpper(r))
			needsUpper = false
		} else {
			res.WriteRune(r)
		}
	}

	return res.String()
}
