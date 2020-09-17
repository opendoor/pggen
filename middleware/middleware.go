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
	/*
		execMiddleware     ExecMiddleware
		queryMiddleware    QueryMiddleware
		queryRowMiddleware QueryRowMiddleware
	*/

	execFunc     ExecFunc
	queryFunc    QueryFunc
	queryRowFunc QueryRowFunc
	beginTxFunc  BeginTxFunc
}

// NewDBConnWrapper wraps the DBConn in struct to which middlewares can be added
func NewDBConnWrapper(dbConn pggen.DBConn) *DBConnWrapper {
	return &DBConnWrapper{
		dbConn: dbConn,
	}
}

// WithExecMiddleware adds the middleware for the ExecContext to the DBConnWrapper
func (w *DBConnWrapper) WithExecMiddleware(execMiddleware ExecMiddleware) *DBConnWrapper {
	execFunc := w.dbConn.ExecContext
	if w.execFunc != nil {
		execFunc = w.execFunc
	}

	w.execFunc = execMiddleware(execFunc)
	return w
}

// ExecContext apply the middleware if any and execute ExecContext on the wrapped DBConn
func (w *DBConnWrapper) ExecContext(ctx context.Context, stmt string, args ...interface{}) (sql.Result, error) {
	if w.execFunc != nil {
		return w.execFunc(ctx, stmt, args...)
	}
	return w.dbConn.ExecContext(ctx, stmt, args...)
}

// WithQueryMiddleware adds the middleware for the QueryContext to the DBConnWrapper
func (w *DBConnWrapper) WithQueryMiddleware(queryMiddleware QueryMiddleware) *DBConnWrapper {
	queryFunc := w.dbConn.QueryContext
	if w.queryFunc != nil {
		queryFunc = w.queryFunc
	}

	w.queryFunc = queryMiddleware(queryFunc)
	return w
}

func (w *DBConnWrapper) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if w.queryFunc != nil {
		return w.queryFunc(ctx, query, args...)
	}

	return w.dbConn.QueryContext(ctx, query, args...)
}

// WithQueryRowMiddleware adds the middleware for the QueryRowContext to the DBConnWrapper
func (w *DBConnWrapper) WithQueryRowMiddleware(queryRowMiddleware QueryRowMiddleware) *DBConnWrapper {
	queryRowFunc := w.dbConn.QueryRowContext
	if w.queryRowFunc != nil {
		queryRowFunc = w.queryRowFunc
	}

	w.queryRowFunc = queryRowMiddleware(queryRowFunc)
	return w
}

func (w *DBConnWrapper) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if w.queryRowFunc != nil {
		return w.queryRowFunc(ctx, query, args...)
	}
	return w.dbConn.QueryRowContext(ctx, query, args...)
}

func (w *DBConnWrapper) WithBeginTxMiddleware(beginTxMiddleware BeginTxMiddleware) *DBConnWrapper {
	beginTxFunc := w.dbConn.BeginTx
	if w.beginTxFunc != nil {
		beginTxFunc = w.beginTxFunc
	}

	w.beginTxFunc = beginTxMiddleware(beginTxFunc)
	return w
}

func (w *DBConnWrapper) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if w.beginTxFunc != nil {
		return w.beginTxFunc(ctx, opts)
	}
	return w.dbConn.BeginTx(ctx, opts)
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
