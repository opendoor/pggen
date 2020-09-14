package middleware_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/opendoor-labs/pggen/middleware"
	"github.com/opendoor-labs/pggen/testing/mocks"
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
			result, err := execFunc(ctx, query, args)
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
