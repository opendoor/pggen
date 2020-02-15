package gen

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/jinzhu/inflection"
)

// Generate code for all of the tables
func (g *Generator) genTables(into io.Writer, tables []tableConfig) error {
	if len(tables) > 0 {
		g.infof("	generating %d tables\n", len(tables))
	} else {
		return nil
	}

	g.imports[`"database/sql"`] = true
	g.imports[`"context"`] = true
	g.imports[`"fmt"`] = true
	g.imports[`"math"`] = true
	g.imports[`"github.com/lib/pq"`] = true
	g.imports[`"github.com/willf/bitset"`] = true

	for _, table := range tables {
		err := g.genTable(into, &table)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) genTable(
	into io.Writer,
	table *tableConfig,
) (err error) {
	g.infof("		generating table '%s'\n", table.Name)
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"while generating table '%s': %s", table.Name, err.Error())
		}
	}()

	meta := g.tables[table.Name].meta
	if meta.PkeyCol == nil {
		err = fmt.Errorf("no primary key for table")
		return
	}

	// Filter out all the references from tables that are not
	// mentioned in the TOML, or have explicitly asked us not to
	// infer relationships. We only want to generate code about the
	// part of the database schema that we have been explicitly
	// asked to generate code for.
	kept := 0
	for _, ref := range meta.References {
		if fromTable, inMap := g.tables[ref.PgPointsFrom]; inMap {
			if !fromTable.config.NoInferBelongsTo {
				meta.References[kept] = ref
				kept++
			}
		}

		if len(ref.PointsFromFields) != 1 {
			err = fmt.Errorf("multi-column foreign keys not supported")
			return
		}
	}
	meta.References = meta.References[:kept]

	meta.References = append(
		meta.References,
		g.tables[table.Name].explicitBelongsTo...,
	)

	// Emit the type seperately to prevent double defintions
	var tableType strings.Builder
	err = tableTypeTmpl.Execute(&tableType, meta)
	if err != nil {
		return
	}
	var tableSig strings.Builder
	err = tableTypeFieldSigTmpl.Execute(&tableSig, meta)
	if err != nil {
		return
	}
	err = g.types.emitType(meta.GoName, tableSig.String(), tableType.String())
	if err != nil {
		return
	}

	return tableShimTmpl.Execute(into, meta)
}

var tableTypeFieldSigTmpl *template.Template = template.Must(template.New("table-type-field-sig-tmpl").Parse(`
{{- range .Cols }}
{{- if .Nullable }}
{{ .GoName }} {{ .TypeInfo.NullName }}
{{- else }}
{{ .GoName }} {{ .TypeInfo.Name }}
{{- end }}
{{- end }}
`))

var tableTypeTmpl *template.Template = template.Must(template.New("table-type-tmpl").Parse(`
type {{ .GoName }} struct {
	{{- range .Cols }}
	{{- if .Nullable }}
	{{ .GoName }} {{ .TypeInfo.NullName }}
	{{- else }}
	{{ .GoName }} {{ .TypeInfo.Name }}
	{{- end }} ` +
	"`" + `gorm:"column:{{ .PgName }}"
	{{- if .IsPrimary }} gorm:"is_primary" {{- end }}` +
	"`" + `
	{{- end }}
	{{- range .References }}
	{{- if .OneToOne }}
	{{ .GoPointsFrom }} *{{ .GoPointsFrom }}
	{{- else }}
	{{ .PluralGoPointsFrom }} []{{ .GoPointsFrom }}
	{{- end }}
	{{- end }}
}
func (r *{{ .GoName }}) Scan(rs *sql.Rows) error {
	return rs.Scan(
		{{- range .Cols }}
		{{ call .TypeInfo.SqlReceiver (printf "r.%s" .GoName) }},
		{{- end }}
	)
}
`))

var tableShimTmpl *template.Template = template.Must(template.New("table-shim-tmpl").Parse(`

func (p *PGClient) Get{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
) ({{ .GoName }}, error) {
	values, err := p.List{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id})
	if err != nil {
		return {{ .GoName }}{}, err
	}

	// List{{ .GoName }} always returns the same number of records as were
	// requested, so this is safe.
	return values[0], err
}

func (p *PGClient) List{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) ([]{{ .GoName }}, error) {
	rows, err := p.DB.QueryContext(
		ctx,
		"SELECT * FROM \"{{ .PgName }}\" WHERE {{ .PkeyCol.PgName }} = ANY($1)",
		pq.Array(ids),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := make([]{{ .GoName }}, len(ids))[:0]
	for rows.Next() {
		var value {{ .GoName }}
		err = value.Scan(rows)
		if err != nil {
			return nil, err
		}
		ret = append(ret, value)
	}

	if len(ret) != len(ids) {
		return nil, fmt.Errorf(
			"List{{ .GoName }}: asked for %d records, found %d",
			len(ids),
			len(ret),
		)
	}

	return ret, nil
}

// Insert a {{ .GoName }} into the database. Returns the primary
// key of the inserted row.
func (p *PGClient) Insert{{ .GoName }}(
	ctx context.Context,
	value {{ .GoName }},
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var ids []{{ .PkeyCol.TypeInfo.Name }}
	ids, err = p.BulkInsert{{ .GoName }}(ctx, []{{ .GoName }}{value})
	if err != nil {
		return
	}

	if len(ids) != 1 {
		err = fmt.Errorf("inserting a {{ .GoName }}: %d ids (expected 1)", len(ids))
		return
	}

	ret = ids[0]
	return
}
// Insert a list of {{ .GoName }}. Returns a list of the primary keys of
// the inserted rows.
func (p *PGClient) BulkInsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	var fields []string = []string{
		{{- range .Cols }}
		{{- if (not .IsPrimary) }}
		"{{ .PgName }}",
		{{- end }}
		{{- end }}
	}

	args := make([]interface{}, {{ len .Cols }} * len(values))[:0]
	for _, v := range values {
		{{- range .Cols }}
		{{- if (not .IsPrimary) }}
		args = append(args, v.{{ .GoName }})
		{{- end }}
		{{- end }}
	}

	bulkInsertQuery := genBulkInsert(
		"{{ .PgName }}",
		fields,
		len(values),
		"{{ .PkeyCol.PgName }}",
	)

	rows, err := p.DB.QueryContext(ctx, bulkInsertQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]{{ .PkeyCol.TypeInfo.Name }}, len(values))[:0]
	for rows.Next() {
		var id {{ .PkeyCol.TypeInfo.Name }}
		err = rows.Scan({{ call .PkeyCol.TypeInfo.SqlReceiver "id" }})
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// bit indicies for 'fieldMask' parameters
const (
	{{- range $i, $c := .Cols }}
	{{ $.GoName }}{{ $c.GoName }}FieldIndex uint = {{ $i }}
	{{- end }}
)

// A bitset saying that all fields in {{ .GoName }} should be updated.
// For use as a 'fieldMask' parameter
var {{ .GoName }}AllFields *bitset.BitSet = func() *bitset.BitSet {
	ret := bitset.New({{ len .Cols }})
	var i uint
	for i = 0; i < uint({{ len .Cols }}); i++ {
		ret.Set(i)
	}
	return ret
}()

// Update a {{ .GoName }}. 'value' must at the least have
// a primary key set. The 'fieldMask' bitset indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (p *PGClient) Update{{ .GoName }}(
	ctx context.Context,
	value {{ .GoName }},
	fieldMask *bitset.BitSet,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var fields []string = []string{
		{{- range .Cols }}
		"{{ .PgName }}",
		{{- end }}
	}

	if !fieldMask.Test({{ .GoName }}{{ .PkeyCol.GoName }}FieldIndex) {
		err = fmt.Errorf("primary key required for updates to '{{ .PgName }}'")
		return
	}

	updateStmt := genUpdateStmt(
		"{{ .PgName }}",
		"{{ .PkeyCol.PgName }}",
		fields,
		fieldMask,
		"{{ .PkeyCol.PgName }}",
	)

	args := make([]interface{}, {{ len .Cols }})[:0]

	{{- range .Cols }}
	if fieldMask.Test({{ $.GoName }}{{ .GoName }}FieldIndex) {
		args = append(args, value.{{ .GoName }})
	}
	{{- end }}

	// add the primary key arg for the WHERE condition
	args = append(args, value.{{ .PkeyCol.GoName }})

	var id {{ .PkeyCol.TypeInfo.Name }}
	err = p.DB.QueryRowContext(ctx, updateStmt, args...).
                Scan({{ call .PkeyCol.TypeInfo.SqlReceiver "id" }})
	if err != nil {
		return
	}

	return id, nil
}

func (p *PGClient) Delete{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
) error {
	return p.BulkDelete{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id})
}

func (p *PGClient) BulkDelete{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) error {
	res, err := p.DB.ExecContext(
		ctx,
		"DELETE FROM \"{{ .PgName }}\" WHERE {{ .PkeyCol.PgName }} = ANY($1)",
		pq.Array(ids),
	)
	if err != nil {
		return err
	}

	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if nrows != int64(len(ids)) {
		return fmt.Errorf(
			"BulkDelete{{ .GoName }}: %d rows deleted, expected %d",
			nrows,
			len(ids),
		)
	}

	return err
}

func (p *PGClient) {{ .GoName }}FillAll(
	ctx context.Context,
	rec *{{ .GoName }},
) error {
	return p.{{ .GoName }}FillToDepth(ctx, []*{{ .GoName }}{rec}, math.MaxInt64)
}

func (p *PGClient) {{ .GoName }}FillToDepth(
	ctx context.Context,
	recs []*{{ .GoName }},
	maxDepth int64,
) (err error) {
	if maxDepth <= 0 {
		return
	}
{{- range .References }}

	// Fill in the {{ .PluralGoPointsFrom }}
	{{- if .OneToOne }}
	err = p.{{ $.GoName }}Fill{{ .GoPointsFrom }}(ctx, recs)
	{{- else }}
	err = p.{{ $.GoName }}Fill{{ .PluralGoPointsFrom }}(ctx, recs)
	{{- end }}
	if err != nil {
		return
	}
	var sub{{ .PluralGoPointsFrom }} []*{{ .GoPointsFrom }}
	for _, outer := range recs {
		{{- if .OneToOne }}
		sub{{ .PluralGoPointsFrom }} = append(sub{{ .PluralGoPointsFrom }}, outer.{{ .GoPointsFrom }})
		{{- else }}
		for i, _ := range outer.{{ .PluralGoPointsFrom }} {
			sub{{ .PluralGoPointsFrom }} = append(sub{{ .PluralGoPointsFrom }}, &outer.{{ .PluralGoPointsFrom }}[i])
		}
		{{- end }}
	}
	err = p.{{ .GoPointsFrom }}FillToDepth(ctx, sub{{ .PluralGoPointsFrom }}, maxDepth - 1)
	if err != nil {
		return
	}
{{- end }}

	return
}
{{- range .References }}

// For a give set of {{ $.GoName }}, fill in all the {{ .GoPointsFrom }}
// connected to them using a single query.
{{- if .OneToOne }}
func (p *PGClient) {{ $.GoName }}Fill{{ .GoPointsFrom }}(
{{- else }}
func (p *PGClient) {{ $.GoName }}Fill{{ .PluralGoPointsFrom }}(
{{- end }}
	ctx context.Context,
	parentRecs []*{{ $.GoName }},
) error {
	ids := make([]{{ $.PkeyCol.TypeInfo.Name }}, len(parentRecs))[:0]
	idToRecord := map[{{ $.PkeyCol.TypeInfo.Name }}]*{{ $.GoName }}{}
	for i, elem := range parentRecs {
		ids = append(ids, elem.{{ $.PkeyCol.GoName }})
		idToRecord[elem.{{ $.PkeyCol.GoName }}] = parentRecs[i]
	}

	rows, err := p.DB.QueryContext(
		ctx,
		` + "`" +
	`SELECT * FROM "{{ .PgPointsFrom }}"
		 WHERE "{{ (index .PointsFromFields 0).PgName }}" = ANY($1)
		 {{- if .OneToOne }}
		 LIMIT 1
		 {{- end }}
		 ` +
	"`" + `,
		pq.Array(ids),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	// pull all the child records from the database and associate them with
	// the correct parent.
	for rows.Next() {
		var childRec {{ .GoPointsFrom }}
		err = childRec.Scan(rows)
		if err != nil {
			return err
		}

		parentRec := idToRecord[childRec.{{ (index .PointsFromFields 0).GoName }}]
		{{- if .OneToOne }}
		parentRec.{{ .GoPointsFrom }} = &childRec
		break
		{{- else }}
		parentRec.{{ .PluralGoPointsFrom }} = append(parentRec.{{ .PluralGoPointsFrom }}, childRec)
		{{- end }}
	}

	return nil
}

{{ end }}
`))

// Information about tables required for code generation.
//
// The reason there is both a *Meta and *GenInfo struct for tables
// is that `tableMeta` is meant to be narrowly focused on metadata
// that postgres provides us, while things in `tableGenInfo` are
// more specific to `pggen`'s internal needs.
type tableGenInfo struct {
	config *tableConfig
	// Table relationships that have been explicitly configured
	// rather than infered from the database schema itself.
	explicitBelongsTo []refMeta
	meta              tableMeta
}

// nullFlags computes the null flags specifying the nullness of this
// table in the same format used by the `null_flags` config option
func (info tableGenInfo) nullFlags() string {
	nf := make([]byte, len(info.meta.Cols))[:0]
	for _, c := range info.meta.Cols {
		if c.Nullable {
			nf = append(nf, 'n')
		} else {
			nf = append(nf, '-')
		}
	}
	return string(nf)
}

func (g *Generator) populateTableInfo(tables []tableConfig) error {
	g.tables = map[string]tableGenInfo{}
	g.tableTyNameToTableName = map[string]string{}
	for i, table := range tables {
		info := tableGenInfo{}
		info.config = &tables[i]

		meta, err := g.tableMeta(table.Name)
		if err != nil {
			return fmt.Errorf("table '%s': %s", table.Name, err.Error())
		}
		info.meta = meta

		g.tables[table.Name] = info
		g.tableTyNameToTableName[meta.GoName] = meta.PgName
	}

	err := buildExplicitBelongsToMapping(tables, g.tables)
	if err != nil {
		return err
	}

	return nil
}

func buildExplicitBelongsToMapping(
	tables []tableConfig,
	infoTab map[string]tableGenInfo,
) error {
	for _, table := range tables {
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

			pointsToMeta := infoTab[belongsTo.Table].meta
			ref := refMeta{
				PgPointsTo: belongsTo.Table,
				GoPointsTo: snakeToPascal(inflection.Singular(belongsTo.Table)),
				PointsToFields: []fieldNames{
					{
						PgName: pointsToMeta.PkeyCol.PgName,
						GoName: pointsToMeta.PkeyCol.GoName,
					},
				},
				PgPointsFrom:       table.Name,
				GoPointsFrom:       snakeToPascal(inflection.Singular(table.Name)),
				PluralGoPointsFrom: snakeToPascal(table.Name),
				PointsFromFields: []fieldNames{
					{
						PgName: belongsTo.KeyField,
						GoName: snakeToPascal(belongsTo.KeyField),
					},
				},
				OneToOne: belongsTo.OneToOne,
			}

			info := infoTab[belongsTo.Table]
			info.explicitBelongsTo = append(info.explicitBelongsTo, ref)
			infoTab[belongsTo.Table] = info
		}
	}

	return nil
}
