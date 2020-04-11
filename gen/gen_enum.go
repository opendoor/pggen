package gen

import (
	"fmt"
	"strings"
	"text/template"
)

func (g *Generator) maybeEmitEnumType(
	pgTypeName string,
) (*goTypeInfo, error) {
	// check if it is an enum
	variants, err := g.enumVariants(pgTypeName)
	if err != nil {
		return nil, fmt.Errorf(
			"unknown pg type: '%s': %v", pgTypeName, err,
		)
	}
	// if there are no variants, then it is not an enum
	if len(variants) > 0 {
		goName := pgToGoName(pgTypeName)

		typeInfo := goTypeInfo{
			Name:            goName,
			NullName:        "*" + goName,
			ScanNullName:    "Null" + goName,
			NullConvertFunc: convertCall("convertNull" + goName),
			SqlReceiver:     refWrap,
			SqlArgument:     stringizeWrap,
			isEnum:          true,
		}

		if g.types.probe(typeInfo.Name) {
			// we've already generated a type for this enum, so we can
			// just return
			return &typeInfo, nil
		}

		g.imports[`"database/sql/driver"`] = true

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

		err = g.types.emitType(typeInfo.Name, typeSig.String(), typeDef.String())
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
			GoName: pgToGoName(name),
			PgName: name,
			Value:  v,
		})
	}
	return evs
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
