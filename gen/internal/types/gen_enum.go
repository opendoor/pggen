package types

import (
	"fmt"
	"strings"
	"text/template"

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
		goName := names.PgToGoName(pgTypeName)

		typeInfo := Info{
			Name:            goName,
			NullName:        "*" + goName,
			ScanNullName:    "Null" + goName,
			NullConvertFunc: convertCall("convertNull" + goName),
			NullSqlReceiver: func(v string) string {
				return fmt.Sprintf("&%s", v)
			},
			SqlReceiver: func(v string) string {
				return fmt.Sprintf("&ScanInto%s{value: &%s}", goName, v)
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

		evs := variantsToEnumEnumVars(variants)

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
			return pq.Array(ret)
		}()`, variable, variable)
}

type enumVar struct {
	GoName string
	PgName string
	Value  string
}

func variantsToEnumEnumVars(variants []string) []enumVar {
	varTab := map[string]bool{}
	for _, v := range variants {
		varTab[v] = true
	}

	var evs []enumVar
	for _, v := range variants {
		name := v
		if v == "" {
			// blank enum variants will cause a name clash
			proposed := "blank"
			for i := 0; varTab[proposed]; i++ {
				proposed = fmt.Sprintf("blank%d", i)
			}

			name = proposed
		}

		evs = append(evs, enumVar{
			GoName: names.PgToGoName(name),
			PgName: name,
			Value:  v,
		})
	}
	return evs
}

// Given the oid of a postgres type, return all the variants that
// that enum has.
func (r *Resolver) enumVariants(typeName string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT e.enumlabel
		FROM pg_type t
		JOIN pg_enum e
			ON (t.oid = e.enumtypid)
		WHERE t.typname = $1
		`, typeName)
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

type ScanInto{{ .TypeName }} struct {
	value *{{ .TypeName }}
}
func (s *ScanInto{{ .TypeName }}) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("unexpected NULL {{ .TypeName }}")
	}

	buff, ok := value.([]byte)
	if !ok {
		return fmt.Errorf(
			"ScanInto{{ .TypeName }}.Scan: expected a []byte",
		)
	}

	val, err := {{ .TypeName }}FromString(string(buff))
	if err != nil {
		return fmt.Errorf("Null{{ .TypeName }}.Scan: %s", err.Error())
	}

	*s.value = val

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
	buff, ok := value.([]byte)
	if !ok {
		return fmt.Errorf(
			"Null{{ .TypeName }}.Scan: expected a []byte",
		)
	}

	val, err := {{ .TypeName }}FromString(string(buff))
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