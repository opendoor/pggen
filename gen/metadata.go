package gen

import (
	"context"
	"fmt"

	"github.com/jinzhu/inflection"
	"github.com/lib/pq"
)

//
// This file contains queries used for extracting metadata about the
// database objects we are keying off of to generate code.
//

// arg represents an argument to both a postgres query and the golang
// shim which wraps that query.
//
// fields are only public for template reflection
type arg struct {
	// The 1-based index of this argument
	Idx int
	// The golang name of this argument
	GoName string
	// The postgres name of this argument
	PgName string
	// Information about the go version of this type
	TypeInfo goTypeInfo
}

type queryMeta struct {
	// The configuation data for this query from the .toml file
	ConfigData queryConfig
	// The metadata for the arguments to this query
	Args []arg
	// The metadata for the return values of this function
	ReturnCols []colMeta
	// Flag indicating if there are multiple columns.
	// Included for the convenience of templates.
	MultiReturn bool
	// The name of the return type for a row returned by this query
	ReturnTypeName string
}

func (g *Generator) queryMeta(
	config *queryConfig,
	inferArgTypes bool,
) (ret queryMeta, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("getting metadata for '%s': %s", config.Name, err.Error())
		}
	}()

	ret.ConfigData = *config

	if inferArgTypes {
		var args []arg
		args, err = g.argsOfStmt(config.Body)
		if err != nil {
			err = fmt.Errorf("getting query argument types: %s", err.Error())
			return
		}
		ret.Args = args
	}

	// Resolve the return type by factoring in the null flags and
	// whether or not it is an alias for a table type.
	nullFlags := config.NullFlags
	pgTableName, isTable := g.tableTyNameToTableName[pgToGoName(config.ReturnType)]
	if isTable {
		if len(config.NullFlags) > 0 || len(config.NotNullFields) > 0 {
			err = fmt.Errorf("don't set null flags on query returning table struct")
			return
		}

		nullFlags = g.tables[pgTableName].nullFlags()
	}
	returnCols, err := g.queryReturns(config.Body)
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

type stmtMeta struct {
	// The configuation data for this stmt from the .toml file
	ConfigData stmtConfig
	// The metadata for the arguments to this query
	Args []arg
}

func (g *Generator) stmtMeta(
	config *stmtConfig,
) (ret stmtMeta, err error) {
	ret.ConfigData = *config

	args, err := g.argsOfStmt(config.Body)
	if err != nil {
		err = fmt.Errorf("getting statement argument types: %s", err.Error())
		return
	}
	ret.Args = args

	return
}

// argsOfStmt infers the types of all the placeholders in the `body` statement
// and uses that to generate a list of argument metadata
func (g *Generator) argsOfStmt(body string) ([]arg, error) {
	// Connections require a context, so we'll use a dummy
	ctx := context.TODO()

	// prepared statements are scoped to the database session
	// (the tcp connection to postgres, or connection in go terms)
	// In order to ensure that the prepared statement we make will
	// be visible in the `pg_prepared_statements` view, we need to
	// explicitly ask our connection pool for a connection so that it
	// doesn't give us a different one for a subsequent query.
	conn, err := g.db.Conn(ctx)
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
		return nil, err
	}
	args := make([]arg, len(types.pgTypes))[:0]
	for i, t := range types.pgTypes {
		name := fmt.Sprintf("arg%d", i)
		typeInfo, err := g.typeInfoOf(t)
		if err != nil {
			return nil, err
		}
		args = append(args, arg{
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
	buff, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("[]regtype Scan: expected a []byte")
	}

	if buff[0] != '{' || buff[len(buff)-1] != '}' {
		return fmt.Errorf("[]regtype Scan: malformed data '%s'", string(buff))
	}
	buff = buff[1 : len(buff)-1]

	if len(buff) == 0 {
		r.pgTypes = []string{}
		return nil
	}

	for len(buff) > 0 {
		var ty string
		var err error
		ty, buff, err = splitType(buff)
		if err != nil {
			return err
		}
		r.pgTypes = append(r.pgTypes, ty)
	}

	return nil
}

// given a comma separated list of possibly quoted values,
// splitType takes the first one off the `types` slice.
func splitType(types []byte) (ty string, rest []byte, err error) {
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
					rest = []byte{}
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
	rest = []byte{}
	return
}

func overrideNullability(
	cols []colMeta,
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
func (g *Generator) funcArgs(funcName string) ([]arg, error) {
	rows, err := g.db.Query(`
        SELECT f.argname, t.typname
        FROM (
            SELECT unnest(p.proargnames) as argname, unnest(p.proargtypes) as argtype
            FROM pg_proc p
            WHERE p.proname = $1
        ) f
        JOIN pg_type t
            ON (f.argtype = t.oid)
        `, funcName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var args []arg
	i := 1
	for rows.Next() {
		var (
			a          arg
			pgTypeName string
		)
		err = rows.Scan(&a.PgName, &pgTypeName)
		if err != nil {
			return nil, err
		}

		a.Idx = i
		a.GoName = pgToGoName(a.PgName)
		typeInfo, err := g.typeInfoOf(pgTypeName)
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
func (g *Generator) queryReturns(query string) ([]colMeta, error) {
	viewName := randomName("tmp_view")
	view := fmt.Sprintf(
		`CREATE OR REPLACE TEMP VIEW %s AS %s`,
		viewName, nullOutArgs(query),
	)

	_, err := g.db.Exec(view)
	if err != nil {
		return nil, err
	}

	viewMeta, err := g.tableMeta(viewName)
	if err != nil {
		return nil, err
	}

	// This should be totally unneeded, but I have observed the tmp
	// views popping up in psql sessions that were already active
	// when pggen was run. We intentionally don't check the error
	// code here because we really don't care too much if this
	// doesn't work.
	_, err = g.db.Exec(fmt.Sprintf(`DROP VIEW IF EXISTS %s`, viewName))
	if err != nil {
		return nil, err
	}

	return viewMeta.Cols, nil
}

// tableMeta contains metadata about a postgres table
type tableMeta struct {
	PgName string
	GoName string
	// metadata for the primary key column
	PkeyCol *colMeta
	// Metadata about the tables columns
	Cols []colMeta
	// A list of the postgres names of tables which reference this one
	References []refMeta
	// If true, this table does have an update timestamp field
	HasUpdateAtField bool
	// If true, this table does have a create timestamp field
	HasCreatedAtField bool
}

// colMeta contains metadata about postgres table columns such column
// names, types, nullability, default...
type colMeta struct {
	ColNum      int32
	GoName      string
	PgName      string
	PgType      string
	TypeInfo    goTypeInfo
	Nullable    bool
	DefaultExpr string
	IsPrimary   bool
}

// refMeta contains metadata for a reference between two tables
// (a foreign key relationship)
type refMeta struct {
	// The name of the table that this reference is referring to
	PgPointsTo string
	// The name of the go struct which corresponds to PgPointsTo
	GoPointsTo string
	// The names of the fields in the referenced table that are used as keys
	// (usually the primary keys of that table). Order matters.
	PointsToFields []fieldNames
	// The name of the table that is referring to another table
	PgPointsFrom string
	// The name of the go struct which corresponds to PgPointsFrom
	GoPointsFrom string
	// A pluralized version of GoPointsFrom
	PluralGoPointsFrom string
	// The names of the fields that are being used to refer to the key fields
	// for the referenced table. Order matters.
	PointsFromFields []fieldNames
	// Indicates that there can be at most one of these references between
	// the two tables.
	OneToOne bool
}

type fieldNames struct {
	PgName string
	GoName string
}

// Given the name of a table returns metadata about it
func (g *Generator) tableMeta(table string) (tableMeta, error) {
	rows, err := g.db.Query(`
		SELECT
			a.attnum AS col_num,
			a.attname AS col_name,
			format_type(a.atttypid, a.atttypmod) AS col_type,
			NOT a.attnotnull AS nullable,
			COALESCE(pg_get_expr(ad.adbin, ad.adrelid), '') AS default_expr,
			COALESCE(ct.contype = 'p', false) AS is_primary
		FROM pg_attribute a
		INNER JOIN pg_class c
			ON (c.oid = a.attrelid)
		LEFT JOIN pg_constraint ct
			ON (ct.conrelid = c.oid AND a.attnum = ANY(ct.conkey) AND ct.contype = 'p')
		LEFT JOIN pg_attrdef ad
			ON (ad.adrelid = c.oid AND ad.adnum = a.attnum)
		WHERE a.attisdropped = false AND c.relname = $1 AND (a.attnum > 0)
		ORDER BY a.attnum
		`, table)
	if err != nil {
		return tableMeta{}, err
	}

	var cols []colMeta
	for rows.Next() {
		var col colMeta
		err = rows.Scan(
			&col.ColNum,
			&col.PgName,
			&col.PgType,
			&col.Nullable,
			&col.DefaultExpr,
			&col.IsPrimary,
		)
		if err != nil {
			return tableMeta{}, err
		}
		typeInfo, err := g.typeInfoOf(col.PgType)
		if err != nil {
			return tableMeta{}, err
		}
		col.TypeInfo = *typeInfo
		col.GoName = pgToGoName(col.PgName)
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		return tableMeta{}, fmt.Errorf(
			"could not find table '%s' in the database",
			table,
		)
	}

	var pkeyCol *colMeta
	for i, c := range cols {
		if c.IsPrimary {
			if pkeyCol != nil {
				return tableMeta{}, fmt.Errorf("tables with multiple primary keys not supported")
			}

			pkeyCol = &cols[i]
		}
	}

	meta := tableMeta{
		PgName:  table,
		GoName:  pgToGoName(inflection.Singular(table)),
		PkeyCol: pkeyCol,
		Cols:    cols,
	}
	err = g.fillTableReferences(&meta)
	if err != nil {
		return tableMeta{}, err
	}

	return meta, nil
}

// Given a tableMeta with the PgName and Cols already filled out, fill in the
// References list
func (g *Generator) fillTableReferences(meta *tableMeta) error {
	// This runs N+1 queries where N is the number of foreign keys referencing
	// the given table. We might be able to do better with UNNEST hacks, but
	// I'm not sure how worth it that would be. Also, N is likely to be small.
	rows, err := g.db.Query(`
		SELECT
			pt.relname as points_to,
			c.confkey as points_to_keys,
			pf.relname as points_from,
			c.conkey as points_from_keys
		FROM pg_constraint c
		JOIN pg_class pt
			ON (pt.oid = c.confrelid)
		JOIN pg_class pf
			ON (c.conrelid = pf.oid)
		WHERE c.contype = 'f'
		  AND pt.relname = $1
		`, meta.PgName)
	if err != nil {
		return err
	}
	for rows.Next() {
		pointsToIdxs := []int64{}
		pointsFromIdxs := []int64{}
		var ref refMeta
		err = rows.Scan(
			&ref.PgPointsTo, pq.Array(&pointsToIdxs),
			&ref.PgPointsFrom, pq.Array(&pointsFromIdxs),
		)
		if err != nil {
			return err
		}
		ref.GoPointsTo = pgToGoName(inflection.Singular(ref.PgPointsTo))
		ref.PluralGoPointsFrom = pgToGoName(ref.PgPointsFrom)
		ref.GoPointsFrom = inflection.Singular(ref.PluralGoPointsFrom)

		for _, idx := range pointsToIdxs {
			// attnum is 1-based, so we will first convert it into a 0-based
			// index
			idx--

			if idx < 0 || int64(len(meta.Cols)) <= idx {
				return fmt.Errorf("out of bounds foreign key field (to) at index %d", idx)
			}
			ref.PointsToFields = append(ref.PointsToFields, fieldNames{
				PgName: meta.Cols[idx].PgName,
				GoName: meta.Cols[idx].GoName,
			})
		}

		// this call is what makes this routine run N+1 queries
		fromFields, err := g.pointsFromColMeta(ref.PgPointsFrom)
		if err != nil {
			return err
		}

		ref.OneToOne = true
		for _, idx := range pointsFromIdxs {
			idx--

			if idx < 0 || int64(len(fromFields)) <= idx {
				return fmt.Errorf("out of bounds foreign key field (from) at index %d", idx)
			}

			ffield := &fromFields[idx]
			ref.PointsFromFields = append(ref.PointsFromFields, ffield.name)
			ref.OneToOne = ref.OneToOne && ffield.hasUniqueIndex
		}

		meta.References = append(meta.References, ref)
	}

	return nil
}

type pointsFromColMeta struct {
	name           fieldNames
	hasUniqueIndex bool
}

// pointsFromMeta returns the names of a tables columns given the table name
// in postgres
func (g *Generator) pointsFromColMeta(table string) (
	[]pointsFromColMeta,
	error,
) {
	rows, err := g.db.Query(`
		WITH unique_cols AS (
			SELECT
				UNNEST(ix.indkey) as colnum,
				ix.indisunique as is_unique
			FROM pg_class c
			JOIN pg_index ix
				ON (c.oid = ix.indrelid)
			WHERE c.relname = $1
		)

		SELECT a.attname, COALESCE(u.is_unique, 'f'::bool)
		FROM pg_attribute a
		INNER JOIN pg_class c
			ON (c.oid = a.attrelid)
		LEFT JOIN unique_cols u
			ON (u.colnum = a.attnum)
		WHERE a.attisdropped = false
		  AND a.attnum > 0
		  AND c.relname = $1
		`, table)
	if err != nil {
		return nil, err
	}

	cols := []pointsFromColMeta{}

	for rows.Next() {
		var col pointsFromColMeta
		err = rows.Scan(&col.name.PgName, &col.hasUniqueIndex)
		if err != nil {
			return nil, err
		}
		col.name.GoName = pgToGoName(col.name.PgName)
		cols = append(cols, col)
	}

	return cols, nil
}

// Given the oid of a postgres type, return all the variants that
// that enum has.
func (g *Generator) enumVariants(typeName string) ([]string, error) {
	rows, err := g.db.Query(`
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
