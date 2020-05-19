package pggen

// options.go contains functional options that can be passed to generated code.

type InsertOpt func(opts *InsertOptions)
type InsertOptions struct {
	UsePkey bool
}

// UsePkey tells an insert method to insert the primary key into the database
// rather than let the database compute it automatically from the default value
// as is the default.
func UsePkey(opts *InsertOptions) {
	opts.UsePkey = true
}
