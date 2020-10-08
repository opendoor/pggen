package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/examples/statement/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	id, err := pgClient.InsertUser(ctx, &models.User{
		Email:    "alphonso@yehaw.website",
		Nickname: "Alph",
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = pgClient.DeleteUsersByNickname(ctx, "Alph")
	if err != nil {
		log.Fatal(err)
	}

	_, err = pgClient.GetUser(ctx, id)
	if err == nil {
		log.Fatal("Alph is unexpectedly still in the db")
	}
	if pggen.IsNotFoundError(err) {
		fmt.Printf("Alph not found\n")
	}
}
