# Example: timestamps

This example shows how to make use of the automatic timestamp features that
pggen provides. You can ask pggen to automatically take care of keeping track
created at, updated at, and soft delete timestamps.

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
