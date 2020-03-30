package gen

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/jinzhu/inflection"

	"github.com/opendoor-labs/pggen/include"
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
	g.imports[`"github.com/lib/pq"`] = true
	g.imports[`"github.com/opendoor-labs/pggen/include"`] = true
	g.imports[`"github.com/opendoor-labs/pggen"`] = true

	for _, table := range tables {
		err := g.genTable(into, &table)
		if err != nil {
			return err
		}
	}

	return nil
}

type tableGenCtx struct {
	// taken from tableMeta
	PgName string
	// taken from tableMeta
	GoName string
	// taken from tableMeta
	PkeyCol *colMeta
	// taken from tableMeta
	Cols []colMeta
	// taken from tableMeta
	References []refMeta
	// The include spec which represents the transitive closure of
	// this tables family
	AllIncludeSpec           string
	HasCreatedAtField        bool
	CreatedAtFieldIsNullable bool
	CreatedAtHasTimezone     bool
	CreatedAtField           string
	HasUpdatedAtField        bool
	UpdatedAtHasTimezone     bool
	UpdatedAtFieldIsNullable bool
	UpdatedAtField           string
}

func tableGenCtxFromInfo(info *tableGenInfo) tableGenCtx {
	return tableGenCtx{
		PgName:         info.meta.PgName,
		GoName:         info.meta.GoName,
		PkeyCol:        info.meta.PkeyCol,
		Cols:           info.meta.Cols,
		References:     info.meta.References,
		AllIncludeSpec: info.allIncludeSpec.String(),

		HasCreatedAtField:        info.hasCreatedAtField,
		CreatedAtField:           pgToGoName(info.config.CreatedAtField),
		CreatedAtFieldIsNullable: info.createdAtFieldIsNullable,
		CreatedAtHasTimezone:     info.createdAtHasTimezone,

		HasUpdatedAtField:        info.hasUpdateAtField,
		UpdatedAtField:           pgToGoName(info.config.UpdatedAtField),
		UpdatedAtFieldIsNullable: info.updatedAtFieldIsNullable,
		UpdatedAtHasTimezone:     info.updatedAtHasTimezone,
	}
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

	tableInfo := g.tables[table.Name]

	genCtx := tableGenCtxFromInfo(tableInfo)
	if genCtx.PkeyCol == nil {
		err = fmt.Errorf("no primary key for table")
		return
	}

	// Filter out all the references from tables that are not
	// mentioned in the TOML, or have explicitly asked us not to
	// infer relationships. We only want to generate code about the
	// part of the database schema that we have been explicitly
	// asked to generate code for.
	kept := 0
	for _, ref := range genCtx.References {
		if fromTable, inMap := g.tables[ref.PgPointsFrom]; inMap {
			if !fromTable.config.NoInferBelongsTo {
				genCtx.References[kept] = ref
				kept++
			}
		}

		if len(ref.PointsFromFields) != 1 {
			err = fmt.Errorf("multi-column foreign keys not supported")
			return
		}
	}
	genCtx.References = genCtx.References[:kept]

	genCtx.References = append(
		genCtx.References,
		g.tables[table.Name].explicitBelongsTo...,
	)

	if tableInfo.hasUpdateAtField || tableInfo.hasCreatedAtField {
		g.imports[`"time"`] = true
	}

	// Emit the type seperately to prevent double defintions
	var tableType strings.Builder
	err = tableTypeTmpl.Execute(&tableType, genCtx)
	if err != nil {
		return
	}
	var tableSig strings.Builder
	err = tableTypeFieldSigTmpl.Execute(&tableSig, genCtx)
	if err != nil {
		return
	}
	err = g.types.emitType(genCtx.GoName, tableSig.String(), tableType.String())
	if err != nil {
		return
	}

	return tableShimTmpl.Execute(into, genCtx)
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
func (r *{{ .GoName }}) Scan(ctx context.Context, client *PGClient, rs *sql.Rows) error {
	if client.colIdxTabFor{{ .GoName }} == nil {
		err := client.fillColPosTab(
			ctx,
			genTimeColIdxTabFor{{ .GoName }},
			` + "`" + `{{ .PgName }}` + "`" + `,
			&client.colIdxTabFor{{ .GoName }},
		)
		if err != nil {
			return err
		}
	}

	var nullableTgts nullableScanTgtsFor{{ .GoName }}

	scanTgts := make([]interface{}, len(client.colIdxTabFor{{ .GoName }}))
	for genIdx, runIdx := range client.colIdxTabFor{{ .GoName }} {
		scanTgts[runIdx] = scannerTabFor{{ .GoName }}[genIdx](r, &nullableTgts)
	}

	err := rs.Scan(scanTgts...)
	if err != nil {
		return err
	}

	{{- range .Cols }}
	{{- if .Nullable }}
	r.{{ .GoName }} = {{ call .TypeInfo.NullConvertFunc (printf "nullableTgts.scan%s" .GoName) }}
	{{- end }}
	{{- end }}

	return nil
}

type nullableScanTgtsFor{{ .GoName }} struct {
	{{- range .Cols }}
	{{- if .Nullable }}
	scan{{ .GoName }} {{ .TypeInfo.ScanNullName }}
	{{- end }}
	{{- end }}
}

// a table mapping codegen-time col indicies to functions returning a scanner for the
// field that was at that column index at codegen-time.
var scannerTabFor{{ .GoName }} = [...]func(*{{ .GoName }}, *nullableScanTgtsFor{{ .GoName }}) interface{} {
	{{- range .Cols }}
	func (
		r *{{ $.GoName }},
		nullableTgts *nullableScanTgtsFor{{ $.GoName }},
	) interface{} {
		{{- if .Nullable }}
		return {{ call .TypeInfo.SqlReceiver (printf "nullableTgts.scan%s" .GoName) }}
		{{- else }}
		return {{ call .TypeInfo.SqlReceiver (printf "r.%s" .GoName) }}
		{{- end }}
	},
	{{- end }}
}

var genTimeColIdxTabFor{{ .GoName }} map[string]int = map[string]int{
	{{- range $i, $col := .Cols }}
	` + "`" + `{{ $col.PgName }}` + "`" + `: {{ $i }},
	{{- end }}
}
`))

var tableShimTmpl *template.Template = template.Must(template.New("table-shim-tmpl").Parse(`

func (p *PGClient) Get{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
) (*{{ .GoName }}, error) {
	return p.impl.Get{{ .GoName }}(ctx, id)
}
func (tx *TxPGClient) Get{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
) (*{{ .GoName }}, error) {
	return tx.impl.Get{{ .GoName }}(ctx, id)
}
func (p *pgClientImpl) Get{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
) (*{{ .GoName }}, error) {
	values, err := p.List{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id})
	if err != nil {
		return nil, err
	}

	// List{{ .GoName }} always returns the same number of records as were
	// requested, so this is safe.
	return &values[0], err
}

func (p *PGClient) List{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) (ret []{{ .GoName }}, err error) {
	return p.impl.List{{ .GoName }}(ctx, ids)
}
func (tx *TxPGClient) List{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) (ret []{{ .GoName }}, err error) {
	return tx.impl.List{{ .GoName }}(ctx, ids)
}
func (p *pgClientImpl) List{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) (ret []{{ .GoName }}, err error) {
	rows, err := p.db.QueryContext(
		ctx,
		"SELECT * FROM \"{{ .PgName }}\" WHERE \"{{ .PkeyCol.PgName }}\" = ANY($1)",
		pq.Array(ids),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			err = rows.Close()
			if err != nil {
				ret = nil
			}
		} else {
			rowErr := rows.Close()
			if rowErr != nil {
				err = fmt.Errorf("%s AND %s", err.Error(), rowErr.Error())
			}
		}
	}()

	ret = make([]{{ .GoName }}, len(ids))[:0]
	for rows.Next() {
		var value {{ .GoName }}
		err = value.Scan(ctx, p.client, rows)
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
	value *{{ .GoName }},
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	return p.impl.Insert{{ .GoName }}(ctx, value)
}
// Insert a {{ .GoName }} into the database. Returns the primary
// key of the inserted row.
func (tx *TxPGClient) Insert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	return tx.impl.Insert{{ .GoName }}(ctx, value)
}
// Insert a {{ .GoName }} into the database. Returns the primary
// key of the inserted row.
func (p *pgClientImpl) Insert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var ids []{{ .PkeyCol.TypeInfo.Name }}
	ids, err = p.BulkInsert{{ .GoName }}(ctx, []{{ .GoName }}{*value})
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
	return p.impl.BulkInsert{{ .GoName }}(ctx, values)
}
// Insert a list of {{ .GoName }}. Returns a list of the primary keys of
// the inserted rows.
func (tx *TxPGClient) BulkInsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	return tx.impl.BulkInsert{{ .GoName }}(ctx, values)
}
// Insert a list of {{ .GoName }}. Returns a list of the primary keys of
// the inserted rows.
func (p *pgClientImpl) BulkInsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	var fields []string = []string{
		{{- range .Cols }}
		{{- if (not .IsPrimary) }}
		` + "`" + `{{ .PgName }}` + "`" + `,
		{{- end }}
		{{- end }}
	}

	{{- if (or .HasCreatedAtField .HasUpdatedAtField) }}
	var now time.Time
	{{- end }}

	{{- if .HasCreatedAtField }}
	{{- if .CreatedAtHasTimezone }}
	now = time.Now()
	{{- else }}
	now = time.Now().UTC()
	{{- end }}
	for i := range values {
		{{- if .HasCreatedAtField }}
		{{- if .CreatedAtFieldIsNullable }}
		values[i].{{ .CreatedAtField }} = &now
		{{- else }}
		values[i].{{ .CreatedAtField }} = now
		{{- end }}
		{{- end }}
	}
	{{- end }}

	{{- if .HasUpdatedAtField }}
	{{- if .UpdatedAtHasTimezone }}
	now = time.Now()
	{{- else }}
	now = time.Now().UTC()
	{{- end}}
	for i := range values {
		{{- if .HasUpdatedAtField }}
		{{- if .UpdatedAtFieldIsNullable }}
		values[i].{{ .UpdatedAtField }} = &now
		{{- else }}
		values[i].{{ .UpdatedAtField }} = now
		{{- end }}
		{{- end }}
	}
	{{- end }}

	args := make([]interface{}, {{ len .Cols }} * len(values))[:0]
	for _, v := range values {
		{{- range .Cols }}
		{{- if (not .IsPrimary) }}
		args = append(args, {{ call .TypeInfo.SqlArgument (printf "v.%s" .GoName) }})
		{{- end }}
		{{- end }}
	}

	bulkInsertQuery := genBulkInsert(
		"{{ .PgName }}",
		fields,
		len(values),
		"{{ .PkeyCol.PgName }}",
	)

	rows, err := p.db.QueryContext(ctx, bulkInsertQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]{{ .PkeyCol.TypeInfo.Name }}, 0, len(values))
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
	{{ $.GoName }}{{ $c.GoName }}FieldIndex int = {{ $i }}
	{{- end }}
	{{ $.GoName }}MaxFieldIndex int = ({{ len .Cols }} - 1)
)

// A field set saying that all fields in {{ .GoName }} should be updated.
// For use as a 'fieldMask' parameter
var {{ .GoName }}AllFields pggen.FieldSet = pggen.NewFieldSetFilled({{ len .Cols }})

// Update a {{ .GoName }}. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (p *PGClient) Update{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	fieldMask pggen.FieldSet,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	return p.impl.Update{{ .GoName }}(ctx, value, fieldMask)
}
// Update a {{ .GoName }}. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (tx *TxPGClient) Update{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	fieldMask pggen.FieldSet,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	return tx.impl.Update{{ .GoName }}(ctx, value, fieldMask)
}
// Update a {{ .GoName }}. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (p *pgClientImpl) Update{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	fieldMask pggen.FieldSet,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var fields []string = []string{
		{{- range .Cols }}
		` + "`" + `{{ .PgName }}` + "`" + `,
		{{- end }}
	}

	if !fieldMask.Test({{ .GoName }}{{ .PkeyCol.GoName }}FieldIndex) {
		err = fmt.Errorf("primary key required for updates to '{{ .PgName }}'")
		return
	}

	{{- if .HasUpdatedAtField }}
	{{- if .UpdatedAtHasTimezone }}
	now := time.Now()
	{{- else }}
	now := time.Now().UTC()
	{{- end }}
	{{- if .UpdatedAtFieldIsNullable }}
	value.{{ .UpdatedAtField }} = &now
	{{- else }}
	value.{{ .UpdatedAtField }} = now
	{{- end }}
	fieldMask.Set({{ .GoName }}{{ .UpdatedAtField }}FieldIndex, true)
	{{- end }}

	updateStmt := genUpdateStmt(
		"{{ .PgName }}",
		"{{ .PkeyCol.PgName }}",
		fields,
		fieldMask,
		"{{ .PkeyCol.PgName }}",
	)

	args := make([]interface{}, 0, {{ len .Cols }})

	{{- range .Cols }}
	if fieldMask.Test({{ $.GoName }}{{ .GoName }}FieldIndex) {
		args = append(args, {{ call .TypeInfo.SqlArgument (printf "value.%s" .GoName) }})
	}
	{{- end }}

	// add the primary key arg for the WHERE condition
	args = append(args, value.{{ .PkeyCol.GoName }})

	var id {{ .PkeyCol.TypeInfo.Name }}
	err = p.db.QueryRowContext(ctx, updateStmt, args...).
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
	return p.impl.BulkDelete{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id})
}
func (tx *TxPGClient) Delete{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
) error {
	return tx.impl.BulkDelete{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id})
}

func (p *PGClient) BulkDelete{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) error {
	return p.impl.BulkDelete{{ .GoName }}(ctx, ids)
}
func (tx *TxPGClient) BulkDelete{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) error {
	return tx.impl.BulkDelete{{ .GoName }}(ctx, ids)
}
func (p *pgClientImpl) BulkDelete{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) error {
	res, err := p.db.ExecContext(
		ctx,
		"DELETE FROM \"{{ .PgName }}\" WHERE \"{{ .PkeyCol.PgName }}\" = ANY($1)",
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

var {{ .GoName }}AllIncludes *include.Spec = include.Must(include.Parse(
	` + "`" + `{{ .AllIncludeSpec }}` + "`" + `,
))

func (p *PGClient) {{ .GoName }}FillIncludes(
	ctx context.Context,
	rec *{{ .GoName }},
	includes *include.Spec,
) error {
	return p.impl.{{ .GoName }}BulkFillIncludes(ctx, []*{{ .GoName }}{rec}, includes)
}
func (tx *TxPGClient) {{ .GoName }}FillIncludes(
	ctx context.Context,
	rec *{{ .GoName }},
	includes *include.Spec,
) error {
	return tx.impl.{{ .GoName }}BulkFillIncludes(ctx, []*{{ .GoName }}{rec}, includes)
}

func (p *PGClient) {{ .GoName }}BulkFillIncludes(
	ctx context.Context,
	recs []*{{ .GoName }},
	includes *include.Spec,
) (err error) {
	return p.impl.{{ .GoName }}BulkFillIncludes(ctx, recs, includes)
}
func (tx *TxPGClient) {{ .GoName }}BulkFillIncludes(
	ctx context.Context,
	recs []*{{ .GoName }},
	includes *include.Spec,
) (err error) {
	return tx.impl.{{ .GoName }}BulkFillIncludes(ctx, recs, includes)
}
func (p *pgClientImpl) {{ .GoName }}BulkFillIncludes(
	ctx context.Context,
	recs []*{{ .GoName }},
	includes *include.Spec,
) (err error) {
	if includes.TableName != "{{ .PgName }}" {
		return fmt.Errorf(
			"expected includes for '{{ .PgName }}', got '%s'",
			includes.TableName,
		)
	}

	{{- if .References }}
	var subSpec *include.Spec
	var inIncludeSet bool
	{{- end }}

	{{- range .References }}
	// Fill in the {{ .PluralGoPointsFrom }} if it is in includes
	subSpec, inIncludeSet = includes.Includes["{{ .PgPointsFrom }}"]
	if inIncludeSet {
		{{- if .OneToOne }}
		err = p.private{{ $.GoName }}Fill{{ .GoPointsFrom }}(ctx, recs)
		{{- else }}
		err = p.private{{ $.GoName }}Fill{{ .PluralGoPointsFrom }}(ctx, recs)
		{{- end }}
		if err != nil {
			return
		}
		var subRecs []*{{ .GoPointsFrom }}
		for _, outer := range recs {
			{{- if .OneToOne }}
			subRecs = append(subRecs, outer.{{ .GoPointsFrom }})
			{{- else }}
			for i, _ := range outer.{{ .PluralGoPointsFrom }} {
				subRecs = append(subRecs, &outer.{{ .PluralGoPointsFrom }}[i])
			}
			{{- end }}
		}
		err = p.{{ .GoPointsFrom }}BulkFillIncludes(ctx, subRecs, subSpec)
		if err != nil {
			return
		}
	}
	{{- end }}

	return
}
{{- range .References }}

// For a give set of {{ $.GoName }}, fill in all the {{ .GoPointsFrom }}
// connected to them using a single query.
{{- if .OneToOne }}
func (p *pgClientImpl) private{{ $.GoName }}Fill{{ .GoPointsFrom }}(
{{- else }}
func (p *pgClientImpl) private{{ $.GoName }}Fill{{ .PluralGoPointsFrom }}(
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

	rows, err := p.db.QueryContext(
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
		err = childRec.Scan(ctx, p.client, rows)
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
	// The include spec which represents the transitive closure of
	// this tables family
	allIncludeSpec *include.Spec
	// If true, this table does have an update timestamp field
	hasUpdateAtField bool
	// True if the update at field can be null
	updatedAtFieldIsNullable bool
	// True if the updated at field has a time zone
	updatedAtHasTimezone bool
	// If true, this table does have a create timestamp field
	hasCreatedAtField bool
	// True if the created at field can be null
	createdAtFieldIsNullable bool
	// True if the created at field has a time zone
	createdAtHasTimezone bool
	// The table metadata as postgres reports it
	meta tableMeta
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
	g.tables = map[string]*tableGenInfo{}
	g.tableTyNameToTableName = map[string]string{}
	for i, table := range tables {
		info := &tableGenInfo{}
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

	// fill in all the allIncludeSpecs
	for _, info := range g.tables {
		err := ensureSpec(g.tables, info)
		if err != nil {
			return err
		}
	}

	for _, info := range g.tables {
		g.setTimestampFlags(info)
	}

	return nil
}

func (g *Generator) setTimestampFlags(info *tableGenInfo) {
	if len(info.config.CreatedAtField) > 0 {
		for _, cm := range info.meta.Cols {
			if cm.PgName == info.config.CreatedAtField {
				info.hasCreatedAtField = true
				info.createdAtFieldIsNullable = cm.Nullable
				info.createdAtHasTimezone = cm.TypeInfo.IsTimestampWithZone
				break
			}
		}

		if !info.hasCreatedAtField {
			g.warnf(
				"table '%s' has no '%s' created at timestamp\n",
				info.config.Name,
				info.config.CreatedAtField,
			)
		}
	}

	if len(info.config.UpdatedAtField) > 0 {
		for _, cm := range info.meta.Cols {
			if cm.PgName == info.config.UpdatedAtField {
				info.hasUpdateAtField = true
				info.updatedAtFieldIsNullable = cm.Nullable
				info.updatedAtHasTimezone = cm.TypeInfo.IsTimestampWithZone
				break
			}
		}

		if !info.hasUpdateAtField {
			g.warnf(
				"table '%s' has no '%s' updated at timestamp\n",
				info.config.Name,
				info.config.UpdatedAtField,
			)
		}
	}
}

func ensureSpec(tables map[string]*tableGenInfo, info *tableGenInfo) error {
	if info.allIncludeSpec != nil {
		// Some other `ensureSpec` already filled this in for us. Great!
		return nil
	}

	info.allIncludeSpec = &include.Spec{
		TableName: info.meta.PgName,
		Includes:  map[string]*include.Spec{},
	}

	ensureReferencedSpec := func(ref *refMeta) error {
		subInfo := tables[ref.PgPointsFrom]
		if subInfo == nil {
			// This table is referenced in the database schema but not in the
			// config file.
			return nil
		}

		err := ensureSpec(tables, subInfo)
		if err != nil {
			return err
		}
		subSpec := subInfo.allIncludeSpec
		info.allIncludeSpec.Includes[subSpec.TableName] = subSpec

		return nil
	}

	for _, ref := range info.meta.References {
		err := ensureReferencedSpec(&ref)
		if err != nil {
			return err
		}
	}
	for _, ref := range info.explicitBelongsTo {
		err := ensureReferencedSpec(&ref)
		if err != nil {
			return err
		}
	}

	if len(info.allIncludeSpec.Includes) == 0 {
		info.allIncludeSpec.Includes = nil
	}

	return nil
}

func buildExplicitBelongsToMapping(
	tables []tableConfig,
	infoTab map[string]*tableGenInfo,
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
				GoPointsTo: pgToGoName(inflection.Singular(belongsTo.Table)),
				PointsToFields: []fieldNames{
					{
						PgName: pointsToMeta.PkeyCol.PgName,
						GoName: pointsToMeta.PkeyCol.GoName,
					},
				},
				PgPointsFrom:       table.Name,
				GoPointsFrom:       pgToGoName(inflection.Singular(table.Name)),
				PluralGoPointsFrom: pgToGoName(table.Name),
				PointsFromFields: []fieldNames{
					{
						PgName: belongsTo.KeyField,
						GoName: pgToGoName(belongsTo.KeyField),
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
