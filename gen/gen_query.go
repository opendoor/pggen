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
	requireComments bool,
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
	err := g.typeResolver.EmitType("ensure-unstable-used", "sig", "var _ = unstable.NotFoundError{}")
	if err != nil {
		return fmt.Errorf("internal-error: emitting bogus NotFoundError usage: %s", err)
	}

	for i, query := range queries {
		if requireComments && query.Comment == "" {
			return fmt.Errorf("query '%s' is missing a comment but require_query_comments is set", query.Name)
		}

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

	// tables in non-public schemas are allowed to have underscores in their names, so
	// we don't want to convert in that case.
	if !g.typeResolver.Probe(config.ReturnType) {
		config.ReturnType = names.PgToGoName(config.ReturnType)
	}

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
		genCtx := buildTableGenCtx(&meta)
		err = g.typeResolver.EmitStructType(meta.ReturnTypeName, &genCtx)
		if err != nil {
			return fmt.Errorf("generating return struct for '%s': %s", config.Name, err.Error())
		}
	}

	return queryShimTmpl.Execute(into, meta)
}

// buildTableGenCtx converts a meta.QueryMeta object into a fake table gen context
// that is good enough to use to generate a return type and scan method.
//
// poison the strings so that mistakes are easier to spot
func buildTableGenCtx(qm *meta.QueryMeta) meta.TableGenCtx {
	return meta.TableGenCtx{
		PgName:         "BOGUS_PGNAME",
		GoName:         qm.ConfigData.Name + "Row",
		PkeyColIdx:     -1,
		AllIncludeSpec: "BOGUS_ALL_INCLUDE_SPEC",
		Meta: &meta.TableMeta{
			Info: meta.PgTableInfo{
				PgName:       "BOGUS_PGNAME-inner",
				GoName:       qm.ConfigData.Name + "Row",
				PluralGoName: "BOGUS_PLURAL_GONAME-inner",
				Cols:         qm.ReturnCols,
			},
		},
	}
}

var queryShimTmpl = template.Must(template.New("query-shim").Parse(`
{{ if .ConfigData.SingleResult }}
{{ .Comment }}
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
{{ .Comment }}
func (tx *TxPGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
{{- if (not .MultiReturn) }}
) (ret {{ .ReturnTypeName }}, err error) {
{{- else }}
) (ret *{{ .ReturnTypeName }}, err error) {
{{- end }}
	return tx.impl.{{ .ConfigData.Name }}(
		ctx,
		{{- range .Args }}
		{{ .GoName }},
		{{- end }}
	)
}
{{ .Comment }}
func (conn *ConnPGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
{{- if (not .MultiReturn) }}
) (ret {{ .ReturnTypeName }}, err error) {
{{- else }}
) (ret *{{ .ReturnTypeName }}, err error) {
{{- end }}
	return conn.impl.{{ .ConfigData.Name }}(
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
{{ .Comment }}
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
{{ .Comment }}
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
{{ .Comment }}
func (conn *ConnPGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
) (ret []{{ .ReturnTypeName }}, err error) {
	return conn.impl.{{ .ConfigData.Name }}(
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

{{ .Comment }}
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
{{ .Comment }}
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
{{ .Comment }}
func (conn *ConnPGClient) {{ .ConfigData.Name }}Query(
	ctx context.Context,
	{{- range .Args }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
) (*sql.Rows, error) {
	return conn.impl.{{ .ConfigData.Name }}Query(
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
