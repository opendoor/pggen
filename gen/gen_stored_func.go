package gen

import (
	"strings"
	"text/template"

	"github.com/opendoor-labs/pggen/gen/internal/config"
	"github.com/opendoor-labs/pggen/gen/internal/meta"
	"github.com/opendoor-labs/pggen/gen/internal/names"
)

func (g *Generator) genStoredFuncs(
	into *strings.Builder,
	funcs []config.StoredFuncConfig,
) error {
	if len(funcs) == 0 {
		return nil
	}

	g.log.Infof("	generating %d stored functions\n", len(funcs))

	for i := range funcs {
		// generate a fake query config because stored procs are
		// just a special case of queries where we can do a little
		// bit better when it comes to naming arguments.
		queryConf, args, err := g.storedFuncToQueryConf(&funcs[i])
		if err != nil {
			return err
		}

		err = g.genQuery(into, queryConf, args)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) storedFuncToQueryConf(
	storedFunc *config.StoredFuncConfig,
) (*config.QueryConfig, []meta.Arg, error) {
	args, err := g.metaResolver.FuncArgs(storedFunc.Name)
	if err != nil {
		return nil, nil, err
	}

	var queryTxt strings.Builder
	err = storedFuncQueryTmpl.Execute(&queryTxt, map[string]interface{}{
		"name": storedFunc.Name,
		"args": args,
	})
	if err != nil {
		return nil, nil, err
	}

	return &config.QueryConfig{
		Name:          names.PgToGoName(storedFunc.Name),
		Body:          queryTxt.String(),
		NullFlags:     storedFunc.NullFlags,
		NotNullFields: storedFunc.NotNullFields,
		ReturnType:    storedFunc.ReturnType,
	}, args, nil
}

var storedFuncQueryTmpl *template.Template = template.Must(template.New("stored-func-shim").Parse(`
SELECT * FROM "{{ index . "name" }}"(
	{{- range $i, $a := index . "args" -}}
		{{- if $i }},{{end -}}
		${{ $a.Idx -}}
	{{- end }})`))
