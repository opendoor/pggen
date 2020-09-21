package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/examples/single_results/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("postgres", os.Getenv("DB_URL"))
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

	// we can use this result directly rather than having to unpack it
	// from a singleton slice.
	val, err := pgClient.MyGetFooValue(ctx, foo1ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("val = %s\n", *val)

	// single_result queries will tag their error as not found errors if no
	// results are returned.
	_, err = pgClient.MyGetFooValue(ctx, foo1ID+1)
	if pggen.IsNotFoundError(err) {
		fmt.Println("2nd query: not found")
	} else {
		fmt.Println("2nd query: found")
	}
}
