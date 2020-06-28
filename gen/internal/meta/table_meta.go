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
	tableInfo map[string]*TableMeta
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
			tableInfo:              map[string]*TableMeta{},
		},
		log:          l,
		typeResolver: typeResolver,
		db:           db,
	}
}

// TableMeta contains information about a single table required for
// code generation.
//
// The reason there is both a *Meta and *Info struct for tables
// is that `PgTableInfo` is meant to be narrowly focused on metadata
// that postgres provides us, while things in `TableMeta` are
// more specific to `pggen`'s internal needs and contain some computed
// fields.
type TableMeta struct {
	Config *config.TableConfig
	// All references to this table (both infered and configured).
	AllReferences []RefMeta
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
	Info PgTableInfo
}

// nullFlags computes the null flags specifying the nullness of this
// table in the same format used by the `null_flags` config option
func (info TableMeta) nullFlags() string {
	nf := make([]byte, 0, len(info.Info.Cols))
	for _, c := range info.Info.Cols {
		if c.Nullable {
			nf = append(nf, 'n')
		} else {
			nf = append(nf, '-')
		}
	}
	return string(nf)
}

func (tr *tableResolver) populateTableInfo(tables []config.TableConfig) error {
	tr.meta.tableInfo = map[string]*TableMeta{}
	tr.meta.tableTyNameToTableName = map[string]string{}
	for i, table := range tables {
		info := &TableMeta{}
		info.Config = &tables[i]

		meta, err := tr.tableInfo(table.Name)
		if err != nil {
			return fmt.Errorf("table '%s': %s", table.Name, err.Error())
		}
		info.Info = meta

		tr.meta.tableInfo[table.Name] = info
		tr.meta.tableTyNameToTableName[meta.GoName] = meta.PgName
	}

	// fill in all the reference we can automatically detect
	for _, table := range tr.meta.tableInfo {
		err := tr.fillTableReferences(&table.Info)
		if err != nil {
			return err
		}
	}

	// copy the auto-detected references over and add the explicitly configured
	// ones into the mix
	err := tr.buildAllReferencesMapping(tables, tr.meta.tableInfo)
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

func (tr *tableResolver) setTimestampFlags(info *TableMeta) {
	if len(info.Config.CreatedAtField) > 0 {
		for _, cm := range info.Info.Cols {
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
		for _, cm := range info.Info.Cols {
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

func ensureSpec(tables map[string]*TableMeta, info *TableMeta) error {
	if info.AllIncludeSpec != nil {
		// Some other `ensureSpec` already filled this in for us. Great!
		return nil
	}

	info.AllIncludeSpec = &include.Spec{
		TableName: info.Info.PgName,
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
		info.AllIncludeSpec.Includes[ref.PgPointsFromFieldName] = subSpec

		return nil
	}

	for _, ref := range info.AllReferences {
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

func (tr *tableResolver) buildAllReferencesMapping(
	tables []config.TableConfig,
	infoTab map[string]*TableMeta,
) error {
	// a mapping of table names to sets of reference names which were
	// infered rather than explicitly configured.
	inferedReferencs := map[string]map[string]bool{}
	for _, table := range tables {
		meta := infoTab[table.Name]
		inferedReferencs[table.Name] = map[string]bool{}
		for _, ref := range meta.Info.References {
			refererMeta := infoTab[ref.PointsFrom.PgName]

			// don't pass the infered relationship along if we've been asked not to
			if !refererMeta.Config.NoInferBelongsTo {
				inferedReferencs[table.Name][ref.PointsFrom.PgName] = true
			}
		}
	}

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
			for i, col := range pointsFromTable.Info.Cols {
				if col.PgName == belongsTo.KeyField {
					belongsToColMeta = &pointsFromTable.Info.Cols[i]
				}
			}
			if belongsToColMeta == nil {
				return fmt.Errorf(
					"table '%s' has no field '%s'",
					table.Name,
					belongsTo.KeyField,
				)
			}

			pgPointsFromFieldName := belongsTo.ParentFieldName
			goPointsFromFieldName := names.PgToGoName(belongsTo.ParentFieldName)
			if pgPointsFromFieldName == "" {
				info := &tr.meta.tableInfo[table.Name].Info
				if belongsTo.OneToOne {
					goPointsFromFieldName = info.GoName
				} else {
					goPointsFromFieldName = info.PluralGoName
				}
				pgPointsFromFieldName = info.PgName
			}

			pointsToMeta := infoTab[belongsTo.Table].Info
			ref := RefMeta{
				PointsTo:              &tr.meta.tableInfo[belongsTo.Table].Info,
				PointsToFields:        []*ColMeta{pointsToMeta.PkeyCol},
				PointsFrom:            &tr.meta.tableInfo[table.Name].Info,
				PointsFromFields:      []*ColMeta{belongsToColMeta},
				GoPointsFromFieldName: goPointsFromFieldName,
				PgPointsFromFieldName: pgPointsFromFieldName,
				OneToOne:              belongsTo.OneToOne,
				Nullable:              belongsToColMeta.Nullable,
			}
			// prevent inference when we have an explicit config
			inferedReferencs[belongsTo.Table][ref.PointsFrom.PgName] = false

			info := infoTab[belongsTo.Table]
			info.AllReferences = append(info.AllReferences, ref)
			infoTab[belongsTo.Table] = info
		}
	}

	// fill in with infered references that have not been overridden by an
	// explicit config
	for _, table := range tables {
		meta := infoTab[table.Name]

		for _, ref := range meta.Info.References {
			if inferedReferencs[table.Name][ref.PointsFrom.PgName] {
				meta.AllReferences = append(meta.AllReferences, ref)
			}
		}
	}

	return nil
}

//
// queries
//

// PgTableInfo contains metadata about a postgres table that we get directly
// from postgres. Contrast with the `TableMeta` struct which also contains
// computed fields that are needed for codegen.
type PgTableInfo struct {
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

// ColMeta contains metadata about postgres table columns such column
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
func (tr *tableResolver) tableInfo(table string) (PgTableInfo, error) {
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
		return PgTableInfo{}, err
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
			return PgTableInfo{}, err
		}
		typeInfo, err := tr.typeResolver.TypeInfoOf(col.PgType)
		if err != nil {
			return PgTableInfo{}, fmt.Errorf("column '%s': %s", col.PgName, err.Error())
		}
		col.TypeInfo = *typeInfo
		col.GoName = names.PgToGoName(col.PgName)
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		return PgTableInfo{}, fmt.Errorf(
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
				return PgTableInfo{}, fmt.Errorf("tables with multiple primary keys not supported")
			}

			pkeyCol = &cols[i]
			pkeyColIdx = i
		}
	}

	return PgTableInfo{
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
func (tr *tableResolver) fillTableReferences(meta *PgTableInfo) error {
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

		ref.PointsTo = &tr.meta.tableInfo[pgPointsTo].Info
		ref.PointsFrom = &tr.meta.tableInfo[pgPointsFrom].Info

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

		// generate a name to use to refer to the referencing table
		if ref.OneToOne {
			ref.GoPointsFromFieldName = ref.PointsFrom.GoName
		} else {
			ref.GoPointsFromFieldName = ref.PointsFrom.PluralGoName
		}
		ref.PgPointsFromFieldName = ref.PointsFrom.PgName

		meta.References = append(meta.References, ref)
	}

	return nil
}
