package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sort"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/opendoor-labs/pggen/examples/id_in_set/models"
	"github.com/opendoor-labs/pggen/middleware"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("pgx", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	execLoggingMiddleware := func(execFunc middleware.ExecFunc) middleware.ExecFunc {
		return func(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
			fmt.Printf("ExecContext query: %s\n", query)
			result, err := execFunc(ctx, query, args...)
			return result, err
		}
	}

	queryLoggingMiddleware := func(queryFunc middleware.QueryFunc) middleware.QueryFunc {
		return func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
			fmt.Printf("QueryContext query: %s\n", query)
			result, err := queryFunc(ctx, query, args...)
			return result, err
		}
	}

	queryRowLoggingMiddleware := func(queryRowFunc middleware.QueryRowFunc) middleware.QueryRowFunc {
		return func(ctx context.Context, query string, args ...interface{}) *sql.Row {
			fmt.Printf("QueryRowContext query: %s\n", query)
			result := queryRowFunc(ctx, query, args...)
			return result
		}
	}

	wrappedConn := middleware.NewDBConnWrapper(conn)
	wrappedConn = wrappedConn.WithExecMiddleware(execLoggingMiddleware)
	wrappedConn = wrappedConn.WithQueryMiddleware(queryLoggingMiddleware)
	wrappedConn = wrappedConn.WithQueryRowMiddleware(queryRowLoggingMiddleware)

	pgClient := models.NewPGClient(wrappedConn)

	bar := "bar"
	// InsertFoo will be intercepted by the QueryMiddleware
	foo1ID, err := pgClient.InsertFoo(ctx, &models.Foo{
		Value: &bar,
	})
	if err != nil {
		log.Fatal(err)
	}

	bax := "bax"
	// InsertFoo will be intercepted by the QueryMiddleware
	foo2ID, err := pgClient.InsertFoo(ctx, &models.Foo{
		Value: &bax,
	})
	if err != nil {
		log.Fatal(err)
	}

	baz := "baz"
	// InsertFoo will be intercepted by the QueryMiddleware
	foo3ID, err := pgClient.InsertFoo(ctx, &models.Foo{
		Value: &baz,
	})
	if err != nil {
		log.Fatal(err)
	}

	lish := "lish"
	// UpdateFoo will be intercepted by the QueryRowMiddleware
	_, err = pgClient.UpdateFoo(ctx, &models.Foo{
		Id:    foo1ID,
		Value: &lish,
	}, models.FooAllFields)
	if err != nil {
		log.Fatal(err)
	}

	// DeleteFoo will be intercepted by the ExecMiddleware
	err = pgClient.DeleteFoo(ctx, foo3ID)
	if err != nil {
		log.Fatal(err)
	}

	foos, err := pgClient.ListFoo(ctx, []int64{foo1ID, foo2ID})
	if err != nil {
		log.Fatal(err)
	}

	// ensure stable output
	sort.Slice(foos, func(i, j int) bool {
		return *foos[i].Value < *foos[j].Value
	})

	for _, foo := range foos {
		fmt.Printf("%s\n", *foo.Value)
	}
}
