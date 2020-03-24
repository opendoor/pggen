package gen

import (
	"io"
	"text/template"
)

func (g *Generator) genPGClient(into io.Writer, tables []tableConfig) error {
	g.imports[`"github.com/opendoor-labs/pggen"`] = true
	g.imports[`"database/sql"`] = true

	type genCtx struct {
		ModelNames []string
	}

	names := make([]string, 0, len(tables))
	for _, tc := range tables {
		names = append(names, pgTableToGoModel(tc.Name))
	}

	gCtx := genCtx{ModelNames: names}

	return pgClientTmpl.Execute(into, &gCtx)
}

var pgClientTmpl *template.Template = template.Must(template.New("pgclient-tmpl").Parse(`

// PGClient wraps either a 'sql.DB' or a 'sql.Tx'. All pggen-generated
// database access methods for this package are attached to it.
type PGClient struct {
	db pggen.DBHandle

	transactions []*sql.Tx
	topLevelDB *sql.DB

	// These column indexes are used at run time to enable us to 'SELECT *' against
	// a table that has the same columns in a different order from the ones that we
	// saw in the table we used to generate code. This means that you don't have to worry
	// about migrations merging in a slightly different order than their timestamps have
	// breaking 'SELECT *'.
	{{- range .ModelNames }}
	colIdxTabFor{{ . }} []int
	{{- end }}
}

func NewPGClient(conn *sql.DB) *PGClient {
	return &PGClient {
		db: conn,
		transactions: []*sql.Tx{},
		topLevelDB: conn,
	}
}

func (p *PGClient) Handle() pggen.DBHandle {
	return p.db
}

func (p *PGClient) BeginTx(ctx context.Context, opts *sql.TxOptions) error {
	if len(p.transactions) > 0 {
		return fmt.Errorf("nested transactions not yet supported")
	}

	tx, err := p.topLevelDB.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	p.transactions = append(p.transactions, tx)
	p.db = tx

	return nil
}

func (p *PGClient) Commit() error {
	if len(p.transactions) == 0 {
		return fmt.Errorf("no transaction to commit")
	}

	tx := p.transactions[len(p.transactions)-1]
	p.popTx()
	return tx.Commit()
}

func (p *PGClient) Rollback() error {
	if len(p.transactions) == 0 {
		return fmt.Errorf("no transaction to rollback")
	}

	tx := p.transactions[len(p.transactions)-1]
	p.popTx()
	return tx.Rollback()
}

func (p *PGClient) popTx() {
	p.transactions = p.transactions[:len(p.transactions)-1]
	if len(p.transactions) == 0 {
		p.db = p.topLevelDB
	} else {
		p.db = p.transactions[len(p.transactions)-1]
	}
}

`))
