// package jsontypes defines some types to (de)serialize to for the json_values
// table
package jsontypes

type SomeData struct {
	Foo string `json:"foo"`
	Bar *int   `json:"bar"`
}
