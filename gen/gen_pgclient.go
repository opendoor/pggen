// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package gen

import (
	"io"
	"text/template"

	"github.com/opendoor-labs/pggen/gen/internal/config"
	"github.com/opendoor-labs/pggen/gen/internal/names"
)

func (g *Generator) genPGClient(into io.Writer, conf *config.DbConfig) error {
	g.imports[`"github.com/opendoor-labs/pggen"`] = true
	g.imports[`"database/sql"`] = true
	g.imports[`"sync"`] = true

	type genCtx struct {
		ScanStructNames []string
	}

	scanStructNames := make([]string, 0, len(conf.Tables))
	for _, tc := range conf.Tables {
		scanStructNames = append(scanStructNames, names.PgTableToGoModel(tc.Name))
	}
	for _, qc := range conf.Queries {
		scanStructNames = append(scanStructNames, names.PgToGoName(qc.Name)+"Row")
	}

	gCtx := genCtx{ScanStructNames: scanStructNames}

	return pgClientTmpl.Execute(into, &gCtx)
}

var pgClientTmpl *template.Template = template.Must(template.New("pgclient-tmpl").Parse(`

// PGClient wraps either a 'sql.DB' or a 'sql.Tx'. All pggen-generated
// database access methods for this package are attached to it.
type PGClient struct {
	impl pgClientImpl
	topLevelDB pggen.DBConn

	// These column indexes are used at run time to enable us to 'SELECT *' against
	// a table that has the same columns in a different order from the ones that we
	// saw in the table we used to generate code. This means that you don't have to worry
	// about migrations merging in a slightly different order than their timestamps have
	// breaking 'SELECT *'.
	{{- range .ScanStructNames }}
	rwlockFor{{ . }} sync.RWMutex
	colIdxTabFor{{ . }} []int
	{{- end }}
}

// bogus usage so we can compile with no tables configured
var _ = sync.RWMutex{}

// NewPGClient creates a new PGClient out of a '*sql.DB' or a
// custom wrapper around a db connection.
//
// If you provide your own wrapper around a '*sql.DB' for logging or
// custom tracing, you MUST forward all calls to an underlying '*sql.DB'
// member of your wrapper.
func NewPGClient(conn pggen.DBConn) *PGClient {
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

func (p *PGClient) Conn(ctx context.Context) (*ConnPGClient, error) {
	conn, err := p.topLevelDB.Conn(ctx)
	if err != nil {
		return nil, err
	}

	return &ConnPGClient{impl: pgClientImpl{ db: conn, client: p }}, nil
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

type ConnPGClient struct {
	impl pgClientImpl
}

func (conn *ConnPGClient) Close() error {
	return conn.impl.db.(*sql.Conn).Close()
}

func (conn *ConnPGClient) Handle() pggen.DBHandle {
	return conn.impl.db
}

// A database client that can wrap either a direct database connection or a transaction
type pgClientImpl struct {
	db pggen.DBHandle
	// a reference back to the owning PGClient so we can always get at the resolver tables
	client *PGClient
}

`))
