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

func (g *Generator) initTypeTable(overrides []typeOverride) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("while applying type overrides: %s", err.Error())
		}
	}()

	g.pgType2GoType = map[string]*goTypeInfo{}
	// just for the sake of cleanliness, let's avoid aliasing a global
	for k, v := range defaultPgType2GoType {
		g.pgType2GoType[k] = v
	}

	for _, override := range overrides {
		if len(override.PgTypeName) == 0 {
			return fmt.Errorf("type overrides must include a postgres type")
		}
		if len(override.TypeName) == 0 && len(override.NullableTypeName) == 0 {
			return fmt.Errorf(
				"type override must override the type or the nullable type")
		}
		if len(override.Pkg) == 0 && !primitveGoTypes[override.TypeName] {
			return fmt.Errorf(
				"type override must include a package unless the type is a primitive")
		}

		if len(override.Pkg) == 0 {
			g.imports[override.Pkg] = true
		}
		if len(override.NullPkg) == 0 {
			g.imports[override.NullPkg] = true
		}

		info, inMap := g.pgType2GoType[override.PgTypeName]
		if inMap {
			if len(override.TypeName) > 0 {
				info.Name = override.TypeName
			}
			if len(override.NullableTypeName) > 0 {
				info.NullName = override.NullableTypeName
			}
			if len(override.Pkg) > 0 {
				info.Pkg = override.Pkg
			}
			if len(override.NullPkg) > 0 {
				info.NullPkg = override.NullPkg
			}
		} else {
			if len(override.TypeName) == 0 ||
				len(override.NullableTypeName) == 0 {
				return fmt.Errorf(
					"`type_name` and `nullable_type_name` must both be " +
						"provided for a type that pggen does not have default " +
						"values for.")
			}

			g.pgType2GoType[override.PgTypeName] = &goTypeInfo{
				Name:        override.TypeName,
				Pkg:         override.Pkg,
				NullName:    override.NullableTypeName,
				NullPkg:     override.NullPkg,
				SqlReceiver: refWrap,
				SqlArgument: idWrap,
			}
		}
	}

	return nil
}

//
// functions and values internal to this file
//

func (g *Generator) primTypeInfoOf(pgTypeName string) (*goTypeInfo, error) {
	typeInfo, ok := g.pgType2GoType[pgTypeName]
	if ok {
		if len(typeInfo.Pkg) > 0 {
			g.imports[typeInfo.Pkg] = true
		}
		if len(typeInfo.NullPkg) > 0 {
			g.imports[typeInfo.NullPkg] = true
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
	// The package that the type with Name is in
	Pkg string
	// The name of a nullable version of the type
	NullName string
	// The package that the type with NullName is in (may be blank if
	// the same as Pkg)
	NullPkg string
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
	Pkg:         `"time"`,
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
	Pkg:         `uuid "github.com/satori/go.uuid"`,
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

var primitveGoTypes = map[string]bool{
	"string":  true,
	"byte":    true,
	"[]byte":  true,
	"int64":   true,
	"int32":   true,
	"int":     true,
	"bool":    true,
	"float64": true,
	"float32": true,
}

var defaultPgType2GoType = map[string]*goTypeInfo{
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
