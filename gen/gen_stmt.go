package gen

import (
	"io"
	"text/template"
)

func (g *Generator) genStmts(into io.Writer, stmts []stmtConfig) error {
	if len(stmts) > 0 {
		g.infof("	generating %d statements\n", len(stmts))
	} else {
		return nil
	}

	g.imports[`"database/sql"`] = true
	g.imports[`"context"`] = true

	for _, stmt := range stmts {
		err := g.genStmt(into, &stmt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) genStmt(into io.Writer, stmt *stmtConfig) error {
	g.infof("		generating statement '%s'\n", stmt.Name)

	stmt.Name = pgToGoName(stmt.Name)

	meta, err := g.stmtMeta(stmt)
	if err != nil {
		return err
	}

	return stmtShimTmpl.Execute(into, meta)
}

var stmtShimTmpl *template.Template = template.Must(template.New("stmt-shim").Parse(`
func (p *PGClient) {{ .ConfigData.Name }}(
	ctx context.Context,
	{{- range .Args}}
	{{ .GoName }} {{ .TypeInfo.Name }},
	{{- end}}
) (sql.Result, error) {
	return p.DB.ExecContext(
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
