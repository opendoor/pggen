package pggen

// db_handle.go defines the common interface shared by *sql.Tx and *sql.DB

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"
)

// DBHandle is an interface which contains the methods common to
// *sql.Tx and *sql.DB that pggen uses, allowing for code to be generic over
// whether or no the user is operating in a transaction.
type DBHandle interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// DBConn is an interface which contains the methods from `sql.DB` that pggen
// uses. Making the generated `NewPGClient` functions take a `DBConn` rather
// than a `*sql.DB` allows users to wrap the database connection with their own object
// that performs custom logging, tracing ...
type DBConn interface {
	DBHandle
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)

	// The below methods are unused, but we don't want to have to break compatibility to
	// use them in the future.
	Close() error
	Conn(ctx context.Context) (*sql.Conn, error)
	Driver() driver.Driver
	PingContext(ctx context.Context) error
	SetConnMaxLifetime(d time.Duration)
	SetMaxIdleConns(n int)
	SetMaxOpenConns(n int)
	Stats() sql.DBStats
}
