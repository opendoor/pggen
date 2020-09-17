package test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/middleware"
	// "github.com/opendoor-labs/pggen/cmd/pggen/test/models"
)

func TestExecMiddleware(t *testing.T) {
	ctx := context.Background()

	dbConn := pgClient.Handle().(pggen.DBConn)
	expectedCtx := context.Background()
	stmt := "INSERT INTO middleware_test_recs (value) VALUES ($1)"
	args := []interface{}{"foo"}

	called := false
	testExecMiddleware := func(execFunc middleware.ExecFunc) middleware.ExecFunc {
		return func(ctx context.Context, actualStmt string, actualArgs ...interface{}) (sql.Result, error) {
			if !reflect.DeepEqual(ctx, expectedCtx) {
				t.Fatalf("context do not match, expected: %v, got: %v", expectedCtx, ctx)
			}
			if actualStmt != stmt {
				t.Fatalf("query does not match, expected: %s, got: %s", stmt, actualStmt)
			}
			if !reflect.DeepEqual(actualArgs, args) {
				t.Fatalf("args does not match, expected: %v, got: %v", args, actualArgs)
			}
			result, err := execFunc(ctx, actualStmt, actualArgs...)
			if err != nil {
				t.Fatalf("function returned an error: %v", err)
			}
			called = true
			return result, err
		}
	}
	wrappedDBConn := middleware.NewDBConnWrapper(dbConn).WithExecMiddleware(testExecMiddleware)

	_, err := wrappedDBConn.ExecContext(ctx, stmt, args...)
	chkErr(t, err)
	if !called {
		t.Fatal("not called")
	}
}

func TestQueryMiddleware(t *testing.T) {
	ctx := context.Background()

	dbConn := pgClient.Handle().(pggen.DBConn)
	expectedCtx := context.Background()
	query := "SELECT * FROM middleware_test_recs WHERE value = $1"
	args := []interface{}{"foo"}

	called := false
	testQueryMiddleware := func(queryFunc middleware.QueryFunc) middleware.QueryFunc {
		return func(ctx context.Context, actualQuery string, actualArgs ...interface{}) (*sql.Rows, error) {
			if !reflect.DeepEqual(ctx, expectedCtx) {
				t.Fatalf("context do not match, expected: %v, got: %v", expectedCtx, ctx)
			}
			if actualQuery != query {
				t.Fatalf("query does not match, expected: %s, got: %s", query, actualQuery)
			}
			if !reflect.DeepEqual(actualArgs, args) {
				t.Fatalf("args does not match, expected: %v, got: %v", args, actualArgs)
			}
			result, err := queryFunc(ctx, actualQuery, actualArgs...)
			if err != nil {
				t.Fatalf("function returned an error: %v", err)
			}
			called = true
			return result, err
		}
	}
	wrappedDBConn := middleware.NewDBConnWrapper(dbConn).WithQueryMiddleware(testQueryMiddleware)

	_, err := wrappedDBConn.QueryContext(ctx, query, args...)
	chkErr(t, err)
	if !called {
		t.Fatal("not called")
	}
}

func TestQueryRowMiddleware(t *testing.T) {
	ctx := context.Background()

	dbConn := pgClient.Handle().(pggen.DBConn)
	expectedCtx := context.Background()
	query := "SELECT * FROM middleware_test_recs WHERE value = $1 LIMIT 1"
	args := []interface{}{"foo"}

	called := false
	testQueryRowMiddleware := func(queryRowFunc middleware.QueryRowFunc) middleware.QueryRowFunc {
		return func(ctx context.Context, actualQuery string, actualArgs ...interface{}) *sql.Row {
			if !reflect.DeepEqual(ctx, expectedCtx) {
				t.Fatalf("context do not match, expected: %v, got: %v", expectedCtx, ctx)
			}
			if actualQuery != query {
				t.Fatalf("query does not match, expected: %s, got: %s", query, actualQuery)
			}
			if !reflect.DeepEqual(actualArgs, args) {
				t.Fatalf("args does not match, expected: %v, got: %v", args, actualArgs)
			}
			result := queryRowFunc(ctx, actualQuery, actualArgs...)
			called = true
			return result
		}
	}
	wrappedDBConn := middleware.NewDBConnWrapper(dbConn).WithQueryRowMiddleware(testQueryRowMiddleware)

	wrappedDBConn.QueryRowContext(ctx, query, args...)
	if !called {
		t.Fatal("not called")
	}
}

func TestBeginTxMiddleware(t *testing.T) {
	ctx := context.Background()

	dbConn := pgClient.Handle().(pggen.DBConn)
	expectedCtx := context.Background()
	opts := &sql.TxOptions{ReadOnly: true}

	called := false
	testBeginTxMiddleware := func(beginTxFunc middleware.BeginTxFunc) middleware.BeginTxFunc {
		return func(ctx context.Context, actualOpts *sql.TxOptions) (*sql.Tx, error) {
			if !reflect.DeepEqual(ctx, expectedCtx) {
				t.Fatalf("context do not match, expected: %v, got: %v", expectedCtx, ctx)
			}
			if !reflect.DeepEqual(actualOpts, opts) {
				t.Fatalf("query does not match, expected: %v, got: %v", opts, actualOpts)
			}
			result, err := beginTxFunc(ctx, opts)
			called = true
			return result, err
		}
	}
	wrappedDBConn := middleware.NewDBConnWrapper(dbConn).WithBeginTxMiddleware(testBeginTxMiddleware)

	tx, err := wrappedDBConn.BeginTx(ctx, opts)
	chkErr(t, err)
	defer tx.Commit() // nolint: errcheck
	if !called {
		t.Fatal("not called")
	}
}
