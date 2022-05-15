package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/opendoor/pggen/examples/boxed_values/models"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("pgx", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	ids, err := pgClient.BulkInsertUser(ctx, []models.User{
		{ Nickname: "Jim", Email: "jim@gmail.com" },
		{ Nickname: "Bill", Email: "bill@gmail.com" },
		{ Nickname: "Stacy", Email: "stacy@yahoo.com" },
	})
	if err != nil {
		log.Fatal(err)
	}

	users, err := pgClient.ListUser(ctx, ids)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("type of user result: %T\n", users[0])
	fmt.Println("name 1:", users[0].Nickname)

	res, err := pgClient.GetUsersFromGmail(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("type of user result: %T\n", res[0])

	fmt.Println("users from gmail:", len(res))
	fmt.Println("name 1:", res[0].Nickname)
	fmt.Println("name 2:", res[1].Nickname)
}
