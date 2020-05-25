package models

//go:generate go run ../../../cmd/pggen/main.go -o models.gen.go -c postgres://localhost/pggen_example?sslmode=disable pggen.toml
