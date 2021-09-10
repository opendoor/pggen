package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/opendoor/pggen"
	"github.com/opendoor/pggen/examples/timestamps/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("pgx", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	//
	// add our linear family to the database
	//

	alexID, err := pgClient.InsertUser(ctx, &models.User{
		Email: "alex@macedonia.gov",
	})
	if err != nil {
		log.Fatal(err)
	}
	alex, err := pgClient.GetUser(ctx, alexID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("alex.CreatedAt = %s\n", alex.CreatedAt.Format(time.RFC3339))
	fmt.Printf("alex.UpdatedAt = %s\n", alex.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("alex.DeletedAt = %p\n", alex.DeletedAt)

	err = pgClient.DeleteUser(ctx, alex.Id)
	if err != nil {
		log.Fatal(err)
	}

	_, err = pgClient.GetUser(ctx, alex.Id)
	if pggen.IsNotFoundError(err) {
		fmt.Printf("Alex not found.\n")
	} else {
		log.Fatal(err)
	}

	secretAlex, err := pgClient.GetUserAnyway(ctx, alex.Id)
	if err != nil {
		log.Fatal(err)
	}
	if len(secretAlex) != 1 {
		log.Fatal("could not find alex")
	}

	fmt.Printf("secretAlex.CreatedAt = %s\n", secretAlex[0].CreatedAt.Format(time.RFC3339))
	fmt.Printf("secretAlex.UpdatedAt = %s\n", secretAlex[0].UpdatedAt.Format(time.RFC3339))
	fmt.Printf("secretAlex.DeletedAt = %s\n", secretAlex[0].DeletedAt.Format(time.RFC3339))
}
