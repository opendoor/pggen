// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
// The middleware package is used in pggen to add middleware to be executed
// surrounding the DB calls execution. The intent is to have the ability to add cutom
// logging, metrics, tracing, side effects ...
package middleware

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/opendoor-labs/pggen"
)

type ExecFunc func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
type ExecMiddleware func(ExecFunc) ExecFunc

type QueryFunc func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
type QueryMiddleware func(QueryFunc) QueryFunc

type QueryRowFunc func(ctx context.Context, query string, args ...interface{}) *sql.Row
type QueryRowMiddleware func(QueryRowFunc) QueryRowFunc

type BeginTxFunc func(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
type BeginTxMiddleware func(BeginTxFunc) BeginTxFunc

// DBConnWrapper is a wrapper around DBConn that also contain the middlewares to apply when doing the DB calls
type DBConnWrapper struct {
	dbConn pggen.DBConn

	execFunc     ExecFunc
	queryFunc    QueryFunc
	queryRowFunc QueryRowFunc
	beginTxFunc  BeginTxFunc
}

// NewDBConnWrapper wraps the DBConn in struct to which middlewares can be added
func NewDBConnWrapper(dbConn pggen.DBConn) *DBConnWrapper {
	return &DBConnWrapper{
		dbConn: dbConn,

		execFunc:     dbConn.ExecContext,
		queryFunc:    dbConn.QueryContext,
		queryRowFunc: dbConn.QueryRowContext,
		beginTxFunc:  dbConn.BeginTx,
	}
}

// WithExecMiddleware adds the middleware for the ExecContext to the DBConnWrapper
func (w *DBConnWrapper) WithExecMiddleware(execMiddleware ExecMiddleware) *DBConnWrapper {
	w.execFunc = execMiddleware(w.execFunc)
	return w
}

// ExecContext apply the middleware if any and execute ExecContext on the wrapped DBConn
func (w *DBConnWrapper) ExecContext(ctx context.Context, stmt string, args ...interface{}) (sql.Result, error) {
	return w.execFunc(ctx, stmt, args...)
}

// WithQueryMiddleware adds the middleware for the QueryContext to the DBConnWrapper
func (w *DBConnWrapper) WithQueryMiddleware(queryMiddleware QueryMiddleware) *DBConnWrapper {
	w.queryFunc = queryMiddleware(w.queryFunc)
	return w
}

func (w *DBConnWrapper) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return w.queryFunc(ctx, query, args...)
}

// WithQueryRowMiddleware adds the middleware for the QueryRowContext to the DBConnWrapper
func (w *DBConnWrapper) WithQueryRowMiddleware(queryRowMiddleware QueryRowMiddleware) *DBConnWrapper {
	w.queryRowFunc = queryRowMiddleware(w.queryRowFunc)
	return w
}

func (w *DBConnWrapper) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return w.queryRowFunc(ctx, query, args...)
}

func (w *DBConnWrapper) WithBeginTxMiddleware(beginTxMiddleware BeginTxMiddleware) *DBConnWrapper {
	w.beginTxFunc = beginTxMiddleware(w.beginTxFunc)
	return w
}

func (w *DBConnWrapper) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return w.beginTxFunc(ctx, opts)
}

// Unchanged

func (w *DBConnWrapper) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return w.dbConn.PrepareContext(ctx, query)
}

func (w *DBConnWrapper) Close() error {
	return w.dbConn.Close()
}

func (w *DBConnWrapper) Conn(ctx context.Context) (*sql.Conn, error) {
	return w.dbConn.Conn(ctx)
}

func (w *DBConnWrapper) Driver() driver.Driver {
	return w.dbConn.Driver()
}

func (w *DBConnWrapper) PingContext(ctx context.Context) error {
	return w.dbConn.PingContext(ctx)
}

func (w *DBConnWrapper) SetConnMaxLifetime(d time.Duration) {
	w.dbConn.SetConnMaxLifetime(d)
}

func (w *DBConnWrapper) SetMaxIdleConns(n int) {
	w.dbConn.SetMaxIdleConns(n)
}

func (w *DBConnWrapper) SetMaxOpenConns(n int) {
	w.dbConn.SetMaxOpenConns(n)
}

func (w *DBConnWrapper) Stats() sql.DBStats {
	return w.dbConn.Stats()
}
