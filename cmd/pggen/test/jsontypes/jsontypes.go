// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
// package jsontypes defines some types to (de)serialize to for the json_values
// table
package jsontypes

type SomeData struct {
	Foo string `json:"foo"`
	Bar *int   `json:"bar"`
}
