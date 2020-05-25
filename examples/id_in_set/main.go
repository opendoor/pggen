package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"

	_ "github.com/lib/pq"
	"github.com/opendoor-labs/pggen/examples/id_in_set/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("postgres", "postgres://localhost/pggen_example?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	bar := "bar"
	foo1ID, err := pgClient.InsertFoo(ctx, &models.Foo{
		Value: &bar,
	})
	if err != nil {
		log.Fatal(err)
	}

	baz := "baz"
	foo2ID, err := pgClient.InsertFoo(ctx, &models.Foo{
		Value: &baz,
	})
	if err != nil {
		log.Fatal(err)
	}

	values, err := pgClient.GetFooValues(ctx, []int64{foo1ID, foo2ID})
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
