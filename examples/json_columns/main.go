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
	"github.com/opendoor-labs/pggen/examples/json_columns/config"
	"github.com/opendoor-labs/pggen/examples/json_columns/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("pgx", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	id, err := pgClient.InsertUser(ctx, &models.User{
		Email: "jonny@pielovers.net",
		Bio: models.UserBio{
			Name:        "Jonny Jet",
			FavoritePie: "All of them!",
		},
		Config: config.Config{
			HomepageIsPublic: false,
			Deactivated:      true,
		},
		Homepage: []byte(`{"status": "under construction"}`),
	})
	if err != nil {
		log.Fatal(err)
	}
	user, err := pgClient.GetUser(ctx, id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("name =", user.Bio.Name)
	fmt.Println("deactivated =", user.Config.Deactivated)
	fmt.Println("homepage =", string(user.Homepage))
}
