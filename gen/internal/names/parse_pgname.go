package names

import (
	"fmt"
	"regexp"
	"strings"
)

// file: parse_pgname.go
// This file implements a parser and pretty-printer for postgres names. The main
// thing that it does is split the schema name out.

// NOTE: copy-pasted from the includes package, but there is no great way
//       to share between the two packages.
var unquotedIdentRE = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_$]*$")

// PgName represents an identifier in postgres
type PgName struct {
	// The name of the schema. Never blank (always 'public' if no schema is specified).
	// Not quoted.
	Schema string
	// The inner name. Never blank. Not quoted.
	Name string
}

// String returns a quoted version of this identifier suitable for use in SQL
func (p *PgName) String() string {
	if p.Schema == "public" || p.Schema == "" {
		return toIdentPart(p.Name)
	}

	return fmt.Sprintf("%s.%s", toIdentPart(p.Schema), toIdentPart(p.Name))
}

func toIdentPart(part string) string {
	if unquotedIdentRE.Match([]byte(part)) {
		return part
	}

	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(part, `"`, `""`))
}

func ParsePgName(input string) (PgName, error) {
	var res PgName

	parts := strings.Split(input, ".")
	if len(parts) > 2 {
		return res, fmt.Errorf("parsing '%s': nested schemas are not supported", input)
	}

	var (
		err error
	)
	if len(parts) == 1 {
		res.Name, err = unquoteNamePart(strings.TrimSpace(parts[0]))
	} else if len(parts) == 2 {
		res.Schema, err = unquoteNamePart(strings.TrimSpace(parts[0]))
		if err == nil {
			res.Name, err = unquoteNamePart(strings.TrimSpace(parts[1]))
		}
	}
	if err != nil {
		return res, fmt.Errorf("parsing '%s': %s", input, err.Error())
	}
	if res.Schema == "" || res.Schema == `""` {
		res.Schema = "public"
	}

	return res, nil
}

func unquoteNamePart(part string) (string, error) {
	if len(part) == 0 {
		return "", fmt.Errorf("empty identifier")
	}

	// short circuit for the common case where there are no quotes
	if strings.IndexByte(part, '"') == -1 {
		return part, nil
	}

	if part[0] != '"' {
		return "", fmt.Errorf("identifiers cannot begin quoting in the middle")
	}

	if part[len(part)-1] != '"' || len(part) == 1 {
		return "", fmt.Errorf("unmatched quote")
	}

	part = part[1 : len(part)-1]
	// check again for an empty identifier now that we've trimmed the quotes
	if len(part) == 0 {
		return "", fmt.Errorf("empty identifier")
	}

	// handle escaped quotes, if any
	quoteIdx := strings.IndexByte(part, '"')
	if quoteIdx > -1 {
		var shrunkIdent strings.Builder
		rest := part
		for {
			if quoteIdx+1 >= len(rest) || rest[quoteIdx+1] != '"' {
				return "", fmt.Errorf("unmatched quote")
			}

			shrunkIdent.WriteString(rest[0:quoteIdx])
			shrunkIdent.WriteByte('"')
			if quoteIdx+2 < len(rest) {
				rest = rest[quoteIdx+2:]
			} else {
				rest = ""
				break
			}

			quoteIdx = strings.IndexByte(rest, '"')
			if quoteIdx == -1 {
				break
			}
		}
		shrunkIdent.WriteString(rest)
		part = shrunkIdent.String()
	}

	return part, nil
}
