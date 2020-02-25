package test

//go:generate go run ../main.go -o db_shims/pggen.gen.go pggen.toml

//go:generate go run ../main.go -o overridden_db_shims/pggen.gen.go overrides.pggen.toml
