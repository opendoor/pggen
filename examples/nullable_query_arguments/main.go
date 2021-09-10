package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/opendoor/pggen/examples/nullable_query_arguments/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("pgx", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	_, err = pgClient.InsertUser(ctx, &models.User{
		Email:    "alphonso@yehaw.website",
		Nickname: nil,
	})
	if err != nil {
		log.Fatal(err)
	}

	res, err := pgClient.GetUsersByNullableNickname(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("name:", res[0].Nickname)
	fmt.Println("email:", res[0].Email)
}
