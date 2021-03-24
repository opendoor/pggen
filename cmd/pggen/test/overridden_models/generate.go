// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package overridden_models

// make sure that the schema is in place
//go:generate go run ../../../../tools/ensure-schema/main.go ../db.sql

//go:generate go run ../../main.go -o models.gen.go pggen.toml
