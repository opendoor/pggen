package meta

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/opendoor-labs/pggen/gen/internal/config"
	"github.com/opendoor-labs/pggen/gen/internal/log"
	"github.com/opendoor-labs/pggen/gen/internal/names"
	"github.com/opendoor-labs/pggen/gen/internal/types"
	"github.com/opendoor-labs/pggen/include"
)

// tablesMeta contains information _all_ of the tables that pggen is awair of
type tablesMeta struct {
	// A table mapping go type name for a table struct to the postgres
	// name for that table.
	tableTyNameToTableName map[string]string
	// A mapping from the postgres table name to information about the table
	tableInfo map[string]*TableGenInfo
}

type tableResolver struct {
	meta         tablesMeta
	db           *sql.DB
	log          *log.Logger
	typeResolver *types.Resolver
}

func newTableResolver(l *log.Logger, db *sql.DB, typeResolver *types.Resolver) *tableResolver {
	return &tableResolver{
		meta: tablesMeta{
			tableTyNameToTableName: map[string]string{},
			tableInfo:              map[string]*TableGenInfo{},
		},
		log:          l,
		typeResolver: typeResolver,
		db:           db,
	}
}

// Information about a single table required for code generation.
//
// The reason there is both a *Meta and *GenInfo struct for tables
// is that `tableMeta` is meant to be narrowly focused on metadata
// that postgres provides us, while things in `TableGenInfo` are
// more specific to `pggen`'s internal needs.
type TableGenInfo struct {
	Config *config.TableConfig
	// Table relationships that have been explicitly configured
	// rather than infered from the database schema itself.
	ExplicitBelongsTo []RefMeta
	// The include spec which represents the transitive closure of
	// this tables family
	AllIncludeSpec *include.Spec
	// If true, this table does have an update timestamp field
	HasUpdateAtField bool
	// True if the update at field can be null
	UpdatedAtFieldIsNullable bool
	// True if the updated at field has a time zone
	UpdatedAtHasTimezone bool
	// If true, this table does have a create timestamp field
	HasCreatedAtField bool
	// True if the created at field can be null
	CreatedAtFieldIsNullable bool
	// True if the created at field has a time zone
	CreatedAtHasTimezone bool
	// The table metadata as postgres reports it
	Meta tableMeta
}

// nullFlags computes the null flags specifying the nullness of this
// table in the same format used by the `null_flags` config option
func (info TableGenInfo) nullFlags() string {
	nf := make([]byte, 0, len(info.Meta.Cols))
	for _, c := range info.Meta.Cols {
		if c.Nullable {
			nf = append(nf, 'n')
		} else {
			nf = append(nf, '-')
		}
	}
	return string(nf)
}

func (tr *tableResolver) populateTableInfo(tables []config.TableConfig) error {
	tr.meta.tableInfo = map[string]*TableGenInfo{}
	tr.meta.tableTyNameToTableName = map[string]string{}
	for i, table := range tables {
		info := &TableGenInfo{}
		info.Config = &tables[i]

		meta, err := tr.tableMeta(table.Name)
		if err != nil {
			return fmt.Errorf("table '%s': %s", table.Name, err.Error())
		}
		info.Meta = meta

		tr.meta.tableInfo[table.Name] = info
		tr.meta.tableTyNameToTableName[meta.GoName] = meta.PgName
	}

	// fill in all the reference we can automatically detect
	for _, table := range tr.meta.tableInfo {
		err := tr.fillTableReferences(&table.Meta)
		if err != nil {
			return err
		}
	}

	err := tr.buildExplicitBelongsToMapping(tables, tr.meta.tableInfo)
	if err != nil {
		return err
	}

	// fill in all the allIncludeSpecs
	for _, info := range tr.meta.tableInfo {
		err := ensureSpec(tr.meta.tableInfo, info)
		if err != nil {
			return err
		}
	}

	for _, info := range tr.meta.tableInfo {
		tr.setTimestampFlags(info)
	}

	return nil
}

func (tr *tableResolver) setTimestampFlags(info *TableGenInfo) {
	if len(info.Config.CreatedAtField) > 0 {
		for _, cm := range info.Meta.Cols {
			if cm.PgName == info.Config.CreatedAtField {
				info.HasCreatedAtField = true
				info.CreatedAtFieldIsNullable = cm.Nullable
				info.CreatedAtHasTimezone = cm.TypeInfo.IsTimestampWithZone
				break
			}
		}

		if !info.HasCreatedAtField {
			tr.log.Warnf(
				"table '%s' has no '%s' created at timestamp\n",
				info.Config.Name,
				info.Config.CreatedAtField,
			)
		}
	}

	if len(info.Config.UpdatedAtField) > 0 {
		for _, cm := range info.Meta.Cols {
			if cm.PgName == info.Config.UpdatedAtField {
				info.HasUpdateAtField = true
				info.UpdatedAtFieldIsNullable = cm.Nullable
				info.UpdatedAtHasTimezone = cm.TypeInfo.IsTimestampWithZone
				break
			}
		}

		if !info.HasUpdateAtField {
			tr.log.Warnf(
				"table '%s' has no '%s' updated at timestamp\n",
				info.Config.Name,
				info.Config.UpdatedAtField,
			)
		}
	}
}

func ensureSpec(tables map[string]*TableGenInfo, info *TableGenInfo) error {
	if info.AllIncludeSpec != nil {
		// Some other `ensureSpec` already filled this in for us. Great!
		return nil
	}

	info.AllIncludeSpec = &include.Spec{
		TableName: info.Meta.PgName,
		Includes:  map[string]*include.Spec{},
	}

	ensureReferencedSpec := func(ref *RefMeta) error {
		subInfo := tables[ref.PointsFrom.PgName]
		if subInfo == nil {
			// This table is referenced in the database schema but not in the
			// config file.
			return nil
		}

		err := ensureSpec(tables, subInfo)
		if err != nil {
			return err
		}
		subSpec := subInfo.AllIncludeSpec
		info.AllIncludeSpec.Includes[subSpec.TableName] = subSpec

		return nil
	}

	for _, ref := range info.Meta.References {
		err := ensureReferencedSpec(&ref)
		if err != nil {
			return err
		}
	}
	for _, ref := range info.ExplicitBelongsTo {
		err := ensureReferencedSpec(&ref)
		if err != nil {
			return err
		}
	}

	if len(info.AllIncludeSpec.Includes) == 0 {
		info.AllIncludeSpec.Includes = nil
	}

	return nil
}

func (tr *tableResolver) buildExplicitBelongsToMapping(
	tables []config.TableConfig,
	infoTab map[string]*TableGenInfo,
) error {
	for _, table := range tables {
		pointsFromTable := tr.meta.tableInfo[table.Name]

		for _, belongsTo := range table.BelongsTo {
			if len(belongsTo.Table) == 0 {
				return fmt.Errorf(
					"%s: belongs_to requires 'name' key",
					table.Name,
				)
			}

			if len(belongsTo.KeyField) == 0 {
				return fmt.Errorf(
					"%s: belongs_to requires 'key_field' key",
					table.Name,
				)
			}

			var belongsToColMeta *ColMeta
			for i, col := range pointsFromTable.Meta.Cols {
				if col.PgName == belongsTo.KeyField {
					belongsToColMeta = &pointsFromTable.Meta.Cols[i]
				}
			}
			if belongsToColMeta == nil {
				return fmt.Errorf(
					"table '%s' has no field '%s'",
					table.Name,
					belongsTo.KeyField,
				)
			}

			pointsToMeta := infoTab[belongsTo.Table].Meta
			ref := RefMeta{
				PointsTo:         &tr.meta.tableInfo[belongsTo.Table].Meta,
				PointsToFields:   []*ColMeta{pointsToMeta.PkeyCol},
				PointsFrom:       &tr.meta.tableInfo[table.Name].Meta,
				PointsFromFields: []*ColMeta{belongsToColMeta},
				OneToOne:         belongsTo.OneToOne,
				Nullable:         belongsToColMeta.Nullable,
			}

			info := infoTab[belongsTo.Table]
			info.ExplicitBelongsTo = append(info.ExplicitBelongsTo, ref)
			infoTab[belongsTo.Table] = info
		}
	}

	return nil
}

//
// queries
//

// tableMeta contains metadata about a postgres table
type tableMeta struct {
	PgName       string
	GoName       string
	PluralGoName string
	// metadata for the primary key column
	PkeyCol *ColMeta
	// Metadata about the tables columns
	Cols []ColMeta
	// A list of the postgres names of tables which reference this one
	References []RefMeta
	// If true, this table does have an update timestamp field
	HasUpdateAtField bool
	// If true, this table does have a create timestamp field
	HasCreatedAtField bool
	// The 0-based index of the primary key column
	PkeyColIdx int
}

// colMeta contains metadata about postgres table columns such column
// names, types, nullability, default...
type ColMeta struct {
	// postgres's internal column number for this column
	ColNum int32
	// the name of the field in the go struct which corresponds to this column
	GoName string
	// the name of this column in postgres
	PgName string
	// name of the type of this column
	PgType string
	// a more descriptive record of the type of this column
	TypeInfo types.Info
	// true if this column can be null
	Nullable bool
	// the postgres default value for this column
	DefaultExpr string
	// true if this column is the primary key for this table
	IsPrimary bool
	// true if this column has a UNIQUE index on it
	IsUnique bool
}

// Given the name of a table returns metadata about it
func (tr *tableResolver) tableMeta(table string) (tableMeta, error) {
	rows, err := tr.db.Query(`
		WITH unique_cols AS (
			SELECT
				UNNEST(ix.indkey) as colnum,
				ix.indisunique as is_unique
			FROM pg_class c
			JOIN pg_index ix
				ON (c.oid = ix.indrelid)
			WHERE c.relname = $1
		)

		SELECT DISTINCT ON (a.attnum)
			a.attnum AS col_num,
			a.attname AS col_name,
			format_type(a.atttypid, a.atttypmod) AS col_type,
			NOT a.attnotnull AS nullable,
			COALESCE(pg_get_expr(ad.adbin, ad.adrelid), '') AS default_expr,
			COALESCE(ct.contype = 'p', false) AS is_primary,
			COALESCE(u.is_unique, 'f'::bool) AS is_unique
		FROM pg_attribute a
		INNER JOIN pg_class c
			ON (c.oid = a.attrelid)
		LEFT JOIN pg_constraint ct
			ON (ct.conrelid = c.oid AND a.attnum = ANY(ct.conkey) AND ct.contype = 'p')
		LEFT JOIN pg_attrdef ad
			ON (ad.adrelid = c.oid AND ad.adnum = a.attnum)
		LEFT JOIN unique_cols u
			ON (u.colnum = a.attnum)
		WHERE a.attisdropped = false AND c.relname = $1 AND (a.attnum > 0)
		ORDER BY a.attnum
		`, table)
	if err != nil {
		return tableMeta{}, err
	}

	var cols []ColMeta
	for rows.Next() {
		var col ColMeta
		err = rows.Scan(
			&col.ColNum,
			&col.PgName,
			&col.PgType,
			&col.Nullable,
			&col.DefaultExpr,
			&col.IsPrimary,
			&col.IsUnique,
		)
		if err != nil {
			return tableMeta{}, err
		}
		typeInfo, err := tr.typeResolver.TypeInfoOf(col.PgType)
		if err != nil {
			return tableMeta{}, fmt.Errorf("column '%s': %s", col.PgName, err.Error())
		}
		col.TypeInfo = *typeInfo
		col.GoName = names.PgToGoName(col.PgName)
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		return tableMeta{}, fmt.Errorf(
			"could not find table '%s' in the database",
			table,
		)
	}

	var (
		pkeyCol    *ColMeta
		pkeyColIdx int
	)
	for i, c := range cols {
		if c.IsPrimary {
			if pkeyCol != nil {
				return tableMeta{}, fmt.Errorf("tables with multiple primary keys not supported")
			}

			pkeyCol = &cols[i]
			pkeyColIdx = i
		}
	}

	return tableMeta{
		PgName:       table,
		GoName:       names.PgTableToGoModel(table),
		PluralGoName: names.PgToGoName(table),
		PkeyCol:      pkeyCol,
		PkeyColIdx:   pkeyColIdx,
		Cols:         cols,
	}, nil
}

// Given a tableMeta with the PgName and Cols already filled out, fill in the
// References list. Any tables which are referenced by the given table must
// already be loaded into `g.tables`.
func (tr *tableResolver) fillTableReferences(meta *tableMeta) error {
	rows, err := tr.db.Query(`
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

	metaColNumToIdx := columnResolverTable(meta.Cols)

	for rows.Next() {

		var (
			pgPointsTo   string
			pgPointsFrom string
		)

		pointsToIdxs := []int64{}
		pointsFromIdxs := []int64{}
		var ref RefMeta
		err = rows.Scan(
			&pgPointsTo, pq.Array(&pointsToIdxs),
			&pgPointsFrom, pq.Array(&pointsFromIdxs),
		)
		if err != nil {
			return err
		}

		_, inTOMLConfig := tr.meta.tableInfo[pgPointsFrom]
		if !inTOMLConfig {
			continue
		}

		for _, idx := range pointsToIdxs {
			// convert the ColNum to an index into the Cols array
			if idx < 0 || int64(len(metaColNumToIdx)) <= idx {
				return fmt.Errorf("out of bounds foreign key field (to) at index %d", idx)
			}
			idx = int64(metaColNumToIdx[idx])

			ref.PointsToFields = append(ref.PointsToFields, &meta.Cols[idx])
		}

		ref.PointsTo = &tr.meta.tableInfo[pgPointsTo].Meta
		ref.PointsFrom = &tr.meta.tableInfo[pgPointsFrom].Meta

		fromCols := ref.PointsFrom.Cols
		fromColsColNumToIdx := columnResolverTable(fromCols)

		ref.OneToOne = true
		for _, idx := range pointsFromIdxs {
			if idx < 0 || int64(len(fromColsColNumToIdx)) <= idx {
				return fmt.Errorf("out of bounds foreign key field (from) at index %d", idx)
			}
			idx = int64(fromColsColNumToIdx[idx])

			fcol := &fromCols[idx]
			ref.PointsFromFields = append(ref.PointsFromFields, fcol)
			ref.OneToOne = ref.OneToOne && fcol.IsUnique
			ref.Nullable = fcol.Nullable
		}

		meta.References = append(meta.References, ref)
	}

	return nil
}
