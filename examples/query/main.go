// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/opendoor-labs/pggen/examples/query/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("pgx", os.Getenv("DB_URL"))
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

	res, err := pgClient.GetUserNicknameAndEmail(ctx, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("name:", res[0].Nickname)
	fmt.Println("email:", *res[0].Email)

	user, err := pgClient.MyGetUser(ctx, id)
	if err != nil {
		log.Fatal(err)
	}

	// Note that even though this query will always return one result, pggen still
	// returns a list of results. For a trick to make this eaiser, check out the
	// `single_results` example.
	fmt.Println("user.name:", user[0].Nickname)
	fmt.Println("user.email:", user[0].Email)
}
