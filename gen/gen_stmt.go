package gen

import (
	"io"
	"text/template"

	"github.com/opendoor/pggen/gen/internal/config"
	"github.com/opendoor/pggen/gen/internal/names"
)

func (g *Generator) genStmts(into io.Writer, stmts []config.StmtConfig) error {
	if len(stmts) > 0 {
		g.log.Infof("	generating %d statements\n", len(stmts))
	} else {
		return nil
	}

	g.imports[`"database/sql"`] = true
	g.imports[`"context"`] = true

	for i := range stmts {
		err := g.genStmt(into, &stmts[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) genStmt(into io.Writer, stmt *config.StmtConfig) error {
	g.log.Infof("		generating statement '%s'\n", stmt.Name)

	stmt.Name = names.PgToGoName(stmt.Name)

	meta, err := g.metaResolver.StmtMeta(stmt)
	if err != nil {
		return err
	}

	return stmtShimTmpl.Execute(into, meta)
}

var stmtShimTmpl *template.Template = template.Must(template.New("stmt-shim").Parse(`
{{ .Comment }}
func (p *PGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args}}
	{{- if $.ConfigData.NullableArguments }}
	{{ .GoName }} {{ .TypeInfo.NullName }},
	{{- else }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
	{{- end}}
) (sql.Result, error) {
	return p.impl.{{ .ConfigData.Name }}(
		ctx,
		{{- range .Args}}
		{{ .GoName }},
		{{- end}}
	)
}
{{ .Comment }}
func (tx *TxPGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args}}
	{{- if $.ConfigData.NullableArguments }}
	{{ .GoName }} {{ .TypeInfo.NullName }},
	{{- else }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
	{{- end}}
) (sql.Result, error) {
	return tx.impl.{{ .ConfigData.Name }}(
		ctx,
		{{- range .Args}}
		{{ .GoName }},
		{{- end}}
	)
}
{{ .Comment }}
func (conn *ConnPGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args}}
	{{- if $.ConfigData.NullableArguments }}
	{{ .GoName }} {{ .TypeInfo.NullName }},
	{{- else }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
	{{- end}}
) (sql.Result, error) {
	return conn.impl.{{ .ConfigData.Name }}(
		ctx,
		{{- range .Args}}
		{{ .GoName }},
		{{- end}}
	)
}
func (p *pgClientImpl) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args}}
	{{- if $.ConfigData.NullableArguments }}
	{{ .GoName }} {{ .TypeInfo.NullName }},
	{{- else }}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end }}
	{{- end}}
) (sql.Result, error) {
	res, err := p.db.ExecContext(
		ctx,
		` + "`" +
	`{{ .ConfigData.Body }}` +
	"`" + `,
		{{- range .Args }}
		{{- if $.ConfigData.NullableArguments }}
		{{ call .TypeInfo.NullSqlArgument .GoName }},
		{{- else }}
		{{ call .TypeInfo.SqlArgument .GoName }},
		{{- end }}
		{{- end }}
	)
	return res, p.client.errorConverter(err)
}

`))
