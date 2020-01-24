package test_internal

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func SetupDatabase(dbURL string) {
	// HACK: the "unit_test" database isn't always set up for us,
	// so we will give setting it up the old college try
	adminDB, err := sql.Open(
		"postgres",
		"postgres://postgres:@postgres:5432/postgres?sslmode=disable",
	)
	defer adminDB.Close()
	if err != nil {
		fmt.Printf(
			"WARN: while opening admin connection: %s\n",
			err.Error(),
		)
	}
	_, err = adminDB.Exec("CREATE DATABASE unit_test")
	if err != nil {
		fmt.Printf("WARN: while creating db: %s\n", err.Error())
	}

	goPath := os.Getenv("GOPATH")
	dbSetupFile, err := os.Open(path.Join(
		goPath, "src", "github.com", "opendoor-labs", "code",
		"go", "tools", "pggen", "test", "db.sql"))
	if err != nil {
		log.Fatal(err)
	}
	setTestDBCmds, err := ioutil.ReadAll(dbSetupFile)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("while opening connection: %s", err.Error())
	}
	defer db.Close()

	_, err = db.Exec(string(setTestDBCmds))
	if err != nil {
		log.Fatalf("while executing setup code: %s", err.Error())
	}
}
