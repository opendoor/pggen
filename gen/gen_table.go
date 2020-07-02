package gen

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/opendoor-labs/pggen/gen/internal/config"
	"github.com/opendoor-labs/pggen/gen/internal/meta"
	"github.com/opendoor-labs/pggen/gen/internal/names"
)

// Generate code for all of the tables
func (g *Generator) genTables(into io.Writer, tables []config.TableConfig) error {
	if len(tables) > 0 {
		g.log.Infof("	generating %d tables\n", len(tables))
	} else {
		return nil
	}

	g.imports[`"database/sql"`] = true
	g.imports[`"context"`] = true
	g.imports[`"fmt"`] = true
	g.imports[`"strings"`] = true
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
	PkeyCol *meta.ColMeta
	// taken from tableMeta
	PkeyColIdx int
	// taken from tableMeta
	Cols []meta.ColMeta
	// taken from tableMeta
	References []meta.RefMeta
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

func tableGenCtxFromInfo(info *meta.TableMeta) tableGenCtx {
	return tableGenCtx{
		PgName:         info.Info.PgName,
		GoName:         info.Info.GoName,
		PkeyCol:        info.Info.PkeyCol,
		PkeyColIdx:     info.Info.PkeyColIdx,
		Cols:           info.Info.Cols,
		References:     info.Info.References,
		AllIncludeSpec: info.AllIncludeSpec.String(),

		HasCreatedAtField:        info.HasCreatedAtField,
		CreatedAtField:           names.PgToGoName(info.Config.CreatedAtField),
		CreatedAtFieldIsNullable: info.CreatedAtFieldIsNullable,
		CreatedAtHasTimezone:     info.CreatedAtHasTimezone,

		HasUpdatedAtField:        info.HasUpdateAtField,
		UpdatedAtField:           names.PgToGoName(info.Config.UpdatedAtField),
		UpdatedAtFieldIsNullable: info.UpdatedAtFieldIsNullable,
		UpdatedAtHasTimezone:     info.UpdatedAtHasTimezone,
	}
}

func (g *Generator) genTable(
	into io.Writer,
	table *config.TableConfig,
) (err error) {
	g.log.Infof("		generating table '%s'\n", table.Name)
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"while generating table '%s': %s", table.Name, err.Error())
		}
	}()

	tableInfo, ok := g.metaResolver.TableMeta(table.Name)
	if !ok {
		return fmt.Errorf("could get schema info about table '%s'", table.Name)
	}

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
		fromTable, inMap := g.metaResolver.TableMeta(ref.PointsFrom.PgName)
		if inMap {
			if !fromTable.Config.NoInferBelongsTo {
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

	genCtx.References = tableInfo.AllReferences

	if tableInfo.HasUpdateAtField || tableInfo.HasCreatedAtField {
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
	err = g.typeResolver.EmitType(genCtx.GoName, tableSig.String(), tableType.String())
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
	{{- if .IsPrimary }} gorm:"is_primary" {{- end }} {{ .ExtraTags -}}` +
	"`" + `
	{{- end }}
	{{- range .References }}
	{{- if .OneToOne }}
	{{ .GoPointsFromFieldName }} *{{ .PointsFrom.GoName }}
	{{- else }}
	{{ .GoPointsFromFieldName }} []*{{ .PointsFrom.GoName }}
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
	for runIdx, genIdx := range client.colIdxTabFor{{ .GoName }} {
		if genIdx == -1 {
			scanTgts[runIdx] = &pggenSinkScanner{}
		} else {
			scanTgts[runIdx] = scannerTabFor{{ .GoName }}[genIdx](r, &nullableTgts)
		}
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
		return {{ call .TypeInfo.NullSqlReceiver (printf "nullableTgts.scan%s" .GoName) }}
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
	if len(ids) == 0 {
		return []{{ .GoName }}{}, nil
	}

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

	ret = make([]{{ .GoName }}, 0, len(ids))
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
	opts ...pggen.InsertOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	return p.impl.Insert{{ .GoName }}(ctx, value, opts...)
}
// Insert a {{ .GoName }} into the database. Returns the primary
// key of the inserted row.
func (tx *TxPGClient) Insert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	opts ...pggen.InsertOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	return tx.impl.Insert{{ .GoName }}(ctx, value, opts...)
}
// Insert a {{ .GoName }} into the database. Returns the primary
// key of the inserted row.
func (p *pgClientImpl) Insert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	opts ...pggen.InsertOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var ids []{{ .PkeyCol.TypeInfo.Name }}
	ids, err = p.BulkInsert{{ .GoName }}(ctx, []{{ .GoName }}{*value}, opts...)
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
	opts ...pggen.InsertOpt,
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	return p.impl.BulkInsert{{ .GoName }}(ctx, values, opts...)
}
// Insert a list of {{ .GoName }}. Returns a list of the primary keys of
// the inserted rows.
func (tx *TxPGClient) BulkInsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
	opts ...pggen.InsertOpt,
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	return tx.impl.BulkInsert{{ .GoName }}(ctx, values, opts...)
}
// Insert a list of {{ .GoName }}. Returns a list of the primary keys of
// the inserted rows.
func (p *pgClientImpl) BulkInsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
	opts ...pggen.InsertOpt,
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	if len(values) == 0 {
		return []{{ .PkeyCol.TypeInfo.Name }}{}, nil
	}

	opt := pggen.InsertOptions{}
	for _, o := range opts {
		o(&opt)
	}

	{{- if (or .HasCreatedAtField .HasUpdatedAtField) }}
	now := time.Now()
	{{- end }}

	{{- if .HasCreatedAtField }}
	for i := range values {
		{{- if .CreatedAtHasTimezone }}
		createdAt := now
		{{- else }}
		createdAt := now.UTC()
		{{- end }}

		{{- if .HasCreatedAtField }}
		{{- if .CreatedAtFieldIsNullable }}
		values[i].{{ .CreatedAtField }} = &createdAt
		{{- else }}
		values[i].{{ .CreatedAtField }} = createdAt
		{{- end }}
		{{- end }}
	}
	{{- end }}

	{{- if .HasUpdatedAtField }}
	for i := range values {
		{{- if .CreatedAtHasTimezone }}
		updatedAt := now
		{{- else }}
		updatedAt := now.UTC()
		{{- end }}

		{{- if .HasUpdatedAtField }}
		{{- if .UpdatedAtFieldIsNullable }}
		values[i].{{ .UpdatedAtField }} = &updatedAt
		{{- else }}
		values[i].{{ .UpdatedAtField }} = updatedAt
		{{- end }}
		{{- end }}
	}
	{{- end }}

	args := make([]interface{}, 0, {{ len .Cols }} * len(values))
	for _, v := range values {
		{{- range .Cols }}
		{{- if (not .IsPrimary) }}
		{{- if .Nullable }}
		args = append(args, {{ call .TypeInfo.NullSqlArgument (printf "v.%s" .GoName) }})
		{{- else }}
		args = append(args, {{ call .TypeInfo.SqlArgument (printf "v.%s" .GoName) }})
		{{- end }}
		{{- else }}
		if opt.UsePkey {
			{{- if .Nullable }}
			args = append(args, {{ call .TypeInfo.NullSqlArgument (printf "v.%s" .GoName) }})
			{{- else }}
			args = append(args, {{ call .TypeInfo.SqlArgument (printf "v.%s" .GoName) }})
			{{- end }}
		}
		{{- end }}
		{{- end }}
	}

	bulkInsertQuery := genBulkInsertStmt(
		"{{ .PgName }}",
		fieldsFor{{ .GoName }},
		len(values),
		"{{ .PkeyCol.PgName }}",
		opt.UsePkey,
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

var fieldsFor{{ .GoName }} []string = []string{
	{{- range .Cols }}
	` + "`" + `{{ .PgName }}` + "`" + `,
	{{- end }}
}

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
func (p *pgClientImpl) Update{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	fieldMask pggen.FieldSet,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
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
		fieldsFor{{ .GoName }},
		fieldMask,
		"{{ .PkeyCol.PgName }}",
	)

	args := make([]interface{}, 0, {{ len .Cols }})

	{{- range .Cols }}
	if fieldMask.Test({{ $.GoName }}{{ .GoName }}FieldIndex) {
		{{- if .Nullable }}
		args = append(args, {{ call .TypeInfo.NullSqlArgument (printf "value.%s" .GoName) }})
		{{- else }}
		args = append(args, {{ call .TypeInfo.SqlArgument (printf "value.%s" .GoName) }})
		{{- end }}
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

// Updsert a {{ .GoName }} value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (p *PGClient) Upsert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var val []{{ .PkeyCol.TypeInfo.Name }}
	val, err = p.impl.BulkUpsert{{ .GoName }}(ctx, []{{ .GoName }}{*value}, constraintNames, fieldMask)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.{{ .PkeyCol.GoName }}, nil
}
// Updsert a {{ .GoName }} value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (tx *TxPGClient) Upsert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var val []{{ .PkeyCol.TypeInfo.Name }}
	val, err = tx.impl.BulkUpsert{{ .GoName }}(ctx, []{{ .GoName }}{*value}, constraintNames, fieldMask)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.{{ .PkeyCol.GoName }}, nil
}


// Updsert a set of {{ .GoName }} values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (p *PGClient) BulkUpsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
) (ret []{{ .PkeyCol.TypeInfo.Name }}, err error) {
	return p.impl.BulkUpsert{{ .GoName }}(ctx, values, constraintNames, fieldMask)
}
// Updsert a set of {{ .GoName }} values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (tx *TxPGClient) BulkUpsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
) (ret []{{ .PkeyCol.TypeInfo.Name }}, err error) {
	return tx.impl.BulkUpsert{{ .GoName }}(ctx, values, constraintNames, fieldMask)
}
func (p *pgClientImpl) BulkUpsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	if len(values) == 0 {
		return []{{ .PkeyCol.TypeInfo.Name }}{}, nil
	}

	if constraintNames == nil || len(constraintNames) == 0 {
		constraintNames = []string{` + "`" + `{{ .PkeyCol.PgName }}` + "`" + `}
	}

	{{ if (or .HasCreatedAtField .HasUpdatedAtField) }}
	now := time.Now()

	{{- if .HasCreatedAtField }}
	{{- if .CreatedAtHasTimezone }}
	createdAt := now
	{{- else }}
	createdAt := now.UTC()
	{{- end }}
	for i := range values {
		{{- if .CreatedAtFieldIsNullable }}
		values[i].{{ .CreatedAtField }} = &createdAt
		{{- else }}
		values[i].{{ .CreatedAtField }} = createdAt
		{{- end }}
	}
	{{- end}}

	{{- if .HasUpdatedAtField }}
	{{- if .UpdatedAtHasTimezone }}
	updatedAt := now
	{{- else }}
	updatedAt := now.UTC()
	{{- end }}
	for i := range values {
		{{- if .UpdatedAtFieldIsNullable }}
		values[i].{{ .UpdatedAtField }} = &updatedAt
		{{- else }}
		values[i].{{ .UpdatedAtField }} = updatedAt
		{{- end }}
	}
	fieldMask.Set({{ .GoName }}{{ .UpdatedAtField }}FieldIndex, true)
	{{- end }}
	{{- end }}

	var stmt strings.Builder
	genInsertCommon(
		&stmt,
		` + "`" + `{{ .PgName }}` + "`" + `,
		fieldsFor{{ .GoName }},
		len(values),
		` + "`" + `{{ .PkeyCol.PgName }}` + "`" + `,
		fieldMask.Test({{ .GoName }}{{ .PkeyCol.GoName }}FieldIndex),
	)

	if fieldMask.CountSetBits() > 0 {
		stmt.WriteString("ON CONFLICT (")
		stmt.WriteString(strings.Join(constraintNames, ","))
		stmt.WriteString(") DO UPDATE SET ")

		updateCols := make([]string, 0, {{ len .Cols }})
		updateExprs := make([]string, 0, {{ len .Cols }})
		{{- range .Cols }}
		if fieldMask.Test({{ $.GoName }}{{ .GoName }}FieldIndex) {
			updateCols = append(updateCols, ` + "`" + `{{ .PgName }}` + "`" + `)
			updateExprs = append(updateExprs, ` + "`" + `excluded.{{ .PgName }}` + "`" + `)
		}
		{{- end }}
		if len(updateCols) > 1 {
			stmt.WriteRune('(')
		}
		stmt.WriteString(strings.Join(updateCols, ","))
		if len(updateCols) > 1 {
			stmt.WriteRune(')')
		}
		stmt.WriteString(" = ")
		if len(updateCols) > 1 {
			stmt.WriteRune('(')
		}
		stmt.WriteString(strings.Join(updateExprs, ","))
		if len(updateCols) > 1 {
			stmt.WriteRune(')')
		}
	} else {
		stmt.WriteString("ON CONFLICT DO NOTHING")
	}

	stmt.WriteString(` + "`" + ` RETURNING "{{ .PkeyCol.PgName }}"` + "`" + `)

	args := make([]interface{}, 0, {{ len .Cols }} * len(values))
	for _, v := range values {
		{{- range $i, $col := .Cols }}
		{{- if (eq $i $.PkeyColIdx) }}
		if fieldMask.Test({{ $.GoName }}{{ $col.GoName }}FieldIndex) {
			{{- if .Nullable }}
			args = append(args, {{ call .TypeInfo.NullSqlArgument (printf "v.%s" .GoName) }})
			{{- else }}
			args = append(args, {{ call .TypeInfo.SqlArgument (printf "v.%s" .GoName) }})
			{{- end }}
		}
		{{- else }}
		{{- if .Nullable }}
		args = append(args, {{ call .TypeInfo.NullSqlArgument (printf "v.%s" .GoName) }})
		{{- else }}
		args = append(args, {{ call .TypeInfo.SqlArgument (printf "v.%s" .GoName) }})
		{{- end }}
		{{- end }}
		{{- end }}
	}

	rows, err := p.db.QueryContext(ctx, stmt.String(), args...)
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
	if len(ids) == 0 {
		return nil
	}

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
) error {
	return p.impl.{{ .GoName }}BulkFillIncludes(ctx, recs, includes)
}
func (tx *TxPGClient) {{ .GoName }}BulkFillIncludes(
	ctx context.Context,
	recs []*{{ .GoName }},
	includes *include.Spec,
) error {
	return tx.impl.{{ .GoName }}BulkFillIncludes(ctx, recs, includes)
}
func (p *pgClientImpl) {{ .GoName }}BulkFillIncludes(
	ctx context.Context,
	recs []*{{ .GoName }},
	includes *include.Spec,
) error {
	loadedRecordTab := map[string]interface{}{}

	return p.impl{{ .GoName }}BulkFillIncludes(ctx, recs, includes, loadedRecordTab)
}

func (p *pgClientImpl) impl{{ .GoName }}BulkFillIncludes(
	ctx context.Context,
	recs []*{{ .GoName }},
	includes *include.Spec,
	loadedRecordTab map[string]interface{},
) (err error) {
	if includes.TableName != ` + "`" + `{{ .PgName }}` + "`" + ` {
		return fmt.Errorf(
			"expected includes for '{{ .PgName }}', got '%s'",
			includes.TableName,
		)
	}

	loadedTab, inMap := loadedRecordTab[` + "`" + `{{ .PgName }}` + "`" + `]
	if inMap {
		idToRecord := loadedTab.(map[{{ .PkeyCol.TypeInfo.Name }}]*{{ .GoName }})
		for _, r := range recs {
			_, alreadyLoaded := idToRecord[r.{{ .PkeyCol.GoName }}]
			if !alreadyLoaded {
				idToRecord[r.{{ .PkeyCol.GoName }}] = r
			}
		}
	} else {
		idToRecord := make(map[{{ .PkeyCol.TypeInfo.Name }}]*{{ .GoName }}, len(recs))
		for _, r := range recs {
			idToRecord[r.{{ .PkeyCol.GoName }}] = r
		}
		loadedRecordTab[` + "`" + `{{ .PgName }}` + "`" + `] = idToRecord
	}

	{{- if .References }}
	var subSpec *include.Spec
	var inIncludeSet bool
	{{- end }}

	{{- range .References }}
	// Fill in the {{ .PointsFrom.PluralGoName }} if it is in includes
	subSpec, inIncludeSet = includes.Includes[` + "`" + `{{ .PgPointsFromFieldName }}` + "`" + `]
	if inIncludeSet {
		err = p.private{{ $.GoName }}Fill{{ .GoPointsFromFieldName }}(ctx, loadedRecordTab)
		if err != nil {
			return
		}

		subRecs := make([]*{{ .PointsFrom.GoName }}, 0, len(recs))
		for _, outer := range recs {
			{{- if .OneToOne }}
			if outer.{{ .GoPointsFromFieldName }} != nil {
				subRecs = append(subRecs, outer.{{ .GoPointsFromFieldName }})
			}
			{{- else }}
			for i := range outer.{{ .GoPointsFromFieldName }} {
				{{- if .Nullable }}
				if outer.{{ .GoPointsFromFieldName }}[i] == nil {
					continue
				}
				{{- end }}
				subRecs = append(subRecs, outer.{{ .GoPointsFromFieldName }}[i])
			}
			{{- end }}
		}

		err = p.impl{{ .PointsFrom.GoName }}BulkFillIncludes(ctx, subRecs, subSpec, loadedRecordTab)
		if err != nil {
			return
		}
	}
	{{- end }}

	return
}

{{- range .References }}

// For a give set of {{ $.GoName }}, fill in all the {{ .PointsFrom.GoName }}
// connected to them using a single query.
func (p *pgClientImpl) private{{ $.GoName }}Fill{{ .GoPointsFromFieldName }}(
	ctx context.Context,
	loadedRecordTab map[string]interface{},
) error {
	parentLoadedTab, inMap := loadedRecordTab[` + "`" + `{{ .PointsTo.PgName }}` + "`" + `]
	if !inMap {
		return fmt.Errorf("internal pggen error: table not pre-loaded")
	}
	parentIDToRecord := parentLoadedTab.(map[{{ (index .PointsToFields 0).TypeInfo.Name }}]*{{ .PointsTo.GoName }})
	ids := make([]{{ (index .PointsToFields 0).TypeInfo.Name }}, 0, len(parentIDToRecord))
	for _, rec := range parentIDToRecord{
		ids = append(ids, rec.{{ (index .PointsToFields 0).GoName }})
	}

	var childIDToRecord map[{{ .PointsFrom.PkeyCol.TypeInfo.Name }}]*{{ .PointsFrom.GoName }}
	childLoadedTab, inMap := loadedRecordTab[` + "`" + `{{ .PointsFrom.PgName }}` + "`" + `]
	if inMap {
		childIDToRecord = childLoadedTab.(map[{{ .PointsFrom.PkeyCol.TypeInfo.Name }}]*{{ .PointsFrom.GoName }})
	} else {
		childIDToRecord = map[{{ .PointsFrom.PkeyCol.TypeInfo.Name }}]*{{ .PointsFrom.GoName }}{}
	}

	rows, err := p.db.QueryContext(
		ctx,
		` + "`" +
	`SELECT * FROM "{{ .PointsFrom.PgName }}"
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
		var scannedChildRec {{ .PointsFrom.GoName }}
		err = scannedChildRec.Scan(ctx, p.client, rows)
		if err != nil {
			return err
		}

		var childRec *{{ .PointsFrom.GoName }}

		preloadedChildRec, alreadyLoaded := childIDToRecord[scannedChildRec.{{ .PointsFrom.PkeyCol.GoName }}]
		if alreadyLoaded {
			childRec = preloadedChildRec
		} else {
			childRec = &scannedChildRec
		}

		{{- if .Nullable }}
		// we know that the foreign key can't be null because of the SQL query
		parentRec := parentIDToRecord[*childRec.{{ (index .PointsFromFields 0).GoName }}]
		{{- else }}
		parentRec := parentIDToRecord[childRec.{{ (index .PointsFromFields 0).GoName }}]
		{{- end }}

		{{- if .OneToOne }}
		parentRec.{{ .GoPointsFromFieldName }} = childRec
		break
		{{- else }}
		parentRec.{{ .GoPointsFromFieldName }} = append(parentRec.{{ .GoPointsFromFieldName }}, childRec)
		{{- end }}
	}

	return nil
}

{{ end }}
`))
