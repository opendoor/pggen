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

// genInterfaces emits the DBQueries interface shared between the generated PGClient
// and the generated TxPGClient. This allows user code to be written in such a way to
func (g *Generator) genInterfaces(into io.Writer, conf *config.DbConfig) error {
	g.log.Infof("	generating DBQueries interface\n")

	var genCtx ifaceGenCtx

	// populate tables
	genCtx.Tables = make([]tableIfaceGenCtx, 0, len(conf.Tables))
	for _, tc := range conf.Tables {
		tableInfo, ok := g.metaResolver.TableMeta(tc.Name)
		if !ok {
			return fmt.Errorf("could get schema info about table '%s'", tc.Name)
		}

		genCtx.Tables = append(genCtx.Tables, tableIfaceGenCtx{
			GoName:   tableInfo.Info.GoName,
			PkeyType: tableInfo.Info.PkeyCol.TypeInfo.Name,
		})
	}

	// poplulate queries
	// TODO(ethan): avoid hitting the database twice for query metadata
	genCtx.Queries = make([]meta.QueryMeta, 0, len(conf.Queries))
	for _, qc := range conf.Queries {
		meta, err := g.metaResolver.QueryMeta(&qc, true /* inferArgTypes */)
		if err != nil {
			return err
		}
		genCtx.Queries = append(genCtx.Queries, meta)
	}

	// populate stored function metadata
	// TODO(ethan): avoid computing this stuff twice
	genCtx.StoredFuncs = make([]meta.QueryMeta, 0, len(conf.StoredFunctions))
	for _, sfc := range conf.StoredFunctions {
		args, err := g.metaResolver.FuncArgs(sfc.Name)
		if err != nil {
			return err
		}

		var queryTxt strings.Builder
		err = storedFuncQueryTmpl.Execute(&queryTxt, map[string]interface{}{
			"name": sfc.Name,
			"args": args,
		})
		if err != nil {
			return err
		}

		// generate a fake query config because stored procs are
		// just a special case of queries where we can do a little
		// bit better when it comes to naming arguments.
		queryConf := config.QueryConfig{
			Name:          names.PgToGoName(sfc.Name),
			Body:          queryTxt.String(),
			NullFlags:     sfc.NullFlags,
			NotNullFields: sfc.NotNullFields,
			ReturnType:    names.PgTableToGoModel(sfc.ReturnType),
		}

		meta, err := g.metaResolver.QueryMeta(&queryConf, false /* inferArgTypes */)
		if err != nil {
			return err
		}

		genCtx.StoredFuncs = append(genCtx.StoredFuncs, meta)
	}

	// populate the statement gen ctx
	// TODO(ethan): avoid computing this stuff twice
	genCtx.Stmts = make([]meta.StmtMeta, 0, len(conf.Stmts))
	for _, sc := range conf.Stmts {
		meta, err := g.metaResolver.StmtMeta(&sc)
		if err != nil {
			return err
		}
		genCtx.Stmts = append(genCtx.Stmts, meta)
	}

	return dbQueriesTmpl.Execute(into, genCtx)
}

type tableIfaceGenCtx struct {
	GoName   string
	PkeyType string
}

type queryIfaceGenCtx struct {
	Name           string
	ReturnTypeName string
}

type ifaceGenCtx struct {
	Tables      []tableIfaceGenCtx
	Queries     []meta.QueryMeta
	StoredFuncs []meta.QueryMeta
	Stmts       []meta.StmtMeta
}

var dbQueriesTmpl *template.Template = template.Must(template.New("db-queries-tmpl").Parse(`

type DBQueries interface {
	//
	// automatic CRUD methods
	//

	{{ range .Tables }}
	// {{ .GoName }} methods
	Get{{ .GoName }}(ctx context.Context, id {{ .PkeyType }}) (*{{ .GoName }}, error)
	List{{ .GoName }}(ctx context.Context, ids []{{ .PkeyType }}) ([]{{ .GoName }}, error)
	Insert{{ .GoName }}(ctx context.Context, value *{{ .GoName }}, opts ...pggen.InsertOpt) ({{ .PkeyType }}, error)
	BulkInsert{{ .GoName }}(ctx context.Context, values []{{ .GoName }}, opts ...pggen.InsertOpt) ([]{{ .PkeyType }}, error)
	Update{{ .GoName }}(ctx context.Context, value *{{ .GoName }}, fieldMask pggen.FieldSet) (ret {{ .PkeyType }}, err error)
	Upsert{{ .GoName }}(ctx context.Context, value *{{ .GoName }}, constraintNames []string, fieldMask pggen.FieldSet) ({{ .PkeyType }}, error)
	BulkUpsert{{ .GoName }}(ctx context.Context, values []{{ .GoName }}, constraintNames []string, fieldMask pggen.FieldSet) ([]{{ .PkeyType }}, error)
	Delete{{ .GoName }}(ctx context.Context, id {{ .PkeyType }}) error
	BulkDelete{{ .GoName }}(ctx context.Context, ids []{{ .PkeyType }}) error
	{{ .GoName }}FillIncludes(ctx context.Context, rec *{{ .GoName }}, includes *include.Spec) error
	{{ .GoName }}BulkFillIncludes(ctx context.Context, recs []*{{ .GoName }}, includes *include.Spec) error
	{{ end }}

	//
	// query methods
	//

	{{ range .Queries }}
	// {{ .ConfigData.Name }} query
	{{ .ConfigData.Name }}(
		ctx context.Context,
		{{- range .Args }}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end }}
	) ([]{{ .ReturnTypeName }}, error)
	{{ .ConfigData.Name }}Query(
		ctx context.Context,
		{{- range .Args }}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end }}
	) (*sql.Rows, error)
	{{ end }}

	//
	// stored function methods
	//

	{{ range .StoredFuncs }}
	// {{ .ConfigData.Name }} stored function
	{{ .ConfigData.Name }}(
		ctx context.Context,
		{{- range .Args }}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end }}
	) ([]{{ .ReturnTypeName }}, error)
	{{ .ConfigData.Name }}Query(
		ctx context.Context,
		{{- range .Args }}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end }}
	) (*sql.Rows, error)
	{{ end }}

	//
	// stmt methods
	//

	{{ range .Stmts }}
	// {{ .ConfigData.Name }} stmt
	{{ .ConfigData.Name }}(
		ctx context.Context,
		{{- range .Args}}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end}}
	) (sql.Result, error)
	{{ end }}
}

`))
