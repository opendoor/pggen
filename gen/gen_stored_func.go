package gen

import (
	"strings"
	"text/template"

	"github.com/opendoor-labs/pggen/gen/internal/config"
)

func (g *Generator) genStoredFuncs(
	into *strings.Builder,
	funcs []config.StoredFuncConfig,
) error {
	if len(funcs) == 0 {
		return nil
	}

	g.log.Infof("	generating %d stored functions\n", len(funcs))

	for _, storedFunc := range funcs {
		args, err := g.metaResolver.FuncArgs(storedFunc.Name)
		if err != nil {
			return err
		}

		var queryTxt strings.Builder
		err = storedFuncQueryTmpl.Execute(&queryTxt, map[string]interface{}{
			"name": storedFunc.Name,
			"args": args,
		})
		if err != nil {
			return err
		}

		// generate a fake query config because stored procs are
		// just a special case of queries where we can do a little
		// bit better when it comes to naming arguments.
		queryConf := config.QueryConfig{
			Name:          storedFunc.Name,
			Body:          queryTxt.String(),
			NullFlags:     storedFunc.NullFlags,
			NotNullFields: storedFunc.NotNullFields,
			ReturnType:    storedFunc.ReturnType,
		}

		err = g.genQuery(into, &queryConf, args)
		if err != nil {
			return err
		}
	}

	return nil
}

var storedFuncQueryTmpl *template.Template = template.Must(template.New("stored-func-shim").Parse(`
SELECT * FROM "{{ index . "name" }}"(
	{{- range $i, $a := index . "args" -}}
		{{- if $i }},{{end -}}
		${{ $a.Idx -}}
	{{- end }})`))
