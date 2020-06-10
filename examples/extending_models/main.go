package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/opendoor-labs/pggen/examples/extending_models/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("postgres", "postgres://localhost/pggen_example?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	chihuahuaID, err := pgClient.InsertDog(ctx, &models.Dog{
		Breed:         "chihuahua",
		Size:          models.SizeCategorySmall,
		AgeInDogYears: 38,
	})
	if err != nil {
		log.Fatal(err)
	}
	chihuahua, err := pgClient.GetDog(ctx, chihuahuaID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("a %s says '%s'\n", chihuahua.Breed, chihuahua.Bark())

	wolfHound := &models.Dog{
		Breed:         "irish wolf hound",
		Size:          models.SizeCategoryLarge,
		AgeInDogYears: 17,
	}
	fmt.Printf("an %s says '%s'\n", wolfHound.Breed, wolfHound.Bark())
}
