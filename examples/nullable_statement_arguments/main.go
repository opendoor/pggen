package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/opendoor/pggen"
	"github.com/opendoor/pggen/examples/nullable_statement_arguments/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("pgx", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	nick := "Alph"
	id, err := pgClient.InsertUser(ctx, &models.User{
		Email:    "alphonso@yehaw.website",
		Nickname: nick,
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = pgClient.DeleteUsersByNickname(ctx, &nick)
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
