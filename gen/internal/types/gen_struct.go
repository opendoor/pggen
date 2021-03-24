// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package types

import (
	"strings"
	"text/template"
)

// file: gen_struct.go
// This file exposes an interface for generating a struct with a Scan routine.
// It is shared between the code for generating a query return value and the
// code for generating table code.

// genCtx should be a meta.TableGenCtx. We ask for an interface{} param to avoid a cyclic
// dependency. Using an interface{} is fine because we are just going to pass it into the
// template evaluator anyway.
func (r *Resolver) EmitStructType(typeName string, genCtx interface{}) error {
	var typeBody strings.Builder
	err := structTypeTmpl.Execute(&typeBody, genCtx)
	if err != nil {
		return err
	}

	var typeSig strings.Builder
	err = structTypeSigTmpl.Execute(&typeSig, genCtx)
	if err != nil {
		return err
	}

	err = r.EmitType(typeName, typeSig.String(), typeBody.String())
	if err != nil {
		return err
	}
	return nil
}

var structTypeSigTmpl *template.Template = template.Must(template.New("table-type-field-sig-tmpl").Parse(`
{{- range .Meta.Info.Cols }}
{{- if .Nullable }}
{{ .GoName }} {{ .TypeInfo.NullName }}
{{- else }}
{{ .GoName }} {{ .TypeInfo.Name }}
{{- end }}
{{- end }}
`))

var structTypeTmpl *template.Template = template.Must(template.New("struct-type-tmpl").Parse(`
type {{ .GoName }} struct {
	{{- range .Meta.Info.Cols }}
	{{- if .Nullable }}
	{{ .GoName }} {{ .TypeInfo.NullName }}
	{{- else }}
	{{ .GoName }} {{ .TypeInfo.Name }}
	{{- end }} ` +
	"`" + `{{ .Tags }}` + "`" + `
	{{- end }}
	{{- range .Meta.AllIncomingReferences }}
	{{- if .OneToOne }}
	{{ .GoPointsFromFieldName }} *{{ .PointsFrom.Info.GoName }} ` +
	"`" + `gorm:"foreignKey:{{ .PointsFromField.GoName }}"` + "`" + `
	{{- else }}
	{{ .GoPointsFromFieldName }} []*{{ .PointsFrom.Info.GoName }} ` +
	"`" + `gorm:"foreignKey:{{ .PointsFromField.GoName }}"` + "`" + `
	{{- end }}
	{{- end }}
	{{- range .Meta.AllOutgoingReferences }}
	{{- /* All outgoing references are 1-1, so we don't check the .OneToOne flag */}}
	{{ .GoPointsToFieldName }} *{{ .PointsTo.Info.GoName }}
	{{- end}}
}
func (r *{{ .GoName }}) Scan(ctx context.Context, client *PGClient, rs *sql.Rows) error {
	client.rwlockFor{{ .GoName }}.RLock()
	if client.colIdxTabFor{{ .GoName }} == nil {
		client.rwlockFor{{ .GoName }}.RUnlock() // release the lock to allow the write lock to be aquired
		err := client.fillColPosTab(
			ctx,
			genTimeColIdxTabFor{{ .GoName }},
			&client.rwlockFor{{ .GoName }},
			rs,
			&client.colIdxTabFor{{ .GoName }},
		)
		if err != nil {
			return err
		}
		client.rwlockFor{{ .GoName }}.RLock() // get the lock back for the rest of the routine
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
	client.rwlockFor{{ .GoName }}.RUnlock() // we are now done referencing the idx tab in the happy path

	err := rs.Scan(scanTgts...)
	if err != nil {
		// The database schema may have been changed out from under us, let's
		// check to see if we just need to update our column index tables and retry.
		colNames, colsErr := rs.Columns()
		if colsErr != nil {
			return fmt.Errorf("pggen: checking column names: %s", colsErr.Error())
		}
		client.rwlockFor{{ .GoName }}.RLock()
		if len(client.colIdxTabFor{{ .GoName }}) != len(colNames) {
			client.rwlockFor{{ .GoName }}.RUnlock() // release the lock to allow the write lock to be aquired
			err = client.fillColPosTab(
				ctx,
				genTimeColIdxTabFor{{ .GoName }},
				&client.rwlockFor{{ .GoName }},
				rs,
				&client.colIdxTabFor{{ .GoName }},
			)
			if err != nil {
				return err
			}

			return r.Scan(ctx, client, rs)
		} else {
			client.rwlockFor{{ .GoName }}.RUnlock()
			return err
		}
	}

	{{- range .Meta.Info.Cols }}
	{{- if .Nullable }}
	r.{{ .GoName }} = {{ call .TypeInfo.NullConvertFunc (printf "nullableTgts.scan%s" .GoName) }}
	{{- else if (eq .TypeInfo.Name "time.Time") }}
	r.{{ .GoName }} = {{ printf "nullableTgts.scan%s" .GoName }}.Time
	{{- end }}
	{{- end }}

	return nil
}

type nullableScanTgtsFor{{ .GoName }} struct {
	{{- range .Meta.Info.Cols }}
	{{- if (or .Nullable (eq .TypeInfo.Name "time.Time")) }}
	scan{{ .GoName }} {{ .TypeInfo.ScanNullName }}
	{{- end }}
	{{- end }}
}

// a table mapping codegen-time col indicies to functions returning a scanner for the
// field that was at that column index at codegen-time.
var scannerTabFor{{ .GoName }} = [...]func(*{{ .GoName }}, *nullableScanTgtsFor{{ .GoName }}) interface{} {
	{{- range .Meta.Info.Cols }}
	func (
		r *{{ $.GoName }},
		nullableTgts *nullableScanTgtsFor{{ $.GoName }},
	) interface{} {
		{{- if (or .Nullable (eq .TypeInfo.Name "time.Time")) }}
		return {{ call .TypeInfo.NullSqlReceiver (printf "nullableTgts.scan%s" .GoName) }}
		{{- else }}
		return {{ call .TypeInfo.SqlReceiver (printf "r.%s" .GoName) }}
		{{- end }}
	},
	{{- end }}
}

var genTimeColIdxTabFor{{ .GoName }} map[string]int = map[string]int{
	{{- range $i, $col := .Meta.Info.Cols }}
	` + "`" + `{{ $col.PgName }}` + "`" + `: {{ $i }},
	{{- end }}
}
`))
