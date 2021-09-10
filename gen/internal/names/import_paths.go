package names

// file: import_paths.go
// This file contains code to validate go import paths. Those are sort of names right?

import (
	"fmt"
	"regexp"
	"strings"
)

var quotedStringRE = regexp.MustCompile(`^"[^"]+"$`)
var aliasedQuotedStringRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*\s+"[^"]+"$`)

// ValidateImportPath checks the given path for issues and
// returns an error if it finds any.
func ValidateImportPath(path string) error {
	if strings.IndexByte(path, ' ') == -1 {
		if !quotedStringRE.Match([]byte(path)) {
			return fmt.Errorf("import paths without spaces in them should be quoted strings")
		}
	} else if !aliasedQuotedStringRE.Match([]byte(path)) {
		// if there is a space in the path it should be an identifier followed
		// by a quoted string.
		return fmt.Errorf("import paths containing spaces should be aliased quoted strings")
	}

	return nil
}
