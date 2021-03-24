// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gofrs/uuid"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/opendoor-labs/pggen/examples/uuid/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("pgx", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	tok := uuid.Must(uuid.FromString("4dd819b4-bfa3-46fd-ab9d-54fd330d6702"))
	id, err := pgClient.InsertUser(ctx, &models.User{
		Email: "alphonso@yehaw.website",
		Token: tok,
	})
	if err != nil {
		log.Fatal(err)
	}

	res, err := pgClient.GetUser(ctx, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("email:", res.Email)
	fmt.Println("token:", res.Token.String())
}
