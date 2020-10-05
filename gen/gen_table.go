package gen

import (
	"fmt"
	"io"
	"text/template"

	"github.com/opendoor-labs/pggen/gen/internal/config"
	"github.com/opendoor-labs/pggen/gen/internal/meta"
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
	g.imports[`"sync"`] = true
	g.imports[`"github.com/lib/pq"`] = true
	g.imports[`"github.com/opendoor-labs/pggen/include"`] = true
	g.imports[`"github.com/opendoor-labs/pggen/unstable"`] = true
	g.imports[`"github.com/opendoor-labs/pggen"`] = true

	for i := range tables {
		err := g.genTable(into, &tables[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func tableGenCtxFromInfo(info *meta.TableMeta) meta.TableGenCtx {
	return meta.TableGenCtx{
		PgName:         info.Info.PgName,
		GoName:         info.Info.GoName,
		PkeyCol:        info.Info.PkeyCol,
		PkeyColIdx:     info.Info.PkeyColIdx,
		AllIncludeSpec: info.AllIncludeSpec.String(),
		Meta:           info,
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

	if tableInfo.HasUpdatedAtField || tableInfo.HasCreatedAtField {
		g.imports[`"time"`] = true
	}

	err = g.typeResolver.EmitStructType(genCtx.GoName, genCtx)
	if err != nil {
		return
	}

	return tableShimTmpl.Execute(into, genCtx)
}

var tableShimTmpl *template.Template = template.Must(template.New("table-shim-tmpl").Parse(`

func (p *PGClient) Get{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
	opts ...pggen.GetOpt,
) (*{{ .GoName }}, error) {
	return p.impl.get{{ .GoName }}(ctx, id)
}
func (tx *TxPGClient) Get{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
	opts ...pggen.GetOpt,
) (*{{ .GoName }}, error) {
	return tx.impl.get{{ .GoName }}(ctx, id)
}
func (p *pgClientImpl) get{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
	opts ...pggen.GetOpt,
) (*{{ .GoName }}, error) {
	values, err := p.list{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id}, true /* isGet */)
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
	opts ...pggen.ListOpt,
) (ret []{{ .GoName }}, err error) {
	return p.impl.list{{ .GoName }}(ctx, ids, false /* isGet */)
}
func (tx *TxPGClient) List{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
	opts ...pggen.ListOpt,
) (ret []{{ .GoName }}, err error) {
	return tx.impl.list{{ .GoName }}(ctx, ids, false /* isGet */)
}
func (p *pgClientImpl) list{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
	isGet bool,
	opts ...pggen.ListOpt,
) (ret []{{ .GoName }}, err error) {
	if len(ids) == 0 {
		return []{{ .GoName }}{}, nil
	}

	rows, err := p.db.QueryContext(
		ctx,
		"SELECT * FROM \"{{ .PgName }}\" WHERE \"{{ .PkeyCol.PgName }}\" = ANY($1)
		{{- if .Meta.HasDeletedAtField }} AND \"{{ .Meta.PgDeletedAtField }}\" IS NULL {{ end }}",
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
		if isGet {
			return nil, &unstable.NotFoundError{
				Msg: "Get{{ .GoName }}: record not found",
			}
		} else {
			return nil, &unstable.NotFoundError{
				Msg: fmt.Sprintf(
					"List{{ .GoName }}: asked for %d records, found %d",
					len(ids),
					len(ret),
				),
			}
		}
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
	return p.impl.insert{{ .GoName }}(ctx, value, opts...)
}
// Insert a {{ .GoName }} into the database. Returns the primary
// key of the inserted row.
func (tx *TxPGClient) Insert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	opts ...pggen.InsertOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	return tx.impl.insert{{ .GoName }}(ctx, value, opts...)
}
// Insert a {{ .GoName }} into the database. Returns the primary
// key of the inserted row.
func (p *pgClientImpl) insert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	opts ...pggen.InsertOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var ids []{{ .PkeyCol.TypeInfo.Name }}
	ids, err = p.bulkInsert{{ .GoName }}(ctx, []{{ .GoName }}{*value}, opts...)
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
	return p.impl.bulkInsert{{ .GoName }}(ctx, values, opts...)
}
// Insert a list of {{ .GoName }}. Returns a list of the primary keys of
// the inserted rows.
func (tx *TxPGClient) BulkInsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
	opts ...pggen.InsertOpt,
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	return tx.impl.bulkInsert{{ .GoName }}(ctx, values, opts...)
}
// Insert a list of {{ .GoName }}. Returns a list of the primary keys of
// the inserted rows.
func (p *pgClientImpl) bulkInsert{{ .GoName }}(
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

	{{- if (or .Meta.HasCreatedAtField .Meta.HasUpdatedAtField) }}
	now := time.Now()
	{{- end }}

	{{- if .Meta.HasCreatedAtField }}
	for i := range values {
		{{- if .Meta.CreatedAtHasTimezone }}
		createdAt := now
		{{- else }}
		createdAt := now.UTC()
		{{- end }}

		{{- if .Meta.HasCreatedAtField }}
		{{- if .Meta.CreatedAtFieldIsNullable }}
		values[i].{{ .Meta.GoCreatedAtField }} = &createdAt
		{{- else }}
		values[i].{{ .Meta.GoCreatedAtField }} = createdAt
		{{- end }}
		{{- end }}
	}
	{{- end }}

	{{- if .Meta.HasUpdatedAtField }}
	for i := range values {
		{{- if .Meta.CreatedAtHasTimezone }}
		updatedAt := now
		{{- else }}
		updatedAt := now.UTC()
		{{- end }}

		{{- if .Meta.HasUpdatedAtField }}
		{{- if .Meta.UpdatedAtFieldIsNullable }}
		values[i].{{ .Meta.GoUpdatedAtField }} = &updatedAt
		{{- else }}
		values[i].{{ .Meta.GoUpdatedAtField }} = updatedAt
		{{- end }}
		{{- end }}
	}
	{{- end }}

	args := make([]interface{}, 0, {{ len .Meta.Info.Cols }} * len(values))
	for _, v := range values {
		{{- range .Meta.Info.Cols }}
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
	{{- range $i, $c := .Meta.Info.Cols }}
	{{ $.GoName }}{{ $c.GoName }}FieldIndex int = {{ $i }}
	{{- end }}
	{{ $.GoName }}MaxFieldIndex int = ({{ len .Meta.Info.Cols }} - 1)
)

// A field set saying that all fields in {{ .GoName }} should be updated.
// For use as a 'fieldMask' parameter
var {{ .GoName }}AllFields pggen.FieldSet = pggen.NewFieldSetFilled({{ len .Meta.Info.Cols }})

var fieldsFor{{ .GoName }} []string = []string{
	{{- range .Meta.Info.Cols }}
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
	opts ...pggen.UpdateOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	return p.impl.update{{ .GoName }}(ctx, value, fieldMask)
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
	opts ...pggen.UpdateOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	return tx.impl.update{{ .GoName }}(ctx, value, fieldMask)
}
func (p *pgClientImpl) update{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	if !fieldMask.Test({{ .GoName }}{{ .PkeyCol.GoName }}FieldIndex) {
		err = fmt.Errorf("primary key required for updates to '{{ .PgName }}'")
		return
	}

	{{- if .Meta.HasUpdatedAtField }}
	{{- if .Meta.UpdatedAtHasTimezone }}
	now := time.Now()
	{{- else }}
	now := time.Now().UTC()
	{{- end }}
	{{- if .Meta.UpdatedAtFieldIsNullable }}
	value.{{ .Meta.GoUpdatedAtField }} = &now
	{{- else }}
	value.{{ .Meta.GoUpdatedAtField }} = now
	{{- end }}
	fieldMask.Set({{ .GoName }}{{ .Meta.GoUpdatedAtField }}FieldIndex, true)
	{{- end }}

	updateStmt := genUpdateStmt(
		"{{ .PgName }}",
		"{{ .PkeyCol.PgName }}",
		fieldsFor{{ .GoName }},
		fieldMask,
		"{{ .PkeyCol.PgName }}",
	)

	args := make([]interface{}, 0, {{ len .Meta.Info.Cols }})

	{{- range .Meta.Info.Cols }}
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

// Upsert a {{ .GoName }} value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (p *PGClient) Upsert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var val []{{ .PkeyCol.TypeInfo.Name }}
	val, err = p.impl.bulkUpsert{{ .GoName }}(ctx, []{{ .GoName }}{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.{{ .PkeyCol.GoName }}, nil
}
// Upsert a {{ .GoName }} value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (tx *TxPGClient) Upsert{{ .GoName }}(
	ctx context.Context,
	value *{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var val []{{ .PkeyCol.TypeInfo.Name }}
	val, err = tx.impl.bulkUpsert{{ .GoName }}(ctx, []{{ .GoName }}{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.{{ .PkeyCol.GoName }}, nil
}


// Upsert a set of {{ .GoName }} values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (p *PGClient) BulkUpsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []{{ .PkeyCol.TypeInfo.Name }}, err error) {
	return p.impl.bulkUpsert{{ .GoName }}(ctx, values, constraintNames, fieldMask, opts...)
}
// Upsert a set of {{ .GoName }} values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (tx *TxPGClient) BulkUpsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []{{ .PkeyCol.TypeInfo.Name }}, err error) {
	return tx.impl.bulkUpsert{{ .GoName }}(ctx, values, constraintNames, fieldMask, opts...)
}
func (p *pgClientImpl) bulkUpsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	if len(values) == 0 {
		return []{{ .PkeyCol.TypeInfo.Name }}{}, nil
	}

	options := pggen.UpsertOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if constraintNames == nil || len(constraintNames) == 0 {
		constraintNames = []string{` + "`" + `{{ .PkeyCol.PgName }}` + "`" + `}
	}

	{{ if (or .Meta.HasCreatedAtField .Meta.HasUpdatedAtField) }}
	now := time.Now()

	{{- if .Meta.HasCreatedAtField }}
	{{- if .Meta.CreatedAtHasTimezone }}
	createdAt := now
	{{- else }}
	createdAt := now.UTC()
	{{- end }}
	for i := range values {
		{{- if .Meta.CreatedAtFieldIsNullable }}
		values[i].{{ .Meta.GoCreatedAtField }} = &createdAt
		{{- else }}
		values[i].{{ .Meta.GoCreatedAtField }} = createdAt
		{{- end }}
	}
	{{- end}}

	{{- if .Meta.HasUpdatedAtField }}
	{{- if .Meta.UpdatedAtHasTimezone }}
	updatedAt := now
	{{- else }}
	updatedAt := now.UTC()
	{{- end }}
	for i := range values {
		{{- if .Meta.UpdatedAtFieldIsNullable }}
		values[i].{{ .Meta.GoUpdatedAtField }} = &updatedAt
		{{- else }}
		values[i].{{ .Meta.GoUpdatedAtField }} = updatedAt
		{{- end }}
	}
	fieldMask.Set({{ .GoName }}{{ .Meta.GoUpdatedAtField }}FieldIndex, true)
	{{- end }}
	{{- end }}

	var stmt strings.Builder
	genInsertCommon(
		&stmt,
		` + "`" + `{{ .PgName }}` + "`" + `,
		fieldsFor{{ .GoName }},
		len(values),
		` + "`" + `{{ .PkeyCol.PgName }}` + "`" + `,
		options.UsePkey,
	)

	setBits := fieldMask.CountSetBits()
	hasConflictAction := setBits > 1 ||
		(setBits == 1 && fieldMask.Test({{ .GoName }}{{ .PkeyCol.GoName }}FieldIndex) && options.UsePkey) ||
		(setBits == 1 && !fieldMask.Test({{ .GoName }}{{ .PkeyCol.GoName }}FieldIndex))

	if hasConflictAction {
		stmt.WriteString("ON CONFLICT (")
		stmt.WriteString(strings.Join(constraintNames, ","))
		stmt.WriteString(") DO UPDATE SET ")

		updateCols := make([]string, 0, {{ len .Meta.Info.Cols }})
		updateExprs := make([]string, 0, {{ len .Meta.Info.Cols }})
		if options.UsePkey {
			updateCols = append(updateCols, ` + "`" + `{{ .PkeyCol.PgName }}` + "`" + `)
			updateExprs = append(updateExprs, ` + "`" + `excluded.{{ .PkeyCol.PgName }}` + "`" + `)
		}
		{{- range $i, $col := .Meta.Info.Cols }}
		{{- if (not (eq $i $.PkeyColIdx)) }}
		if fieldMask.Test({{ $.GoName }}{{ $col.GoName }}FieldIndex) {
			updateCols = append(updateCols, ` + "`" + `{{ $col.PgName }}` + "`" + `)
			updateExprs = append(updateExprs, ` + "`" + `excluded.{{ $col.PgName }}` + "`" + `)
		}
		{{- end }}
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

	args := make([]interface{}, 0, {{ len .Meta.Info.Cols }} * len(values))
	for _, v := range values {
		{{- range $i, $col := .Meta.Info.Cols }}
		{{- if (eq $i $.PkeyColIdx) }}
		if options.UsePkey {
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
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDelete{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id}, opts...)
}
func (tx *TxPGClient) Delete{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDelete{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id}, opts...)
}

func (p *PGClient) BulkDelete{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDelete{{ .GoName }}(ctx, ids, opts...)
}
func (tx *TxPGClient) BulkDelete{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDelete{{ .GoName }}(ctx, ids, opts...)
}
func (p *pgClientImpl) bulkDelete{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
	opts ...pggen.DeleteOpt,
) error {
	if len(ids) == 0 {
		return nil
	}

	options := pggen.DeleteOptions{}
	for _, o := range opts {
		o(&options)
	}

	{{- if .Meta.HasDeletedAtField }}
	{{- if .Meta.DeletedAtHasTimezone }}
	now := time.Now()
	{{- else }}
	now := time.Now().UTC()
	{{- end }}
	var (
		res sql.Result
		err error
	)
	if options.DoHardDelete {
		res, err = p.db.ExecContext(
			ctx,
			"DELETE FROM \"{{ .PgName }}\" WHERE \"{{ .PkeyCol.PgName }}\" = ANY($1)",
			pq.Array(ids),
		)
	} else {
		res, err = p.db.ExecContext(
			ctx,
			"UPDATE \"{{ .PgName }}\" SET \"{{ .Meta.PgDeletedAtField }}\" = $1 WHERE \"{{ .PkeyCol.PgName }}\" = ANY($2)",
			now,
			pq.Array(ids),
		)
	}
	{{- else }}
	res, err := p.db.ExecContext(
		ctx,
		"DELETE FROM \"{{ .PgName }}\" WHERE \"{{ .PkeyCol.PgName }}\" = ANY($1)",
		pq.Array(ids),
	)
	{{- end }}
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
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.private{{ .GoName }}BulkFillIncludes(ctx, []*{{ .GoName }}{rec}, includes)
}
func (tx *TxPGClient) {{ .GoName }}FillIncludes(
	ctx context.Context,
	rec *{{ .GoName }},
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.private{{ .GoName }}BulkFillIncludes(ctx, []*{{ .GoName }}{rec}, includes)
}

func (p *PGClient) {{ .GoName }}BulkFillIncludes(
	ctx context.Context,
	recs []*{{ .GoName }},
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.private{{ .GoName }}BulkFillIncludes(ctx, recs, includes)
}
func (tx *TxPGClient) {{ .GoName }}BulkFillIncludes(
	ctx context.Context,
	recs []*{{ .GoName }},
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.private{{ .GoName }}BulkFillIncludes(ctx, recs, includes)
}
func (p *pgClientImpl) private{{ .GoName }}BulkFillIncludes(
	ctx context.Context,
	recs []*{{ .GoName }},
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
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

	{{- if (or .Meta.AllIncomingReferences .Meta.AllOutgoingReferences) }}
	var subSpec *include.Spec
	var inIncludeSet bool
	{{- end }}

	{{- range .Meta.AllIncomingReferences }}
	// Fill in the {{ .PointsFrom.Info.PluralGoName }} if it is in includes
	subSpec, inIncludeSet = includes.Includes[` + "`" + `{{ .PgPointsFromFieldName }}` + "`" + `]
	if inIncludeSet {
		err = p.private{{ $.GoName }}Fill{{ .GoPointsFromFieldName }}(ctx, loadedRecordTab)
		if err != nil {
			return
		}

		subRecs := make([]*{{ .PointsFrom.Info.GoName }}, 0, len(recs))
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

		err = p.impl{{ .PointsFrom.Info.GoName }}BulkFillIncludes(ctx, subRecs, subSpec, loadedRecordTab)
		if err != nil {
			return
		}
	}
	{{- end }}

	{{- range .Meta.AllOutgoingReferences }}
	subSpec, inIncludeSet = includes.Includes[` + "`" + `{{ .PgPointsToFieldName }}` + "`" + `]
	if inIncludeSet {
		err = p.private{{ $.GoName }}FillParent{{ .GoPointsToFieldName }}(ctx, loadedRecordTab)
		if err != nil {
			return
		}

		subRecs := make([]*{{ .PointsTo.Info.GoName }}, 0, len(recs))
		for _, outer := range recs {
			if outer.{{ .GoPointsToFieldName }} != nil {
				subRecs = append(subRecs, outer.{{ .GoPointsToFieldName }})
			}
		}

		err = p.impl{{ .PointsTo.Info.GoName }}BulkFillIncludes(ctx, subRecs, subSpec, loadedRecordTab)
		if err != nil {
			return
		}
	}
	{{- end }}

	return
}

{{- range .Meta.AllIncomingReferences }}

// For a given set of {{ $.GoName }}, fill in all the {{ .PointsFrom.Info.GoName }}
// connected to them using a single query.
func (p *pgClientImpl) private{{ $.GoName }}Fill{{ .GoPointsFromFieldName }}(
	ctx context.Context,
	loadedRecordTab map[string]interface{},
) error {
	parentLoadedTab, inMap := loadedRecordTab[` + "`" + `{{ .PointsTo.Info.PgName }}` + "`" + `]
	if !inMap {
		return fmt.Errorf("internal pggen error: table not pre-loaded")
	}
	parentIDToRecord := parentLoadedTab.(map[{{ .PointsToField.TypeInfo.Name }}]*{{ .PointsTo.Info.GoName }})
	ids := make([]{{ .PointsToField.TypeInfo.Name }}, 0, len(parentIDToRecord))
	for _, rec := range parentIDToRecord {
		ids = append(ids, rec.{{ .PointsToField.GoName }})
	}

	var childIDToRecord map[{{ .PointsFrom.Info.PkeyCol.TypeInfo.Name }}]*{{ .PointsFrom.Info.GoName }}
	childLoadedTab, inMap := loadedRecordTab[` + "`" + `{{ .PointsFrom.Info.PgName }}` + "`" + `]
	if inMap {
		childIDToRecord = childLoadedTab.(map[{{ .PointsFrom.Info.PkeyCol.TypeInfo.Name }}]*{{ .PointsFrom.Info.GoName }})
	} else {
		childIDToRecord = map[{{ .PointsFrom.Info.PkeyCol.TypeInfo.Name }}]*{{ .PointsFrom.Info.GoName }}{}
	}

	rows, err := p.db.QueryContext(
		ctx,
		` + "`" +
	`SELECT * FROM "{{ .PointsFrom.Info.PgName }}"
		 WHERE "{{ .PointsFromField.PgName }}" = ANY($1)
		 {{- if .PointsFrom.HasDeletedAtField }} AND "{{ .PointsFrom.PgDeletedAtField }}" IS NULL {{- end }}
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
		var scannedChildRec {{ .PointsFrom.Info.GoName }}
		err = scannedChildRec.Scan(ctx, p.client, rows)
		if err != nil {
			return err
		}

		var childRec *{{ .PointsFrom.Info.GoName }}

		preloadedChildRec, alreadyLoaded := childIDToRecord[scannedChildRec.{{ .PointsFrom.Info.PkeyCol.GoName }}]
		if alreadyLoaded {
			childRec = preloadedChildRec
		} else {
			childRec = &scannedChildRec
			{{- if .PointsFrom.Info.PkeyCol.Nullable }}
			childIDToRecord[*scannedChildRec.{{ .PointsFrom.Info.PkeyCol.GoName }}] = &scannedChildRec
			{{- else }}
			childIDToRecord[scannedChildRec.{{ .PointsFrom.Info.PkeyCol.GoName }}] = &scannedChildRec
			{{- end }}
		}

		{{- if .Nullable }}
		// we know that the foreign key can't be null because of the SQL query
		parentRec := parentIDToRecord[*childRec.{{ .PointsFromField.GoName }}]
		{{- else }}
		parentRec := parentIDToRecord[childRec.{{ .PointsFromField.GoName }}]
		{{- end }}

		{{- if .OneToOne }}
		parentRec.{{ .GoPointsFromFieldName }} = childRec
		{{- else }}
		parentRec.{{ .GoPointsFromFieldName }} = append(parentRec.{{ .GoPointsFromFieldName }}, childRec)
		{{- end }}
	}

	loadedRecordTab[` + "`" + `{{ .PointsFrom.Info.PgName }}` + "`" + `] = childIDToRecord

	return nil
}

{{ end }}
{{ range .Meta.AllOutgoingReferences }}

// For a given set of {{ $.GoName }}, fill in all the {{ .PointsTo.Info.GoName }}
// connected to them using at most one query.
func (p *pgClientImpl) private{{ $.GoName }}FillParent{{ .GoPointsToFieldName }}(
	ctx context.Context,
	loadedRecordTab map[string]interface{},
) error {
	// lookup the table of child records
	childLoadedTab, inMap := loadedRecordTab[` + "`" + `{{ .PointsFrom.Info.PgName }}` + "`" + `]
	if !inMap {
		return fmt.Errorf("internal pggen error: table not pre-loaded")
	}
	childIDToRecord := childLoadedTab.(map[{{ .PointsFrom.Info.PkeyCol.TypeInfo.Name }}]*{{ .PointsFrom.Info.GoName }})

	// lookup the table of parent records
	var parentIDToRecord map[{{ .PointsTo.Info.PkeyCol.TypeInfo.Name }}]*{{ .PointsTo.Info.GoName }}
	parentLoadedTab, inMap := loadedRecordTab[` + "`" + `{{ .PointsTo.Info.PgName }}` + "`" + `]
	if inMap {
		parentIDToRecord = parentLoadedTab.(map[{{ .PointsTo.Info.PkeyCol.TypeInfo.Name }}]*{{ .PointsTo.Info.GoName }})
	} else {
		parentIDToRecord = map[{{ .PointsTo.Info.PkeyCol.TypeInfo.Name }}]*{{ .PointsTo.Info.GoName }}{}
	}

	// partition the parents into those records which we have already loaded and those
	// which still need to be fetched from the db.
	ids := make([]{{ .PointsToField.TypeInfo.Name }}, 0, len(childIDToRecord))
	for _, rec := range childIDToRecord {
		{{- if .PointsFromField.Nullable }}
		if rec.{{ .PointsFromField.GoName }} == nil {
			continue
		}
		parentID := *rec.{{ .PointsFromField.GoName }}
		{{- else }}
		parentID := rec.{{ .PointsFromField.GoName }}
		{{- end }}

		parentRec, inMap := parentIDToRecord[parentID]
		if inMap {
			// already loaded, no need to hit the DB
			rec.{{ .GoPointsToFieldName }} = parentRec
		} else {
			ids = append(ids, parentID)
		}
	}

	// build a table mapping parent ids to lists of children which hold references to them
	parentIDToChildren := map[{{ .PointsTo.Info.PkeyCol.TypeInfo.Name }}][]*{{ .PointsFrom.Info.GoName }}{}
	for _, rec := range childIDToRecord {
		{{- if .PointsFromField.Nullable }}
		if rec.{{ .PointsFromField.GoName }} == nil {
			continue
		}
		parentID := *rec.{{ .PointsFromField.GoName }}
		{{- else }}
		parentID := rec.{{ .PointsFromField.GoName }}
		{{- end }}

		childSlice, inMap := parentIDToChildren[parentID]
		if inMap {
			childSlice = append(childSlice, rec)
			parentIDToChildren[parentID] = childSlice
		} else {
			parentIDToChildren[parentID] = []*{{ .PointsFrom.Info.GoName }}{rec}
		}
	}

	// fetch any outstanding parent records
	if len(ids) > 0 {
		rows, err := p.db.QueryContext(
			ctx,
		` + "`" +
	`SELECT * FROM "{{ .PointsTo.Info.PgName }}"
			WHERE "{{ .PointsToField.PgName }}" = ANY($1)
		 {{- if .PointsTo.HasDeletedAtField }} AND "{{ .PointsTo.PgDeletedAtField }}" IS NULL {{- end -}}
		 ` + "`" + `,
			pq.Array(ids),
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var parentRec {{ .PointsTo.Info.GoName }}
			err = parentRec.Scan(ctx, p.client, rows)
			if err != nil {
				return fmt.Errorf("scanning parent record: %s", err.Error())
			}

			childRecs := parentIDToChildren[parentRec.{{ .PointsTo.Info.PkeyCol.GoName }}]
			for _, childRec := range childRecs {
				childRec.{{ .GoPointsToFieldName }} = &parentRec
			}
			parentIDToRecord[parentRec.{{ .PointsTo.Info.PkeyCol.GoName }}] = &parentRec
		}
	}

	loadedRecordTab[` + "`" + `{{ .PointsTo.Info.PgName }}` + "`" + `] = parentIDToRecord

	return nil
}
{{ end }}

`))
