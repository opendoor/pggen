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

// DBConnWrapper is a wrapper around DBConn that also contain the middlewares to apply when doing the DB calls
type DBConnWrapper struct {
	dbConn             pggen.DBConn
	execMiddleware     ExecMiddleware
	queryMiddleware    QueryMiddleware
	queryRowMiddleware QueryRowMiddleware
}

// NewDBConnWrapper wraps the DBConn in struct to which middlewares can be added
func NewDBConnWrapper(dbConn pggen.DBConn) *DBConnWrapper {
	return &DBConnWrapper{
		dbConn: dbConn,
	}
}

// WithExecMiddleware adds the middleware for the ExecContext to the DBConnWrapper
func (dbConnWrapper *DBConnWrapper) WithExecMiddleware(execMiddleware ExecMiddleware) *DBConnWrapper {
	dbConnWrapper.execMiddleware = execMiddleware
	return dbConnWrapper
}

// ExecContext apply the middleware if any and execute ExecContext on the wrapped DBConn
func (dbConnWrapper *DBConnWrapper) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	execFunc := dbConnWrapper.dbConn.ExecContext
	if dbConnWrapper.execMiddleware != nil {
		execFunc = dbConnWrapper.execMiddleware(execFunc)
	}
	return execFunc(ctx, query, args...)
}

// WithQueryMiddleware adds the middleware for the QueryContext to the DBConnWrapper
func (dbConnWrapper *DBConnWrapper) WithQueryMiddleware(queryMiddleware QueryMiddleware) *DBConnWrapper {
	dbConnWrapper.queryMiddleware = queryMiddleware
	return dbConnWrapper
}

func (dbConnWrapper *DBConnWrapper) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	queryFunc := dbConnWrapper.dbConn.QueryContext
	if dbConnWrapper.queryMiddleware != nil {
		queryFunc = dbConnWrapper.queryMiddleware(queryFunc)
	}
	return queryFunc(ctx, query, args...)
}

// WithQueryRowMiddleware adds the middleware for the QueryRowContext to the DBConnWrapper
func (dbConnWrapper *DBConnWrapper) WithQueryRowMiddleware(queryRowMiddleware QueryRowMiddleware) *DBConnWrapper {
	dbConnWrapper.queryRowMiddleware = queryRowMiddleware
	return dbConnWrapper
}

func (dbConnWrapper *DBConnWrapper) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	queryRowFunc := dbConnWrapper.dbConn.QueryRowContext
	if dbConnWrapper.queryRowMiddleware != nil {
		queryRowFunc = dbConnWrapper.queryRowMiddleware(queryRowFunc)
	}
	return queryRowFunc(ctx, query, args...)
}

// Unchanged

func (dbConnWrapper *DBConnWrapper) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return dbConnWrapper.dbConn.PrepareContext(ctx, query)
}

func (dbConnWrapper *DBConnWrapper) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return dbConnWrapper.dbConn.BeginTx(ctx, opts)
}

func (dbConnWrapper *DBConnWrapper) Close() error {
	return dbConnWrapper.dbConn.Close()
}

func (dbConnWrapper *DBConnWrapper) Conn(ctx context.Context) (*sql.Conn, error) {
	return dbConnWrapper.dbConn.Conn(ctx)
}

func (dbConnWrapper *DBConnWrapper) Driver() driver.Driver {
	return dbConnWrapper.dbConn.Driver()
}

func (dbConnWrapper *DBConnWrapper) PingContext(ctx context.Context) error {
	return dbConnWrapper.dbConn.PingContext(ctx)
}

func (dbConnWrapper *DBConnWrapper) SetConnMaxLifetime(d time.Duration) {
	dbConnWrapper.dbConn.SetConnMaxLifetime(d)
}

func (dbConnWrapper *DBConnWrapper) SetMaxIdleConns(n int) {
	dbConnWrapper.dbConn.SetMaxIdleConns(n)
}

func (dbConnWrapper *DBConnWrapper) SetMaxOpenConns(n int) {
	dbConnWrapper.dbConn.SetMaxOpenConns(n)
}

func (dbConnWrapper *DBConnWrapper) Stats() sql.DBStats {
	return dbConnWrapper.dbConn.Stats()
}
