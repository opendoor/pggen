package models

// make sure that the schema is in place
//go:generate go run ../../../tools/ensure-schema/main.go ../db.sql

//go:generate go run ../../../cmd/pggen/main.go -o models.gen.go -c $DB_URL pggen.toml
