# Example: id_in_set

This example shows how to write a query that checks for set containment
using the `= ANY` idiom rather than the `IN` operator that is typically
used with other database interactivity libraries.

## Running

Set up the database

```bash
> createdb pggen_example
> psql pggen_example < db.sql
```

edit the `generate.go` file so that the generate line starts with `//go:generate` instead of
`// go:generate`, then generate the code

```bash
> go generate ./...
```

run the program

```bash
> go run ./main.go
```
