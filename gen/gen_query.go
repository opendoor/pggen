package gen

import (
	"fmt"
	"strings"
	"text/template"
)

func (g *Generator) genQueries(
	into *strings.Builder,
	queries []queryConfig,
) error {
	if len(queries) > 0 {
		g.infof("	generating %d queries\n", len(queries))
	} else {
		return nil
	}

	g.imports[`"database/sql"`] = true
	g.imports[`"context"`] = true

	for _, query := range queries {
		err := g.genQuery(into, &query, nil)
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
	config *queryConfig,
	args []arg,
) error {
	g.infof("		generating query '%s'\n", config.Name)

	// ensure that the query name is in the right format for go
	config.Name = snakeToPascal(config.Name)

	// not needed, but it does make the generated code a little nicer
	config.Body = strings.TrimSpace(config.Body)

	meta, err := g.queryMeta(config, args == nil /* inferArgTypes */)
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

		err = g.types.emitType(meta.ReturnTypeName, typeSig.String(), typeBody.String())
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
func (r *{{ .ReturnTypeName }}) Scan(rs *sql.Rows) error {
	return rs.Scan(
		{{- range .ReturnCols }}
		{{ call .TypeInfo.SqlReceiver (printf "r.%s" .GoName) }},
		{{- end }}
	)
}
`))

var queryShimTmpl = template.Must(template.New("query-shim").Parse(`
func (p *PGClient) {{ .ConfigData.Name }}(
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
		err = rows.Close()
		if err != nil {
			ret = nil
		}
	}()

	for rows.Next() {
		var row {{ .ReturnTypeName }}
		{{- if .MultiReturn }}
		err = row.Scan(rows)
		{{- else }}
		err = rows.Scan({{ call (index .ReturnCols 0).TypeInfo.SqlReceiver "row" }})
		{{- end }}
		if err != nil {
			return nil, err
		}
		ret = append(ret, row)
	}

	return
}
func (p *PGClient) {{ .ConfigData.Name }}Query(
	ctx context.Context,
	{{- range .Args}}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end}}
) (*sql.Rows, error) {
	return p.DB.QueryContext(
		ctx,
		` + "`" +
	`{{ .ConfigData.Body }}` +
	"`" + `,
		{{- range .Args }}
		{{ .GoName }},
		{{- end }}
	)
}

`))
