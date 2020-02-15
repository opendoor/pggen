package test

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/opendoor-labs/pggen/pggen/test/db_shims"
)

var (
	ctx      context.Context
	pgClient db_shims.PGClient
	dbURL    string
)

func init() {
	dbURL = os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatalf("no DB_URL in the environment")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	pgClient = db_shims.PGClient{DB: db}
	ctx = context.Background()
}

type Expectation struct {
	call     func() (interface{}, error)
	expected string
}

func (e Expectation) test(t *testing.T) {
	actual, err := e.call()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	actualTxt, err := json.Marshal(actual)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	matched, err := regexp.Match(e.expected, actualTxt)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if !matched {
		t.Errorf("\nExpected Regex: %s\nText: %s\n", e.expected, actualTxt)
	}
}

func newTx(t *testing.T) db_shims.PGClient {
	tx, err := pgClient.DB.(*sql.DB).Begin()
	if err != nil {
		t.Fatal(err)
	}
	return db_shims.PGClient{DB: tx}
}

func chkErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

// Grab root of git repo that we are currently in
func getRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out[:len(out)-1]), nil
}
