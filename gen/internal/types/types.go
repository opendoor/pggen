package types

import (
	"database/sql"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/opendoor-labs/pggen/gen/internal/config"
)

type Resolver struct {
	// A table mapping postgres primitive types to go types.
	pgType2GoType map[string]*Info
	// register the given import string with an import list
	registerImport func(string)
	// The clearing house for types that we emit. They all go here
	// before being generated for real. We do this to prevent generating
	// the same type twice.
	types set
	// A connection to the database we can use to get metadata about the
	// schema.
	db *sql.DB
}

func NewResolver(db *sql.DB, registerImport func(string)) *Resolver {
	return &Resolver{
		pgType2GoType:  map[string]*Info{},
		registerImport: registerImport,
		types:          newSet(),
		db:             db,
	}
}

// Resolve performs any ahead-of-time computations needed to service subsequent
// type resolution requests.
//
// This method _must_ be called before any other methods are called.
func (r *Resolver) Resolve(conf *config.DbConfig) error {
	return r.initTypeTable(conf.TypeOverrides)
}

// emit all the types we have build up into the given Writer
func (r *Resolver) Gen(into io.Writer) error {
	return r.types.gen(into)
}

func (r *Resolver) EmitType(name string, sig string, body string) error {
	return r.types.emitType(name, sig, body)
}

func (r *Resolver) Probe(name string) bool {
	return r.types.probe(name)
}

type Info struct {
	// The Name of the type
	Name string
	// The package that the type with Name is in
	Pkg string
	// The name of a nullable version of the type
	NullName string
	// The package that the type with NullName is in (may be blank if
	// the same as Pkg)
	NullPkg string
	// The name of a nullable version of the type that should be used
	// for interfacing with the database. This type will get converted
	// into `NullName` before it reaches any public-facing part of the
	// generated code.
	ScanNullName string
	// The package that the type with ScanNullName type is in. May be
	// blank if same as either one of the other two packages.
	ScanNullPkg string
	// A function for transforming a variable with the given name of type
	// ScanNullName into a block of code which evaluates to a value of type
	// NullName
	NullConvertFunc func(string) string
	// Given a variable name, SqlReceiver must return an appropriate wrapper
	// around that variable which can be passed as a parameter to Rows.Scan.
	// For many simple types, SqlReceiver will just wrap the variable in a
	// reference.
	SqlReceiver func(string) string
	// Given a variable name, SqlReceiver must return an appropriate wrapper
	// around that variable which can be passed as a parameter to Rows.Scan.
	// Must work for the nullable receiver wrapper.
	NullSqlReceiver func(string) string
	// Given a variable name, SqlArgument must return an appropriate wrapper
	// around that variable which can be passed as a parameter to `sql.Query`
	SqlArgument func(string) string
	// Given a variable name of type pointer-to-type, NullSqlArgument must return
	// an appropriate value to pas as a parameter to `sql.Query`
	NullSqlArgument func(string) string
	// If this is a timestamp type, it has a time zone, otherwise this field
	// is meaningless.
	IsTimestampWithZone bool
	// A flag indicating that this TypeInfo is for an enum. Not for use by
	// templates, only for handling arrays of enums.
	isEnum bool
}

func (r *Resolver) TypeInfoOf(pgTypeName string) (*Info, error) {
	arrayType, err := parsePgArray(pgTypeName)
	if err == nil {
		switch innerTy := arrayType.inner.(type) {
		case *pgArrayType:
			return nil, fmt.Errorf("nested arrays are not supported")
		case *pgPrimType:
			tyInfo, err := r.primTypeInfoOf(innerTy.name)
			if err != nil {
				return nil, err
			}

			sqlArgument := arrayWrap
			if tyInfo.isEnum {
				sqlArgument = stringizeArrayWrap
			}

			return &Info{
				Name:            "[]" + tyInfo.Name,
				NullName:        "[]" + tyInfo.NullName,
				ScanNullName:    "[]" + tyInfo.ScanNullName,
				NullConvertFunc: arrayConvert(tyInfo.NullConvertFunc, tyInfo.NullName),
				// arrays need special wrappers
				SqlReceiver:     arrayRefWrap,
				NullSqlReceiver: arrayRefWrap,
				SqlArgument:     sqlArgument,
				NullSqlArgument: sqlArgument,
			}, nil
		}
	}

	return r.primTypeInfoOf(pgTypeName)
}

func (r *Resolver) initTypeTable(overrides []config.TypeOverride) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("while applying type overrides: %s", err.Error())
		}
	}()

	r.pgType2GoType = map[string]*Info{}
	// just for the sake of cleanliness, let's avoid aliasing a global
	for k, v := range defaultPgType2GoType {
		r.pgType2GoType[k] = v
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

		if len(override.Pkg) > 0 {
			r.registerImport(override.Pkg)
		}
		if len(override.NullPkg) > 0 {
			r.registerImport(override.NullPkg)
		}

		convertFunc := identityConvert
		if override.NullableToBoxed != "" {
			tmpl, err := template.New("nullable_to_boxed_" + override.TypeName).
				Parse(override.NullableToBoxed)
			if err != nil {
				return fmt.Errorf(
					"bad 'nullable_to_boxed' template for '%s': %s",
					override.TypeName,
					err.Error(),
				)
			}
			convertFunc = convertUserTmpl(tmpl)
		}

		info, inMap := r.pgType2GoType[override.PgTypeName]
		if inMap {
			if len(override.TypeName) > 0 {
				info.Name = override.TypeName
			}
			if len(override.NullableTypeName) > 0 {
				info.NullName = "*" + override.TypeName
				info.ScanNullName = override.NullableTypeName
				info.NullConvertFunc = convertFunc
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

			r.pgType2GoType[override.PgTypeName] = &Info{
				Name:            override.TypeName,
				Pkg:             override.Pkg,
				NullName:        "*" + override.TypeName,
				ScanNullName:    override.NullableTypeName,
				NullConvertFunc: convertFunc,
				NullPkg:         override.NullPkg,
				SqlReceiver:     refWrap,
				NullSqlReceiver: refWrap,
				SqlArgument:     idWrap,
				NullSqlArgument: idWrap,
			}
		}
	}

	return nil
}

func (r *Resolver) primTypeInfoOf(pgTypeName string) (*Info, error) {
	typeInfo, ok := r.pgType2GoType[pgTypeName]
	if ok {
		if len(typeInfo.Pkg) > 0 {
			r.registerImport(typeInfo.Pkg)
		}
		if len(typeInfo.NullPkg) > 0 {
			r.registerImport(typeInfo.NullPkg)
		}
		return typeInfo, nil
	}

	enumTypeInfo, err := r.maybeEmitEnumType(pgTypeName)
	if err == nil {
		return enumTypeInfo, nil
	}

	if strings.HasPrefix(pgTypeName, "numeric") {
		return &stringGoTypeInfo, nil
	}
	if strings.HasPrefix(pgTypeName, "character varying") {
		return &stringGoTypeInfo, nil
	}
	if strings.HasPrefix(pgTypeName, "character") {
		return &stringGoTypeInfo, nil
	}

	return nil, fmt.Errorf(
		"unknown pg type: '%s': %s",
		pgTypeName,
		err.Error(),
	)
}

//
// Wrap and Convert routines
//

func idWrap(variable string) string {
	return variable
}

func refWrap(variable string) string {
	return fmt.Sprintf("&(%s)", variable)
}

func arrayWrap(variable string) string {
	return fmt.Sprintf("pgtypes.Array(%s)", variable)
}

func arrayRefWrap(variable string) string {
	return fmt.Sprintf("pgtypes.Array(&(%s))", variable)
}

func convertCall(fun string) func(string) string {
	return func(v string) string {
		return fmt.Sprintf("%s(%s)", fun, v)
	}
}

func identityConvert(v string) string {
	return v
}

func convertUserTmpl(tmpl *template.Template) func(string) string {
	return func(v string) string {
		type tmplCtx struct {
			Value string
		}
		c := tmplCtx{Value: v}

		var out strings.Builder
		err := tmpl.Execute(&out, c)
		if err != nil {
			// This routine will get executed by the code generator template,
			// so there is no really great way to cleanly report this error.
			// We'll just have to panic.
			panic("bad template '" + tmpl.Name() + "': " + err.Error())
		}
		return out.String()
	}
}

func arrayConvert(
	elemConvert func(string) string,
	nullName string,
) func(string) string {
	type tmplCtx struct {
		ElemConvert    func(string) string
		NullName       string
		InputArrayName string
	}
	tmpl := template.Must(template.New("array-convert-tmpl").Parse(
		`func() []{{ .NullName }} {
		out := make([]{{ .NullName }}, 0, len({{ .InputArrayName }}))
		for _, elem := range {{ .InputArrayName }} {
			out = append(out, {{ call .ElemConvert "elem" }})
		}
		return out
	}()`))

	return func(v string) string {
		var out strings.Builder
		ctx := tmplCtx{
			ElemConvert:    elemConvert,
			NullName:       nullName,
			InputArrayName: v,
		}
		_ = tmpl.Execute(&out, ctx)
		return out.String()
	}
}

//
// Generation Tables
//
// These guys are the main drivers behind the conversion between postgres
// types and go types.
//

var stringGoTypeInfo Info = Info{
	Name:            "string",
	NullName:        "*string",
	ScanNullName:    "sql.NullString",
	ScanNullPkg:     `"database/sql"`,
	NullConvertFunc: convertCall("convertNullString"),
	SqlReceiver:     refWrap,
	NullSqlReceiver: refWrap,
	SqlArgument:     idWrap,
	NullSqlArgument: idWrap,
}

var boolGoTypeInfo Info = Info{
	Name:            "bool",
	NullName:        "*bool",
	ScanNullName:    "sql.NullBool",
	ScanNullPkg:     `"database/sql"`,
	NullConvertFunc: convertCall("convertNullBool"),
	SqlReceiver:     refWrap,
	NullSqlReceiver: refWrap,
	SqlArgument:     idWrap,
	NullSqlArgument: idWrap,
}

var timeGoTypeInfo Info = Info{
	Pkg:             `"time"`,
	Name:            "time.Time",
	NullName:        "*time.Time",
	ScanNullName:    "pggenNullTime",
	ScanNullPkg:     "",
	NullConvertFunc: convertCall("convertNullTime"),
	SqlReceiver:     refWrap,
	NullSqlReceiver: refWrap,
	SqlArgument:     idWrap,
	NullSqlArgument: idWrap,
}

var timezGoTypeInfo Info = Info{
	Pkg:                 `"time"`,
	Name:                "time.Time",
	NullName:            "*time.Time",
	ScanNullName:        "pggenNullTime",
	ScanNullPkg:         "",
	NullConvertFunc:     convertCall("convertNullTime"),
	SqlReceiver:         refWrap,
	NullSqlReceiver:     refWrap,
	SqlArgument:         idWrap,
	NullSqlArgument:     idWrap,
	IsTimestampWithZone: true,
}

var int64GoTypeInfo Info = Info{
	Name:            "int64",
	NullName:        "*int64",
	ScanNullName:    "sql.NullInt64",
	ScanNullPkg:     `"database/sql"`,
	NullConvertFunc: convertCall("convertNullInt64"),
	SqlReceiver:     refWrap,
	NullSqlReceiver: refWrap,
	SqlArgument:     idWrap,
	NullSqlArgument: idWrap,
}

var float64GoTypeInfo Info = Info{
	Name:            "float64",
	NullName:        "*float64",
	ScanNullName:    "sql.NullFloat64",
	ScanNullPkg:     `"database/sql"`,
	NullConvertFunc: convertCall("convertNullFloat64"),
	SqlReceiver:     refWrap,
	NullSqlReceiver: refWrap,
	SqlArgument:     idWrap,
	NullSqlArgument: idWrap,
}

var byteArrayGoTypeInfo Info = Info{
	Name:            "[]byte",
	NullName:        "*[]byte",
	ScanNullName:    "*[]byte",
	NullConvertFunc: identityConvert,
	SqlReceiver:     refWrap,
	NullSqlReceiver: refWrap,
	SqlArgument:     idWrap,
	NullSqlArgument: idWrap,
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

var defaultPgType2GoType = map[string]*Info{
	"text":              &stringGoTypeInfo,
	"character varying": &stringGoTypeInfo,
	"bpchar":            &stringGoTypeInfo,
	"citext":            &stringGoTypeInfo,
	"interval":          &stringGoTypeInfo,

	// There is no decimal type in go, so PG money types are returned
	// as text.
	"money": &stringGoTypeInfo,

	"time without time zone":      &timeGoTypeInfo,
	"time with time zone":         &timezGoTypeInfo,
	"timestamp without time zone": &timeGoTypeInfo,
	"timestamp with time zone":    &timezGoTypeInfo,
	"date":                        &timeGoTypeInfo,

	"boolean": &boolGoTypeInfo,

	"bigint":   &int64GoTypeInfo,
	"int4":     &int64GoTypeInfo,
	"int8":     &int64GoTypeInfo,
	"integer":  &int64GoTypeInfo,
	"smallint": &int64GoTypeInfo,

	// Without knowing more about the shape of the data we are getting handed back,
	// we will just use an untyped blob for json. We don't want to use an `interface{}`
	// because the user very well might have a struct that they want to deserialize
	// stuff into.
	"json":  &byteArrayGoTypeInfo,
	"jsonb": &byteArrayGoTypeInfo,

	// numeric types are returned as strings for the same reason that
	// money types are.
	"numeric": &stringGoTypeInfo,

	"real":             &float64GoTypeInfo,
	"double precision": &float64GoTypeInfo,

	"bytea": &byteArrayGoTypeInfo,

	"record": nil,
}
