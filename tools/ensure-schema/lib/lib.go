package lib

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib" // load postgres driver
)

// PopulateDB runs `schemaFilePath` against `$DB_URL`
func PopulateDB(schemaFilePath string) error {
	// read in the schema
	schemaFile, err := os.Open(schemaFilePath)
	if err != nil {
		return fmt.Errorf("populateDB: missing schema file: %s", err.Error())
	}
	defer schemaFile.Close()
	schemaReader := bufio.NewReader(schemaFile)
	schema, err := ioutil.ReadAll(schemaReader)
	if err != nil {
		return fmt.Errorf("populateDB: reading schema file: %s", err.Error())
	}

	// connect to the database
	dbURL, inEnv := os.LookupEnv("DB_URL")
	if !inEnv {
		return fmt.Errorf("populateDB: DB_URL must be present in the environment")
	}
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("populateDB: opening connection to database: %s", err.Error())
	}
	defer db.Close()

	_, err = db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("populateDB: executing schema: %s", err.Error())
	}

	return nil
}
