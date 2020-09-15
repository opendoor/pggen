package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sort"

	_ "github.com/lib/pq"
	"github.com/opendoor-labs/pggen/examples/id_in_set/models"
	"github.com/opendoor-labs/pggen/middleware"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("postgres", os.Getenv("DB_URL"))
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
	foo1ID, err := pgClient.InsertFoo(ctx, &models.Foo{
		Value: &bar,
	})
	if err != nil {
		log.Fatal(err)
	}

	bax := "bax"
	foo2ID, err := pgClient.InsertFoo(ctx, &models.Foo{
		Value: &bax,
	})
	if err != nil {
		log.Fatal(err)
	}

	baz := "baz"
	foo3ID, err := pgClient.InsertFoo(ctx, &models.Foo{
		Value: &baz,
	})
	if err != nil {
		log.Fatal(err)
	}

	lish := "lish"
	_, err = pgClient.UpdateFoo(ctx, &models.Foo{
		Id:    foo1ID,
		Value: &lish,
	}, models.FooAllFields)
	if err != nil {
		log.Fatal(err)
	}

	err = pgClient.DeleteFoo(ctx, foo3ID)
	if err != nil {
		log.Fatal(err)
	}

	values, err := pgClient.GetFooValues(ctx, []int64{foo1ID, foo2ID, foo3ID})
	if err != nil {
		log.Fatal(err)
	}

	// ensure stable output
	sort.Slice(values, func(i, j int) bool {
		return *values[i] < *values[j]
	})

	for _, v := range values {
		fmt.Printf("%s\n", *v)
	}
}
