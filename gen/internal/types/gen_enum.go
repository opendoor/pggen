package types

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/opendoor-labs/pggen/gen/internal/names"
)

func (r *Resolver) maybeEmitEnumType(
	pgTypeName string,
) (*Info, error) {
	// check if it is an enum
	variants, err := r.enumVariants(pgTypeName)
	if err != nil {
		return nil, fmt.Errorf(
			"unknown pg type: '%s': %v", pgTypeName, err,
		)
	}
	// if there are no variants, then it is not an enum
	if len(variants) > 0 {
		// PgTableToGoModel handles enums in non-public schemas a bit better than PgToGoName
		goName := names.PgTableToGoModel(pgTypeName)

		typeInfo := Info{
			Name:            goName,
			NullName:        "*" + goName,
			ScanNullName:    "Null" + goName,
			NullConvertFunc: convertCall("convertNull" + goName),
			NullSqlReceiver: func(v string) string {
				return fmt.Sprintf("&%s", v)
			},
			SqlReceiver: func(v string) string {
				return fmt.Sprintf("&%s", v)
			},
			SqlArgument:     stringizeWrap,
			NullSqlArgument: nullStringizeWrap,
			isEnum:          true,
		}

		if r.types.probe(typeInfo.Name) {
			// we've already generated a type for this enum, so we can
			// just return
			return &typeInfo, nil
		}

		r.registerImport(`"database/sql/driver"`)

		evs := variantsToEnumVars(variants)

		type enumGenCtx struct {
			TypeName string
			Variants []enumVar
		}
		genCtx := enumGenCtx{
			TypeName: typeInfo.Name,
			Variants: evs,
		}

		var typeDef strings.Builder
		err = enumTmpl.Execute(&typeDef, genCtx)
		if err != nil {
			return nil, err
		}
		var typeSig strings.Builder
		err = enumSigTmpl.Execute(&typeSig, genCtx)
		if err != nil {
			return nil, err
		}

		err = r.types.emitType(typeInfo.Name, typeSig.String(), typeDef.String())
		if err != nil {
			return nil, err
		}
		return &typeInfo, nil
	}

	return nil, fmt.Errorf("'%s' is not an enum type", pgTypeName)
}

func stringizeWrap(variable string) string {
	return fmt.Sprintf("%s.String()", variable)
}

func nullStringizeWrap(variable string) string {
	return fmt.Sprintf(`
		func() *string {
			if %s == nil {
				return nil
			}
			s := %s.String()
			return &s
		}()`, variable, variable)
}

func stringizeArrayWrap(variable string) string {
	return fmt.Sprintf(`
		func() interface{} {
			ret := make([]string, 0, len(%s))
			for _, e := range %s {
				ret = append(ret, e.String())
			}
			return pgtypes.Array(ret)
		}()`, variable, variable)
}

type enumVar struct {
	GoName string
	PgName string
	Value  string
}

func variantsToEnumVars(variants []string) []enumVar {
	varTab := map[string]bool{}
	for _, v := range variants {
		varTab[v] = true
	}

	var evs []enumVar
	variantGoNames := enumValuesToGoNames(variants)
	for i, v := range variants {
		goName := variantGoNames[i]

		evs = append(evs, enumVar{
			GoName: goName,
			PgName: v,
			Value:  strings.Replace(v, "`", "` + \"`\" + `", -1),
		})
	}
	return evs
}

// given a set of enum values, generate valid go names that can be used to refer to them
func enumValuesToGoNames(values []string) []string {
	// First we iterate the list and perform a best-effort conversion.
	// We strip out all the special chars and convert all spaces to underscores
	// then run names.PgToGoName over it.
	varTab := map[string]bool{}
	for _, v := range values {
		varTab[v] = true
	}
	goNames := make([]string, 0, len(values))
	for _, v := range values {
		name := v
		if v == "" {
			// blank enum variants will cause a name clash
			proposed := "blank"
			for i := 0; varTab[proposed]; i++ {
				proposed = fmt.Sprintf("blank%d", i)
			}

			name = proposed
		}

		var goName strings.Builder
		for i, r := range name {
			if unicode.IsSpace(r) {
				goName.WriteByte('_')
			} else if unicode.IsLetter(r) || r == '_' || (i > 0 && unicode.IsDigit(r)) {
				goName.WriteRune(r)
			}
		}
		goNames = append(goNames, names.PgToGoName(goName.String()))
	}

	// now we look for collisions and fixup any we find
	seen := map[string]int{}
	for i := range goNames {
		name := goNames[i]
		numSeen, inMap := seen[name]
		if inMap {
			goNames[i] = goNames[i] + strconv.Itoa(numSeen)
		} else {
			numSeen = 0 // technically not needed because of zero values, but I just want to be explicit
		}

		seen[name] = numSeen + 1
	}

	return goNames
}

// Given the oid of a postgres type, return all the variants that
// that enum has.
func (r *Resolver) enumVariants(typeName string) ([]string, error) {
	pgName, err := names.ParsePgName(typeName)
	if err != nil {
		return nil, fmt.Errorf("reflecting on potential enum '%s': %s", typeName, err.Error())
	}

	rows, err := r.db.Query(`
		SELECT e.enumlabel
		FROM pg_type t
		JOIN pg_enum e
			ON (t.oid = e.enumtypid)
		JOIN pg_namespace ns
			ON (t.typnamespace = ns.oid)
		WHERE ns.nspname = $1
		  AND t.typname = $2
		`, pgName.Schema, pgName.Name)
	if err != nil {
		return nil, err
	}

	variants := []string{}
	for rows.Next() {
		var variant string
		err = rows.Scan(&variant)
		if err != nil {
			return nil, err
		}
		variants = append(variants, variant)
	}
	return variants, nil
}

var enumSigTmpl = template.Must(template.New("enum-sig-tmpl").Parse(`
{{- range .Variants }}
{{ $.TypeName }}{{ .GoName }} {{ $.TypeName }} = "{{ .Value }}"
{{- end }}
`))

var enumTmpl = template.Must(template.New("enum-tmpl").Parse(`
type {{ .TypeName }} int
const (
{{- range .Variants }}
	{{ $.TypeName }}{{ .GoName }} {{ $.TypeName }} = iota
{{- end }}
)

func (t {{ .TypeName }}) String() string {
	switch t {
	{{- range .Variants }}
	case {{ $.TypeName }}{{ .GoName }}:
		return ` + "`" + `{{ .Value }}` + "`" + `
	{{- end }}
	default:
		panic(fmt.Sprintf("invalid {{ .TypeName }}: %d", t))
	}
}

func {{ .TypeName }}FromString(s string) ({{ .TypeName }}, error) {
	var zero {{ .TypeName }}

	switch s {
	{{- range .Variants }}
	case ` + "`" + `{{ .Value }}` + "`" + `:
		return {{ $.TypeName }}{{ .GoName }}, nil
	{{- end }}
	default:
		return zero, fmt.Errorf("{{ .TypeName }} unknown variant '%s'", s)
	}
}

func (s *{{ .TypeName }}) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("unexpected NULL {{ .TypeName }}")
	}

	var err error
	switch v := value.(type) {
	case []byte:
		*s, err = {{ .TypeName }}FromString(string(v))
	case string:
		*s, err = {{ .TypeName }}FromString(v)
	default:
		return fmt.Errorf("{{ .TypeName }}.Scan: unexpected type")
	}
	if err != nil {
		return fmt.Errorf("{{ .TypeName }}.Scan: %s", err.Error())
	}

	return nil
}

type Null{{ .TypeName }} struct {
	{{ .TypeName }} {{ .TypeName }}
	Valid bool
}
// Scan implements the sql.Scanner interface
func (n *Null{{ .TypeName }}) Scan(value interface{}) error {
	if value == nil {
		n.{{ .TypeName }}, n.Valid = {{ .TypeName }}(0), false
		return nil
	}

	var (
		val {{ .TypeName }}
		err error
	)
	switch v := value.(type) {
	case []byte:
		val, err = {{ .TypeName }}FromString(string(v))
	case string:
		val, err = {{ .TypeName }}FromString(v)
	default:
		return fmt.Errorf("Null{{ .TypeName }}.Scan: unexpected type")
	}
	if err != nil {
		return fmt.Errorf("Null{{ .TypeName }}.Scan: %s", err.Error())
	}

	n.Valid = true
	n.{{ .TypeName }} = val
	return nil
}
// Value implements the sql.Valuer interface
func (n Null{{ .TypeName }}) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.{{ .TypeName }}.String(), nil
}
func convertNull{{ .TypeName }}(v Null{{ .TypeName }}) *{{ .TypeName }} {
	if v.Valid {
		ret := {{ .TypeName }}(v.{{ .TypeName }})
		return &ret
	}
	return nil
}
`))
