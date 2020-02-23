package gen

import (
	"fmt"
	"strings"
	"text/template"
)

func (g *Generator) typeInfoOf(pgTypeName string) (*goTypeInfo, error) {
	arrayType, err := parsePgArray(pgTypeName)
	if err == nil {
		switch innerTy := arrayType.inner.(type) {
		case *pgArrayType:
			return nil, fmt.Errorf("nested arrays are not supported")
		case *pgPrimType:
			tyInfo, err := g.primTypeInfoOf(innerTy.name)
			if err != nil {
				return nil, err
			}

			return &goTypeInfo{
				Name:     "[]" + tyInfo.Name,
				NullName: "[]" + tyInfo.NullName,
				// arrays need special wrappers
				SqlReceiver: arrayRefWrap,
				SqlArgument: arrayWrap,
			}, nil
		}
	}

	return g.primTypeInfoOf(pgTypeName)
}

func (g *Generator) primTypeInfoOf(pgTypeName string) (*goTypeInfo, error) {
	typeInfo, ok := pgType2GoType[pgTypeName]
	if ok {
		if typeInfo.Name == "time.Time" {
			g.imports[`"time"`] = true
		}
		if typeInfo.Name == "uuid.UUID" {
			g.imports[`uuid "github.com/satori/go.uuid"`] = true
		}
		return typeInfo, nil
	}

	enumTypeInfo, err := g.maybeEmitEnumType(pgTypeName)
	if err == nil {
		return enumTypeInfo, nil
	}

	return nil, fmt.Errorf(
		"unknown pg type: '%s': %s",
		pgTypeName,
		err.Error(),
	)
}

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
		typeInfo := goTypeInfo{
			Name:        pgToGoName(pgTypeName),
			NullName:    "Null" + pgToGoName(pgTypeName),
			SqlReceiver: refWrap,
			SqlArgument: idWrap,
		}

		if g.types.probe(typeInfo.Name) {
			// we've already generated a type for this enum, so we can
			// just return
			return &typeInfo, nil
		}

		enumSigTmpl := template.Must(template.New("enum-sig-tmpl").Parse(`
{{- range (index . "Variants") }}
{{ index $ "TypeName" }}{{ .GoName }} {{ index $ "TypeName" }} = "{{ .PgName }}"
{{- end }}
`))

		g.imports[`"database/sql/driver"`] = true

		enumTmpl := template.Must(template.New("enum-tmpl").Parse(`
type {{ index . "TypeName" }} string
const (
{{- range (index . "Variants") }}
	{{ index $ "TypeName" }}{{ .GoName }} {{ index $ "TypeName" }} = "{{ .PgName }}"
{{- end }}
)

type Null{{ index . "TypeName" }} struct {
	{{ index . "TypeName" }} string
	Valid bool
}
// Scan implements the sql.Scanner interface
func (n *Null{{ index . "TypeName"}}) Scan(value interface{}) error {
	if value == nil {
		n.{{ index . "TypeName" }}, n.Valid = "", false
		return nil
	}
	buff, ok := value.([]byte)
	if !ok {
		return fmt.Errorf(
			"Null{{ index . "TypeName" }}.Scan: expected a []byte",
		)
	}

	n.Valid = true
	n.{{ index . "TypeName" }} = string(buff)
	return nil
}
// Value implements the sql.Valuer interface
func (n Null{{ index . "TypeName" }}) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.{{ index . "TypeName" }}, nil
}
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

	return nil, nil
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
	// Given a variable name, SqlReceiver must return an appropriate wrapper
	// around that variable which can be passed as a parameter to Rows.scan.
	// For many simple types, SqlReceiver will just wrap the variable in a
	// reference.
	SqlReceiver func(string) string
	// Given a variable name, SqlArgument must return an appropriate wrapper
	// around that variable which can be passed as a parameter to `sql.Query`
	SqlArgument func(string) string
}

func idWrap(variable string) string {
	return variable
}

func refWrap(variable string) string {
	return fmt.Sprintf("&(%s)", variable)
}

func arrayWrap(variable string) string {
	return fmt.Sprintf("pq.Array(%s)", variable)
}

func arrayRefWrap(variable string) string {
	return fmt.Sprintf("pq.Array(&(%s))", variable)
}

var stringGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "string",
	NullName:    "sql.NullString",
	SqlReceiver: refWrap,
	SqlArgument: idWrap,
}

var boolGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "bool",
	NullName:    "sql.NullBool",
	SqlReceiver: refWrap,
	SqlArgument: idWrap,
}

var timeGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "time.Time",
	NullName:    "sql.NullTime",
	SqlReceiver: refWrap,
	SqlArgument: idWrap,
}

var int64GoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "int64",
	NullName:    "sql.NullInt64",
	SqlReceiver: refWrap,
	SqlArgument: idWrap,
}

var float64GoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "float64",
	NullName:    "sql.NullFloat64",
	SqlReceiver: refWrap,
	SqlArgument: idWrap,
}

var uuidGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "uuid.UUID",
	NullName:    "uuid.NullUUID",
	SqlReceiver: refWrap,
	SqlArgument: idWrap,
}

var byteArrayGoTypeInfo goTypeInfo = goTypeInfo{
	Name:        "[]byte",
	NullName:    "*[]byte",
	SqlReceiver: refWrap,
	SqlArgument: idWrap,
}

var pgType2GoType = map[string]*goTypeInfo{
	"text":              &stringGoTypeInfo,
	"character varying": &stringGoTypeInfo,
	"bpchar":            &stringGoTypeInfo,

	// There is no decimal type in go, so PG money types are returned
	// as text.
	"money": &stringGoTypeInfo,

	"time without time zone":      &timeGoTypeInfo,
	"time with time zone":         &timeGoTypeInfo,
	"timestamp without time zone": &timeGoTypeInfo,
	"timestamp with time zone":    &timeGoTypeInfo,
	"date":                        &timeGoTypeInfo,

	"boolean": &boolGoTypeInfo,

	"uuid": &uuidGoTypeInfo,

	"smallint": &int64GoTypeInfo,
	"integer":  &int64GoTypeInfo,
	"bigint":   &int64GoTypeInfo,

	// numeric types are returned as strings for the same reason that
	// money types are.
	"numeric": &stringGoTypeInfo,

	"real":             &float64GoTypeInfo,
	"double precision": &float64GoTypeInfo,

	"bytea": &byteArrayGoTypeInfo,

	"record": nil,
}
