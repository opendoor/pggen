package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/opendoor/pggen"
	"github.com/opendoor/pggen/examples/upsert/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("pgx", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// make sure the example can be re-run as many times as we want
	_, err = conn.ExecContext(ctx, "TRUNCATE TABLE users")
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	//
	// insert a record using the upsert interface
	//

	// returns the primary key of the record
	id, err := pgClient.UpsertUser(ctx,
		// the actual data to insert
		&models.User{
			Email:  "calvin@whitehouse.gov",
			Slogan: "Stay Cool with Coolage",
			Rating: "Garbage President",
		},
		// A list of columns to look at to detect conflicts with existing data.
		// If left nil like we are doing here, pggen will just use the primary key.
		nil,
		// This tells pggen to update all of the given fields in the event of a
		// conflict. If there is no conflict, they will all be inserted no matter
		// what this field set contains.
		models.UserAllFields,
	)
	if err != nil {
		log.Fatal(err)
	}

	//
	// let's be a little nicer
	//

	// build the set of fields to update in the event of a conflict
	ratingFieldSet := pggen.NewFieldSet(models.UserMaxFieldIndex)
	ratingFieldSet.Set(models.UserRatingFieldIndex, true)

	id2, err := pgClient.UpsertUser(ctx,
		&models.User{
			Email:  "calvin@whitehouse.gov",
			Rating: "Was in the wrong place at the wrong time.",
		},
		[]string{"email"}, // use the email column to detect conflicts
		ratingFieldSet,
	)
	if err != nil {
		log.Fatal(err)
	}
	if id != id2 {
		log.Fatal("the id should not change")
	}

	pres, err := pgClient.GetUser(ctx, id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("slogan:", pres.Slogan)
	fmt.Println("rating:", pres.Rating)
}
