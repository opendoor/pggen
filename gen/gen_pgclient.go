package gen

import (
	"io"
	"text/template"

	"github.com/opendoor-labs/pggen/gen/internal/config"
	"github.com/opendoor-labs/pggen/gen/internal/names"
)

func (g *Generator) genPGClient(into io.Writer, tables []config.TableConfig) error {
	g.imports[`"github.com/opendoor-labs/pggen"`] = true
	g.imports[`"database/sql"`] = true

	type genCtx struct {
		ModelNames []string
	}

	modelNames := make([]string, 0, len(tables))
	for _, tc := range tables {
		modelNames = append(modelNames, names.PgTableToGoModel(tc.Name))
	}

	gCtx := genCtx{ModelNames: modelNames}

	return pgClientTmpl.Execute(into, &gCtx)
}

var pgClientTmpl *template.Template = template.Must(template.New("pgclient-tmpl").Parse(`

// PGClient wraps either a 'sql.DB' or a 'sql.Tx'. All pggen-generated
// database access methods for this package are attached to it.
type PGClient struct {
	impl pgClientImpl
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
	client := PGClient {
		topLevelDB: conn,
	}
	client.impl = pgClientImpl{
		db: conn,
		client: &client,
	}

	return &client
}

func (p *PGClient) Handle() pggen.DBHandle {
	return p.topLevelDB
}

func (p *PGClient) BeginTx(ctx context.Context, opts *sql.TxOptions) (*TxPGClient, error) {
	tx, err := p.topLevelDB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &TxPGClient{
		impl: pgClientImpl{
			db: tx,
			client: p,
		},
	}, nil
}

// A postgres client that operates within a transaction. Supports all the same
// generated methods that PGClient does.
type TxPGClient struct {
	impl pgClientImpl
}

func (tx *TxPGClient) Handle() pggen.DBHandle {
	return tx.impl.db.(*sql.Tx)
}

func (tx *TxPGClient) Rollback() error {
	return tx.impl.db.(*sql.Tx).Rollback()
}

func (tx *TxPGClient) Commit() error {
	return tx.impl.db.(*sql.Tx).Commit()
}

// A database client that can wrap either a direct database connection or a transaction
type pgClientImpl struct {
	db pggen.DBHandle
	// a reference back to the owning PGClient so we can always get at the resolver tables
	client *PGClient
}

`))
