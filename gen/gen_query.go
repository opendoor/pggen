package gen

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/opendoor-labs/pggen/gen/internal/config"
	"github.com/opendoor-labs/pggen/gen/internal/meta"
	"github.com/opendoor-labs/pggen/gen/internal/names"
)

func (g *Generator) genQueries(
	into *strings.Builder,
	queries []config.QueryConfig,
) error {
	if len(queries) > 0 {
		g.log.Infof("	generating %d queries\n", len(queries))
	} else {
		return nil
	}

	g.imports[`"database/sql"`] = true
	g.imports[`"context"`] = true
	g.imports[`"fmt"`] = true

	g.imports[`"github.com/opendoor-labs/pggen/unstable"`] = true
	// HACK: not really a type, but the type resolver can be used to ensure that
	//       exactly one copy of this declaration makes it into the final output.
	err := g.typeResolver.EmitType("ensure-stable-used", "sig", "var _ = unstable.NotFoundError{}")
	if err != nil {
		return fmt.Errorf("internal-error: emitting bogus NotFoundError usage: %s", err)
	}

	for i, query := range queries {
		err := g.genQuery(into, &queries[i], nil)
		if err != nil {
			return fmt.Errorf("generating query '%s': %s", query.Name, err.Error())
		}
	}

	return nil
}

// generate a query for the given config. If `args` is provided, use it
// instead of the inferred argument types.
func (g *Generator) genQuery(
	into *strings.Builder,
	config *config.QueryConfig,
	args []meta.Arg,
) error {
	g.log.Infof("		generating query '%s'\n", config.Name)

	// ensure that the query name is in the right format for go
	config.Name = names.PgToGoName(config.Name)

	// not needed, but it does make the generated code a little nicer
	config.Body = strings.TrimSpace(config.Body)

	if config.Body == "" {
		return fmt.Errorf("empty query body")
	}

	meta, err := g.metaResolver.QueryMeta(config, args == nil /* inferArgTypes */)
	if err != nil {
		return err
	}
	if args != nil {
		meta.Args = args
	}

	if meta.MultiReturn {
		var typeBody strings.Builder
		err = queryRetTypeTmpl.Execute(&typeBody, meta)
		if err != nil {
			return err
		}

		var typeSig strings.Builder
		err = queryRetTypeSigTmpl.Execute(&typeSig, meta)
		if err != nil {
			return err
		}

		err = g.typeResolver.EmitType(meta.ReturnTypeName, typeSig.String(), typeBody.String())
		if err != nil {
			return err
		}
	}

	return queryShimTmpl.Execute(into, meta)
}

var queryRetTypeSigTmpl *template.Template = template.Must(template.New("query-ret-type-sig").Parse(`
{{- range .ReturnCols }}
{{- if .Nullable }}
{{ .GoName }} {{ .TypeInfo.NullName }}
{{- else }}
{{ .GoName }} {{ .TypeInfo.Name }}
{{- end }}
{{- end }}
`))

var queryRetTypeTmpl *template.Template = template.Must(template.New("query-ret-type").Parse(`
type {{ .ReturnTypeName }} struct {
	{{- range .ReturnCols }}
	{{- if .Nullable }}
	{{ .GoName }} {{ .TypeInfo.NullName }}
	{{- else }}
	{{ .GoName }} {{ .TypeInfo.Name }}
	{{- end }}
	{{- end }}
}
func (r *{{ .ReturnTypeName }}) Scan(ctx context.Context, client *PGClient, rs *sql.Rows) error {
	{{- range .ReturnCols }}
	{{- if (or .Nullable (eq .TypeInfo.Name "time.Time")) }}
	var scan{{ .GoName }} {{ .TypeInfo.ScanNullName }}
	{{- end }}
	{{- end }}

	err := rs.Scan(
		{{- range .ReturnCols }}
		{{- if (or .Nullable (eq .TypeInfo.Name "time.Time")) }}
		{{ call .TypeInfo.NullSqlReceiver (printf "scan%s" .GoName) }},
		{{- else }}
		{{ call .TypeInfo.SqlReceiver (printf "r.%s" .GoName) }},
		{{- end }}
		{{- end }}
	)
	if err != nil {
		return err
	}

	{{- range .ReturnCols }}
	{{- if .Nullable }}
	r.{{ .GoName }} = {{ call .TypeInfo.NullConvertFunc (printf "scan%s" .GoName) }}
	{{- else if (eq .TypeInfo.Name "time.Time") }}
	r.{{ .GoName }} = {{ printf "scan%s" .GoName }}.Time
	{{- end }}
	{{- end }}

	return nil
}
`))

var queryShimTmpl = template.Must(template.New("query-shim").Parse(`
{{ if .ConfigData.SingleResult }}
func (p *PGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
{{- if (not .MultiReturn) }}
) (ret {{ .ReturnTypeName }}, err error) {
{{- else }}
) (ret *{{ .ReturnTypeName }}, err error) {
{{- end }}
	return p.impl.{{ .ConfigData.Name }}(
		ctx,
		{{- range .Args }}
		{{ .GoName }},
		{{- end }}
	)
}
func (p *TxPGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
{{- if (not .MultiReturn) }}
) (ret {{ .ReturnTypeName }}, err error) {
{{- else }}
) (ret *{{ .ReturnTypeName }}, err error) {
{{- end }}
	return p.impl.{{ .ConfigData.Name }}(
		ctx,
		{{- range .Args }}
		{{ .GoName }},
		{{- end }}
	)
}
func (p *pgClientImpl) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
{{- if (not .MultiReturn) }}
) (ret {{ .ReturnTypeName }}, err error) {
{{- else }}
) (ret *{{ .ReturnTypeName }}, err error) {
{{- end }}
	{{- if (not .MultiReturn) }}
	var zero {{ .ReturnTypeName }}
	{{- else }}
	var zero *{{ .ReturnTypeName }}
	{{- end }}

	// we still use QueryConfig rather than QueryRowContext so the scan
	// impl remains consistant. We don't need to split out a seperate Query
	// method though.
	var rows *sql.Rows
	rows, err = p.db.QueryContext(
		ctx,
		` + "`" +
	`{{ .ConfigData.Body }}` +
	"`" + `,
		{{- range .Args }}
		{{ call .TypeInfo.SqlArgument .GoName }},
		{{- end }}
	)
	if err != nil {
		return zero, err
	}
	defer func() {
		if err == nil {
			err = rows.Close()
			if err != nil {
				ret = zero
			}
		} else {
			rowErr := rows.Close()
			if rowErr != nil {
				err = fmt.Errorf("%s AND %s", err.Error(), rowErr.Error())
			}
		}
	}()

	if !rows.Next() {
		return zero, &unstable.NotFoundError{ Msg: "{{ .ConfigData.Name }}: no results" }
	}

	{{- if .MultiReturn }}
	ret = &{{ .ReturnTypeName }}{}
	err = ret.Scan(ctx, p.client, rows)
	{{- else }}
	{{- if (index .ReturnCols 0).Nullable }}
	var scanTgt {{ (index .ReturnCols 0).TypeInfo.ScanNullName }}
	err = rows.Scan({{ call (index .ReturnCols 0).TypeInfo.NullSqlReceiver "scanTgt" }})
	if err != nil {
		return zero, err
	}
	ret = {{ call (index .ReturnCols 0).TypeInfo.NullConvertFunc "scanTgt" }}
	{{- else }}
	err = rows.Scan({{ call (index .ReturnCols 0).TypeInfo.SqlReceiver "ret" }})
	if err != nil {
		return zero, err
	}
	{{- end }}
	{{- end }}

	return
}
{{- else }}
func (p *PGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
) (ret []{{ .ReturnTypeName }}, err error) {
	return p.impl.{{ .ConfigData.Name }}(
		ctx,
		{{- range .Args }}
		{{ .GoName }},
		{{- end }}
	)
}
func (tx *TxPGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
) (ret []{{ .ReturnTypeName }}, err error) {
	return tx.impl.{{ .ConfigData.Name }}(
		ctx,
		{{- range .Args }}
		{{ .GoName }},
		{{- end }}
	)
}
func (p *pgClientImpl) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
) (ret []{{ .ReturnTypeName }}, err error) {
	ret = []{{ .ReturnTypeName }}{}

	var rows *sql.Rows
	rows, err = p.{{ .ConfigData.Name }}Query(
		ctx,
		{{- range .Args}}
		{{ .GoName }},
		{{- end}}
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

	for rows.Next() {
		var row {{ .ReturnTypeName }}
		{{- if .MultiReturn }}
		err = row.Scan(ctx, p.client, rows)
		{{- else }}
		{{- if (index .ReturnCols 0).Nullable }}
		var scanTgt {{ (index .ReturnCols 0).TypeInfo.ScanNullName }}
		err = rows.Scan({{ call (index .ReturnCols 0).TypeInfo.NullSqlReceiver "scanTgt" }})
		if err != nil {
			return nil, err
		}
		row = {{ call (index .ReturnCols 0).TypeInfo.NullConvertFunc "scanTgt" }}
		{{- else }}
		err = rows.Scan({{ call (index .ReturnCols 0).TypeInfo.SqlReceiver "row" }})
		if err != nil {
			return nil, err
		}
		{{- end }}
		{{- end }}
		ret = append(ret, row)
	}

	return
}

func (p *PGClient) {{ .ConfigData.Name }}Query(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
) (*sql.Rows, error) {
	return p.impl.{{ .ConfigData.Name }}Query(
		ctx,
		{{- range .Args}}
		{{ .GoName }},
		{{- end}}
	)
}
func (tx *TxPGClient) {{ .ConfigData.Name }}Query(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
) (*sql.Rows, error) {
	return tx.impl.{{ .ConfigData.Name }}Query(
		ctx,
		{{- range .Args}}
		{{ .GoName }},
		{{- end}}
	)
}
func (p *pgClientImpl) {{ .ConfigData.Name }}Query(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
) (*sql.Rows, error) {
	return p.db.QueryContext(
		ctx,
		` + "`" +
	`{{ .ConfigData.Body }}` +
	"`" + `,
		{{- range .Args }}
		{{ call .TypeInfo.SqlArgument .GoName }},
		{{- end }}
	)
}

{{- end }}
`))
