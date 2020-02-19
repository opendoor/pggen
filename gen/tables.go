package gen

import (
	"fmt"
	"strings"
	"text/template"
)

func (g *Generator) typeInfoOf(pgTypeName string) (goTypeInfo, error) {
	typeInfo, ok := kPgType2GoType[pgTypeName]
	if ok {
		if typeInfo.Name == "time.Time" {
			g.imports[`"time"`] = true
		}
		if typeInfo.Name == "uuid.UUID" {
			g.imports[`uuid "github.com/satori/go.uuid"`] = true
		}
		return typeInfo, nil
	}

	// check if it is an enum
	variants, err := g.enumVariants(pgTypeName)
	if err != nil {
		return kBogusGoTypeInfo, fmt.Errorf(
			"unknown pg type: '%s': %v", pgTypeName, err,
		)
	}
	// if there are no variants, then it is not an enum
	if len(variants) > 0 {
		typeInfo := goTypeInfo{
			Name:        pgToGoName(pgTypeName),
			NullName:    "*" + pgToGoName(pgTypeName),
			SqlReceiver: refWrap,
		}

		if g.types.probe(typeInfo.Name) {
			// we've already generated a type for this enum, so we can
			// just return
			return typeInfo, nil
		}

		enumSigTmpl := template.Must(template.New("enum-tmpl").Parse(`
{{- range (index . "Variants") }}
{{ index $ "TypeName" }}{{ .GoName }} {{ index $ "TypeName" }} = "{{ .PgName }}"
{{- end }}
`))

		enumTmpl := template.Must(template.New("enum-tmpl").Parse(`
type {{ index . "TypeName" }} string
const (
{{- range (index . "Variants") }}
	{{ index $ "TypeName" }}{{ .GoName }} {{ index $ "TypeName" }} = "{{ .PgName }}"
{{- end }}
)

`))

		type enumVar struct {
			GoName string
			PgName string
		}
		var evs []enumVar
		for _, v := range variants {
			evs = append(evs, enumVar{
				GoName: pgToGoName(v),
				PgName: v,
			})
		}
		genCtx := map[string]interface{}{
			"TypeName": typeInfo.Name,
			"Variants": evs,
		}

		var typeDef strings.Builder
		err = enumTmpl.Execute(&typeDef, genCtx)
		if err != nil {
			return kBogusGoTypeInfo, err
		}
		var typeSig strings.Builder
		err = enumSigTmpl.Execute(&typeSig, genCtx)
		if err != nil {
			return kBogusGoTypeInfo, err
		}

		err = g.types.emitType(typeInfo.Name, typeSig.String(), typeDef.String())
		if err != nil {
			return kBogusGoTypeInfo, err
		}
		return typeInfo, nil
	}

	return kBogusGoTypeInfo, fmt.Errorf("unknown pg type: '%s'", pgTypeName)
}

//
// Generation Tables
//
// These guys are the main drivers behind the conversion between postgres
// types and go types.
//

type goTypeInfo struct {
	// The Name of the type
	Name string
	// The name of a nullable version of the type
	NullName string
	// Given a variable name, SqlReceiver must return an appropriate wrapper around
	// that variable which can be passed as a parameter to Rows.scan. For many simple
	// types, SqlReceiver will just wrap the variable in a reference
	SqlReceiver func(string) string
}

func refWrap(variable string) string {
	return fmt.Sprintf("&(%s)", variable)
}

var kStringGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "string",
	NullName:    "*string",
	SqlReceiver: refWrap,
}

var kBoolGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "bool",
	NullName:    "*bool",
	SqlReceiver: refWrap,
}

var kTimeGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "time.Time",
	NullName:    "*time.Time",
	SqlReceiver: refWrap,
}

var kInt64GoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "int64",
	NullName:    "*int64",
	SqlReceiver: refWrap,
}

var kFloat64GoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "float64",
	NullName:    "*float64",
	SqlReceiver: refWrap,
}

var kUUIDGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "uuid.UUID",
	NullName:    "*uuid.UUID",
	SqlReceiver: refWrap,
}

var kByteArrayGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "[]byte",
	NullName:    "*[]byte",
	SqlReceiver: refWrap,
}

var kBogusGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "BOGUS (bug in pggen)",
	SqlReceiver: func(_ string) string { return "bug in pggen" },
}

var kPgType2GoType map[string]goTypeInfo = map[string]goTypeInfo{
	"text":              kStringGoTypeInfo,
	"character varying": kStringGoTypeInfo,
	"bpchar":            kStringGoTypeInfo,

	// There is no decimal type in go, so PG money types are returned
	// as text.
	"money": kStringGoTypeInfo,

	"time without time zone":      kTimeGoTypeInfo,
	"time with time zone":         kTimeGoTypeInfo,
	"timestamp without time zone": kTimeGoTypeInfo,
	"timestamp with time zone":    kTimeGoTypeInfo,
	"date":                        kTimeGoTypeInfo,

	"boolean": kBoolGoTypeInfo,

	"uuid": kUUIDGoTypeInfo,

	"smallint": kInt64GoTypeInfo,
	"integer":  kInt64GoTypeInfo,
	"bigint":   kInt64GoTypeInfo,
	// numeric types are returned as strings for the same reason that
	// money types are.
	"numeric":          kStringGoTypeInfo,
	"real":             kFloat64GoTypeInfo,
	"double precision": kFloat64GoTypeInfo,

	"bytea": kByteArrayGoTypeInfo,

	"record": kBogusGoTypeInfo,
}
