// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package meta

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/opendoor/pggen/gen/internal/config"
	"github.com/opendoor/pggen/gen/internal/log"
	"github.com/opendoor/pggen/gen/internal/names"
	"github.com/opendoor/pggen/gen/internal/types"
	"github.com/opendoor/pggen/gen/internal/utils"
)

//
// This file contains queries used for extracting metadata about the
// database objects we are keying off of to generate code.
//

// Resolver knows how to query postgres for metadata about the database schema
type Resolver struct {
	db            *sql.DB
	tableResolver *tableResolver
	typeResolver  *types.Resolver
}

func NewResolver(
	l *log.Logger,
	db *sql.DB,
	typeResolver *types.Resolver,
	registerImport func(string),
) *Resolver {
	return &Resolver{
		db:            db,
		tableResolver: newTableResolver(l, db, typeResolver, registerImport),
		typeResolver:  typeResolver,
	}
}

// Resolve the metadata for the given database config that needs to be resolved ahead of
// time.
//
// This method _must_ be called before any of the query methods can be called.
func (r *Resolver) Resolve(conf *config.DbConfig) error {
	return r.tableResolver.populateTableInfo(conf.Tables)
}

// Get gen information about the given table
//
// The second return value indicates if the value was found
func (r *Resolver) TableMeta(pgName string) (*TableMeta, bool) {
	n, err := names.ParsePgName(pgName)
	if err != nil {
		return nil, false
	}

	res, ok := r.tableResolver.meta.tableInfo[n.String()]
	return res, ok
}

// Close closes the database connection that the resolver holds
func (r *Resolver) Close() error {
	return r.db.Close()
}

// Arg represents an argument to both a postgres query and the golang
// shim which wraps that query.
//
// fields are only public for template reflection
type Arg struct {
	// The 1-based index of this argument
	Idx int
	// The golang name of this argument
	GoName string
	// The postgres name of this argument
	PgName string
	// Information about the go version of this type
	TypeInfo types.Info
}

type QueryMeta struct {
	// The configuation data for this query from the .toml file
	ConfigData config.QueryConfig
	// The metadata for the arguments to this query
	Args []Arg
	// The metadata for the return values of this function
	ReturnCols []ColMeta
	// Flag indicating if there are multiple columns.
	// Included for the convenience of templates.
	MultiReturn bool
	// The name of the return type for a row returned by this query
	ReturnTypeName string
	// A golang comment derived from the Comment field from the query
	// config.
	Comment string
}

func (mc *Resolver) QueryMeta(
	config *config.QueryConfig,
	inferArgTypes bool,
) (ret QueryMeta, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("getting metadata for '%s': %s", config.Name, err.Error())
		}
	}()

	ret.ConfigData = *config

	ret.Comment = configCommentToGoComment(config.Comment)

	if inferArgTypes {
		var args []Arg
		args, err = mc.argsOfStmt(config.Body, config.ArgNames)
		if err != nil {
			err = fmt.Errorf("getting query argument types: %s", err.Error())
			return
		}
		ret.Args = args
	}

	// Resolve the return type by factoring in the null flags and
	// whether or not it is an alias for a table type.
	nullFlags := config.NullFlags
	pgTableName, isTable := mc.tableResolver.meta.tableTyNameToTableName[config.ReturnType]
	if isTable {
		if len(config.NullFlags) > 0 || len(config.NotNullFields) > 0 {
			err = fmt.Errorf("don't set null flags on query returning table struct")
			return
		}

		nullFlags = mc.tableResolver.meta.tableInfo[pgTableName].nullFlags()
	}
	returnCols, err := mc.queryReturns(config.Body)
	if err != nil {
		return
	}
	err = overrideNullability(returnCols, nullFlags, config.NotNullFields)
	if err != nil {
		return
	}
	ret.ReturnCols = returnCols

	if len(ret.ReturnCols) == 1 {
		ret.MultiReturn = false
		if ret.ReturnCols[0].Nullable {
			ret.ReturnTypeName = ret.ReturnCols[0].TypeInfo.NullName
		} else {
			ret.ReturnTypeName = ret.ReturnCols[0].TypeInfo.Name
		}

		if len(config.ReturnType) > 0 {
			err = fmt.Errorf("return_type cannot be provided for a query returning a primitive")
			return
		}
	} else {
		ret.MultiReturn = true
		if len(config.ReturnType) > 0 {
			ret.ReturnTypeName = config.ReturnType
		} else {
			ret.ReturnTypeName = ret.ConfigData.Name + "Row"
		}
	}
	ret.MultiReturn = len(ret.ReturnCols) > 1

	return
}

type StmtMeta struct {
	// The configuation data for this stmt from the .toml file
	ConfigData config.StmtConfig
	// The metadata for the arguments to this query
	Args []Arg
}

func (mc *Resolver) StmtMeta(
	config *config.StmtConfig,
) (ret StmtMeta, err error) {
	ret.ConfigData = *config

	args, err := mc.argsOfStmt(config.Body, config.ArgNames)
	if err != nil {
		err = fmt.Errorf("getting statement argument types: %s", err.Error())
		return
	}
	ret.Args = args

	return
}

// argsOfStmt infers the types of all the placeholders in the `body` statement
// and uses that to generate a list of argument metadata
func (mc *Resolver) argsOfStmt(body string, argNamesSpec string) ([]Arg, error) {
	// Connections require a context, so we'll use a dummy
	ctx := context.Background()

	// prepared statements are scoped to the database session
	// (the tcp connection to postgres, or connection in go terms)
	// In order to ensure that the prepared statement we make will
	// be visible in the `pg_prepared_statements` view, we need to
	// explicitly ask our connection pool for a connection so that it
	// doesn't give us a different one for a subsequent query.
	conn, err := mc.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		// The only mutation we've made is adding a prepared statement,
		// which is a session local thing, so we don't need to make a
		// hullabaloo if the rollback fails.
		_ = tx.Rollback()
	}()

	stmt, err := tx.Prepare(body)
	if err != nil {
		fmt.Println("failed prep, body =", body)
		return nil, err
	}
	// Don't check the error code. Not worth bringing down the process over.
	defer stmt.Close()

	var types RegTypeArray
	err = tx.QueryRow(`
		SELECT parameter_types
		FROM pg_prepared_statements
		WHERE statement = $1`, body).Scan(&types)
	if err != nil {
		return nil, fmt.Errorf("getting parameter types: %s", err.Error())
	}

	argNames, err := argNamesToSlice(argNamesSpec, len(types.pgTypes))
	if err != nil {
		return nil, err
	}
	args := make([]Arg, 0, len(types.pgTypes))
	for i, t := range types.pgTypes {
		name := argNames[i]
		typeInfo, err := mc.typeResolver.TypeInfoOf(t)
		if err != nil {
			return nil, fmt.Errorf("resolving type info: %s", err.Error())
		}
		args = append(args, Arg{
			Idx:      i + 1,
			GoName:   name,
			PgName:   name,
			TypeInfo: *typeInfo,
		})
	}

	return args, nil
}

type RegTypeArray struct {
	pgTypes []string
}

// Scan implements the `sql.Scanner` interface
func (r *RegTypeArray) Scan(src interface{}) error {
	// buff, ok := src.([]byte)
	regArrayString, ok := src.(string)
	if !ok {
		return fmt.Errorf("[]regtype Scan: expected a string")
	}

	if regArrayString[0] != '{' || regArrayString[len(regArrayString)-1] != '}' {
		return fmt.Errorf("[]regtype Scan: malformed data '%s'", regArrayString)
	}
	regArrayString = regArrayString[1 : len(regArrayString)-1]

	if len(regArrayString) == 0 {
		r.pgTypes = []string{}
		return nil
	}

	for len(regArrayString) > 0 {
		var ty string
		var err error
		ty, regArrayString, err = splitType(regArrayString)
		if err != nil {
			return err
		}
		r.pgTypes = append(r.pgTypes, ty)
	}

	return nil
}

// given a comma separated list of possibly quoted values,
// splitType takes the first one off the `types` slice.
func splitType(types string) (ty string, rest string, err error) {
	switch types[0] {
	case '"':
		for i := 1; i < len(types); i++ {
			switch types[i] {
			case '"':
				if types[i-1] == '\\' {
					continue
				}

				ty = string(types[1:i])

				if i+1 < len(types) {
					if i+2 < len(types) && types[i+1] == ',' {
						rest = types[i+2:]
					} else {
						rest = types[i+1:]
					}
				} else {
					// s[len(s):] is an error rather than returning the
					// empty slice, which is why we need this special case.
					rest = ""
				}

				return
			default:
				// do nothing
			}
		}
	default:
		for i, b := range types {
			if b == ',' {
				ty = string(types[:i])

				if i+1 >= len(types) {
					err = fmt.Errorf("[]regtype Scan: trailing comma")
					return
				}
				rest = types[i+1:]
				return
			}
		}
	}

	// the last (non-quoted) type
	ty = string(types)
	rest = ""
	return
}

func overrideNullability(
	cols []ColMeta,
	nullFlags string,
	notNullFields []string,
) error {
	if len(nullFlags) > 0 && len(notNullFields) > 0 {
		return fmt.Errorf(
			"cannot specify both null_flags and not_null_fields",
		)
	}

	if len(nullFlags) > 0 {
		if len(nullFlags) != len(cols) {
			return fmt.Errorf(
				"there are %d cols but %d null flags",
				len(cols), len(nullFlags),
			)
		}

		for i := range cols {
			switch nullFlags[i] {
			case 'n':
				cols[i].Nullable = true
			case '-':
				cols[i].Nullable = false
			default:
				return fmt.Errorf(
					"unknown null flag %s",
					string([]byte{nullFlags[i]}),
				)
			}
		}
	}

	if len(notNullFields) > 0 {
		nonNull := map[string]bool{}
		for _, f := range notNullFields {
			nonNull[f] = true
		}

		for i, c := range cols {
			cols[i].Nullable = !nonNull[c.PgName]
		}
	}

	return nil
}

// Given the name of a postgres stored function, return a list
// describing its arguments
func (mc *Resolver) FuncArgs(funcName names.PgName) ([]Arg, error) {
	rows, err := mc.db.Query(`
		WITH proc_args AS (
			SELECT
				UNNEST(p.proargnames) as argname,
				UNNEST(p.proargtypes) as argtype
			FROM pg_proc p
			JOIN pg_namespace ns
				ON (p.pronamespace = ns.oid)
			WHERE ns.nspname = $1
			  AND p.proname = $2
		), argmodes AS (
			SELECT
				UNNEST(p.proargnames) as argname,
				UNNEST(p.proargmodes) as argmode
			FROM pg_proc p
			JOIN pg_namespace ns
				ON (p.pronamespace = ns.oid)
			WHERE ns.nspname = $1
			  AND p.proname = $2
		)

		SELECT arg.argname, t.typname
		FROM proc_args arg
		JOIN pg_type t
			ON (arg.argtype = t.oid)
		LEFT JOIN argmodes mode
			ON (mode.argname = arg.argname)
		WHERE mode.argmode != 't'
		   OR mode.argmode IS NULL
		`, funcName.Schema, funcName.Name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var args []Arg
	i := 1
	for rows.Next() {
		var (
			a          Arg
			pgTypeName string
		)
		err = rows.Scan(&a.PgName, &pgTypeName)
		if err != nil {
			return nil, err
		}

		a.Idx = i
		a.GoName = names.PgToGoName(a.PgName)
		typeInfo, err := mc.typeResolver.TypeInfoOf(pgTypeName)
		if err != nil {
			return nil, err
		}
		a.TypeInfo = *typeInfo

		i++
		args = append(args, a)
	}

	return args, nil
}

// Given a query string, return metadata about the columns that it will return
func (mc *Resolver) queryReturns(query string) ([]ColMeta, error) {
	viewName := utils.RandomName("tmp_view")
	view := fmt.Sprintf(
		`CREATE OR REPLACE TEMP VIEW %s AS %s`,
		viewName, utils.NullOutArgs(query),
	)

	_, err := mc.db.Exec(view)
	if err != nil {
		return nil, err
	}

	viewMeta, err := mc.tableResolver.tableInfo(&config.TableConfig{Name: viewName})
	if err != nil {
		return nil, err
	}

	// This should be totally unneeded, but I have observed the tmp
	// views popping up in psql sessions that were already active
	// when pggen was run. We intentionally don't check the error
	// code here because we really don't care too much if this
	// doesn't work.
	_, err = mc.db.Exec(fmt.Sprintf(`DROP VIEW IF EXISTS %s`, viewName))
	if err != nil {
		return nil, err
	}

	return viewMeta.Cols, nil
}

// RefMeta contains metadata for a reference between two tables
// (a foreign key relationship)
type RefMeta struct {
	// The metadata for the table that holds the foreign key
	PointsTo *TableMeta
	// The names of the fields in the referenced table that are used as keys
	// (usually the primary keys of that table). Order matters.
	PointsToField *ColMeta
	// The metadata for the table is being referred to
	PointsFrom *TableMeta
	// The names of the fields that are being used to refer to the key fields
	// for the referenced table. Order matters.
	PointsFromField *ColMeta
	// The name of the field that should be generated in the model being pointed
	// to by the foreign key (parent model).
	GoPointsFromFieldName string
	// A snake_case version of GoPointsFromFieldName
	PgPointsFromFieldName string
	// The name of the field that should be generated in the model being pointed
	// from by the foreign key (child model).
	GoPointsToFieldName string
	// A snake_case version of GoPointsToFieldName.
	PgPointsToFieldName string
	// Indicates that there can be at most one of these references between
	// the two tables.
	OneToOne bool
	// Indicates whether or not the foreign key associated with this reference
	// is nullable.
	Nullable bool
}

// given a slice of columns, return a table mapping the ColNums to indicies in the slice
func columnResolverTable(cols []ColMeta) []int {
	max := 0
	for _, col := range cols {
		if int(col.ColNum) > max {
			max = int(col.ColNum)
		}
	}
	colNumToIdx := make([]int, max+1)
	for i, col := range cols {
		colNumToIdx[col.ColNum] = i
	}

	return colNumToIdx
}
