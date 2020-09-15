package middleware_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/opendoor-labs/pggen/middleware"
	"github.com/opendoor-labs/pggen/middleware/internal/mocks"
)

func TestExecMiddleware(t *testing.T) {
	mockDBConn := &mocks.DBConn{}
	expectedCtx := context.Background()
	expectedQuery := "myQuery"
	expectedArgs := []interface{}{1, "a"}
	expectedResult := &mocks.Result{}

	beforeExecDone := false
	afterExecDone := false
	testExecMiddleware := func(execFunc middleware.ExecFunc) middleware.ExecFunc {
		return func(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
			if !reflect.DeepEqual(ctx, expectedCtx) {
				t.Fatalf("context do not match, expected: %v, got: %v", expectedCtx, ctx)
			}
			if query != expectedQuery {
				t.Fatalf("query does not match, expected: %s, got: %s", expectedQuery, query)
			}
			if !reflect.DeepEqual(args, expectedArgs) {
				t.Fatalf("args does not match, expected: %v, got: %v", expectedArgs, args)
			}
			beforeExecDone = true
			result, err := execFunc(ctx, query, args...)
			if err != nil {
				t.Fatalf("exec function returned an error: %v", err)
			}
			if result != expectedResult {
				t.Fatalf("result does not match, expected: %v, got: %v", expectedResult, result)
			}
			afterExecDone = true
			return result, err
		}
	}
	connWrapper := middleware.NewDBConnWrapper(mockDBConn).WithExecMiddleware(testExecMiddleware)

	mockDBConn.ExecContextReturns(expectedResult, nil)

	result, err := connWrapper.ExecContext(expectedCtx, expectedQuery, expectedArgs...)
	if err != nil {
		t.Fatalf("exec function returned an error: %v", err)
	}
	if result != expectedResult {
		t.Fatalf("result does not match, expected: %v, got: %v", expectedResult, result)
	}

	if !beforeExecDone {
		t.Fatalf("before execution did not happen")
	}
	if !afterExecDone {
		t.Fatalf("after execution did not happen")
	}
}

func TestQueryMiddleware(t *testing.T) {
	mockDBConn := &mocks.DBConn{}
	expectedCtx := context.Background()
	expectedQuery := "myQuery"
	expectedArgs := []interface{}{1, "a"}
	expectedRows := &sql.Rows{}

	beforeQueryDone := false
	afterQueryDone := false
	testQueryMiddleware := func(queryFunc middleware.QueryFunc) middleware.QueryFunc {
		return func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
			if !reflect.DeepEqual(ctx, expectedCtx) {
				t.Fatalf("context do not match, expected: %v, got: %v", expectedCtx, ctx)
			}
			if query != expectedQuery {
				t.Fatalf("query does not match, expected: %s, got: %s", expectedQuery, query)
			}
			if !reflect.DeepEqual(args, expectedArgs) {
				t.Fatalf("args does not match, expected: %v, got: %v", expectedArgs, args)
			}
			beforeQueryDone = true
			rows, err := queryFunc(ctx, query, args...)
			if err != nil {
				t.Fatalf("exec function returned an error: %v", err)
			}
			if rows != expectedRows {
				t.Fatalf("rows do not match, expected: %v, got: %v", expectedRows, rows)
			}
			afterQueryDone = true
			return rows, err
		}
	}
	connWrapper := middleware.NewDBConnWrapper(mockDBConn).WithQueryMiddleware(testQueryMiddleware)

	mockDBConn.QueryContextReturns(expectedRows, nil)

	rows, err := connWrapper.QueryContext(expectedCtx, expectedQuery, expectedArgs...)
	if err != nil {
		t.Fatalf("exec function returned an error: %v", err)
	}
	if rows != expectedRows {
		t.Fatalf("rows do not match, expected: %v, got: %v", expectedRows, rows)
	}

	if !beforeQueryDone {
		t.Fatalf("before execution did not happen")
	}
	if !afterQueryDone {
		t.Fatalf("after execution did not happen")
	}
}

func TestQueryRowMiddleware(t *testing.T) {
	mockDBConn := &mocks.DBConn{}
	expectedCtx := context.Background()
	expectedQuery := "myQuery"
	expectedArgs := []interface{}{1, "a"}
	expectedRow := &sql.Row{}

	beforeQueryRowDone := false
	afterQueryRowDone := false
	testQueryRowMiddleware := func(queryRowFunc middleware.QueryRowFunc) middleware.QueryRowFunc {
		return func(ctx context.Context, query string, args ...interface{}) *sql.Row {
			if !reflect.DeepEqual(ctx, expectedCtx) {
				t.Fatalf("context do not match, expected: %v, got: %v", expectedCtx, ctx)
			}
			if query != expectedQuery {
				t.Fatalf("query does not match, expected: %s, got: %s", expectedQuery, query)
			}
			if !reflect.DeepEqual(args, expectedArgs) {
				t.Fatalf("args does not match, expected: %v, got: %v", expectedArgs, args)
			}
			beforeQueryRowDone = true
			row := queryRowFunc(ctx, query, args...)
			if row != expectedRow {
				t.Fatalf("row does not match, expected: %v, got: %v", expectedRow, row)
			}
			afterQueryRowDone = true
			return row
		}
	}
	connWrapper := middleware.NewDBConnWrapper(mockDBConn).WithQueryRowMiddleware(testQueryRowMiddleware)

	mockDBConn.QueryRowContextReturns(expectedRow)

	row := connWrapper.QueryRowContext(expectedCtx, expectedQuery, expectedArgs...)
	if row != expectedRow {
		t.Fatalf("row does not match, expected: %v, got: %v", expectedRow, row)
	}

	if !beforeQueryRowDone {
		t.Fatalf("before execution did not happen")
	}
	if !afterQueryRowDone {
		t.Fatalf("after execution did not happen")
	}
}
