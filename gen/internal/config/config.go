package config

import (
	"fmt"

	"github.com/opendoor/pggen/gen/internal/names"
)

// The configuration file format used to specify the database objects
// to generate code for.
type DbConfig struct {
	// The name of the field that should be updated by pggen's generated
	// `Insert` methods. Overridden by the config option of the same name
	// on TableConfig.
	CreatedAtField string `toml:"created_at_field"`
	// The name of the field that should be updated by pggen's generated
	// `Update` and `Insert` methods. Overridden by the config option of
	// the same name on TableConfig.
	UpdatedAtField string `toml:"updated_at_field"`
	// The name of the nullable timestamp field that should be used to
	// implement soft deletes. Overridden by the config option of the
	// same name on TableConfig.
	DeletedAtField string `toml:"deleted_at_field"`
	// If true, it is an error for any [[query]] config block to be missing
	// the `comment` field. Useful if you want to be strict about documentation.
	RequireQueryComments bool           `toml:"require_query_comments"`
	TypeOverrides        []TypeOverride `toml:"type_override"`
	Queries              []QueryConfig  `toml:"query"`
	Stmts                []StmtConfig   `toml:"statement"`
	Tables               []TableConfig  `toml:"table"`
}

// Queries registered in the config file represent arbitrary bits of
// SQL, possibly parameterized by $N arguments. The generated code
// will use `sql.QueryContext` and marshal the results into a list of
// rows returned.
type QueryConfig struct {
	// The name that should be used to identify this query in generated go
	// code.
	Name string `toml:"name"`
	// A comment to place on the generated method so that IDEs can provide
	// online documentation for the method.
	Comment string `toml:"comment"`
	// The actual text of the query.
	Body string `toml:"body"`
	// A string consisting of the runes '-' and 'n' to indicate the
	// nullability of return columns. '-' indicates that the column is
	// not nullable (NOT NULL), while 'n' indicates that it is nullable.
	// These need to be specified manually because postgres does not expose
	// a mechanism for infering the nullability of query results that I
	// could discover. The flags string must be exactly as long as the
	// result set is wide.
	NullFlags string `toml:"null_flags"`
	// A long-form way of specifying the same thing as `NullFlags`. Only one
	// of the two options should be provided. Any fields appearing in this list
	// will be treated as not nullable, with all other fields being considered
	// nullable as is the default.
	NotNullFields []string `toml:"not_null_fields"`
	// The name that should be used for this query's return type.
	// This is useful because it allows multiple queries to return
	// values of the same type so that client code does not have to
	// perform a series of endless conversions. If two queries which
	// return different types are given the same name to use for their
	// return type, it is an error.
	ReturnType string `toml:"return_type"`
	// A mapping of argument numbers to names to generate for them.
	// This configuration option allows you to give useful names to the
	// query arguments in the genrated code (normaly pggen will just make up
	// names like `arg0`, `arg1` and so on). An example mapping is
	// `"1:foo 2:bar 3:baz"`.
	ArgNames string `toml:"arg_names"`
	// If true, this query is expected to return just one result row, so
	// `pggen` will generate code that returns just a single result rather
	// than a slice that always has just one record you have to unpack.
	// If `single_result` is true, `pggen` will not generate a *Query method
	// for this query, as there is no point to supporting streaming mode for
	// a single-result query.
	SingleResult bool `toml:"single_result"`
	// If true, allow nullable types to be passed in as arguments to the query.
	// Normally, query arguments are always non-null so making every argument
	// a pointer type would just be annoying for client code, but sometimes you
	// do actually want nullable arguments.
	NullableArguments bool `toml:"nullable_arguments"`
}

// Statements are like queries but they are executed for side effects
// and therefore return `(sql.Result, error)` rather than a set of
// rows. Statements should be used for INSERT, UPDATE, and DELETE
// operations.
type StmtConfig struct {
	// The name that should be used to identify this statement in generated
	// go code.
	Name string `toml:"name"`
	// The actual text of this statement.
	Body string `toml:"body"`
	// A mapping of argument numbers to names to generate for them.
	// This configuration option allows you to give useful names to the
	// query arguments in the genrated code (normaly pggen will just make up
	// names like `arg0`, `arg1` and so on). An example mapping is
	// `"1:foo 2:bar 3:baz"`.
	ArgNames string `toml:"arg_names"`
}

type TableConfig struct {
	// The name of the table in the database
	Name string `toml:"name"`
	// If true, pggen will not infer a relationship between this table
	// and any owning tables based on any foreign keys in this table.
	NoInferBelongsTo bool `toml:"no_infer_belongs_to"`
	// A list of tables that this table belongs to
	BelongsTo []BelongsTo `toml:"belongs_to"`
	// The timestamp to update in `Insert`. Overriddes global version.
	CreatedAtField string `toml:"created_at_field"`
	// The timestamp to update in `Update` and `Insert`.
	// Overriddes global version.
	UpdatedAtField string `toml:"updated_at_field"`
	// The nullable timestamp for implementing soft deletes.
	// Overriddes global version.
	DeletedAtField string `toml:"deleted_at_field"`
	// A list of extra annotations to add to the generated fields.
	FieldTags []FieldTag `toml:"field_tags"`
	// A list of annotations indicating types that specific json columns should
	// be (de)serialized into. By default, all `json` and `jsonb` columns will
	// become byte arrays.
	JsonTypes []JsonType `toml:"json_type"`
}

// An explicitly configured foreign key relationship which can be attached
// to a table's config.
type BelongsTo struct {
	// The table that this table belongs to
	Table string `toml:"table"`
	// The name of the foreign key which points to the table this table
	// belongs to.
	KeyField string `toml:"key_field"`
	// Optional. If true the owning table has at most one of this table
	OneToOne bool `toml:"one_to_one"`
	// Optional. The name to give the pointer field in the generated parent
	// struct. If not provided, this will just be the name of the child struct.
	ParentFieldName string `toml:"parent_field_name"`
	// Optional. The name to give the pointer field in the generated child
	// struct. If not provided, this will just be the name of the parent struct.
	ChildFieldName string `toml:"child_field_name"`
}

// Custom annotations to attach to the field generated for a given
// database column.
type FieldTag struct {
	ColumnName string `toml:"column_name"`
	Tags       string `toml:"tags"`
}

type JsonType struct {
	// The name of the `json` or `jsonb` column which should be parsed and serialized
	// into a structured type using the encoding/json package.
	ColumnName string `toml:"column_name"`
	// The name of the type, including package name, that the column should be parsed
	// into.
	TypeName string `toml:"type_name"`
	// The import string for the package in which the type lives. Should include quotes.
	Pkg string `toml:"pkg"`
}

type TypeOverride struct {
	// The name of the type in postgres
	PgTypeName string `toml:"postgres_type_name"`
	// The name of the package in which the type appears as it would
	// appear in an import list, including quotes. The package name
	// may include an alias just like an import might.
	//
	// Examples:
	//   - '"github.com/google/uuid"'
	//   - 'my_uuid_alias "github.com/google/uuid"'
	Pkg string `toml:"pkg"`
	// The go name for the type, including package name
	TypeName string `toml:"type_name"`
	// The name of the package in which the nullable version of the type
	// appears. If `pkg` was already provided, `nullable_pkg` may be omitted.
	NullPkg string `toml:"nullable_pkg"`
	// The name of a go type which might be null (often Null<TypeName>)
	NullableTypeName string `toml:"nullable_type_name"`
	// This should contain a golang template that expands to a go expressions of type
	// `type_name`. The template can expect a context which includes key `.Value` which
	// will be an expression which evaluates to a value of type `nullable_type_name`.
	// It `pkg` and `nullable_pkg` will have been imported, so you can use them in the
	// expression if needed.
	//
	// For example, the nullable_to_boxed template for the binding the UUID type from the
	// github.com/gofrs/uuid package to postgres' `uuid` type might look like:
	//
	// ```
	// func(u uuid.NullUUID) *uuid.UUID {
	// 	if u.Valid {
	// 		return &u.UUID
	// 	}
	// 	return nil
	// }({{ .Value }})
	// ```
	//
	// If no template expression is provided, `.Value` will be assumed to be directly
	// assignable to a boxed version of `type_name`.
	NullableToBoxed string `toml:"nullable_to_boxed"`
}

// Give a user provided configuration, runs some santity checks on the provided values
// to try to provent users from encountering hard to diagnose issues down the line.
func (c *DbConfig) Validate() error {
	for _, override := range c.TypeOverrides {
		if len(override.Pkg) > 0 {
			err := names.ValidateImportPath(override.Pkg)
			if err != nil {
				return fmt.Errorf("override for type '%s': %s", override.PgTypeName, err.Error())
			}
		}
		if len(override.NullPkg) > 0 {
			err := names.ValidateImportPath(override.NullPkg)
			if err != nil {
				return fmt.Errorf("override for type '%s': %s", override.PgTypeName, err.Error())
			}
		}
	}

	for _, table := range c.Tables {
		for _, jsonType := range table.JsonTypes {
			if len(jsonType.Pkg) > 0 {
				err := names.ValidateImportPath(jsonType.Pkg)
				if err != nil {
					return fmt.Errorf(
						"table '%s': column '%s': %s",
						table.Name,
						jsonType.ColumnName,
						err.Error(),
					)
				}
			}
		}
	}

	return nil
}

// Given a user provided configuration, convert it into a normalized form that
// is suitable for use by pggen.
//
// In particular we:
//   - resolve timestamp overrides and inheritance
func (c *DbConfig) Normalize() error {
	for i, tc := range c.Tables {
		if len(tc.CreatedAtField) == 0 && len(c.CreatedAtField) > 0 {
			c.Tables[i].CreatedAtField = c.CreatedAtField
		}

		if len(tc.UpdatedAtField) == 0 && len(c.UpdatedAtField) > 0 {
			c.Tables[i].UpdatedAtField = c.UpdatedAtField
		}

		if len(tc.DeletedAtField) == 0 && len(c.DeletedAtField) > 0 {
			c.Tables[i].DeletedAtField = c.DeletedAtField
		}
	}

	return nil
}
