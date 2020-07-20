# Example: include_spec

This example shows how to use include specs to tell pggen how to fill in
records which are associated with one another via foreign keys.

## Running

Set up the database

```bash
> createdb pggen_example
> psql pggen_example < db.sql
```

edit the `models/generate.go` file so that the generate line starts with `//go:generate` instead of
`// go:generate`, then generate the code

```bash
> go generate ./...
```

run the program

```bash
> go run ./main.go
```
