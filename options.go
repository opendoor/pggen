package pggen

// options.go contains functional options that can be passed to generated code.

type InsertOpt func(opts *InsertOptions)
type InsertOptions struct {
	UsePkey bool
}

// InsertUsePkey tells an insert method to insert the primary key into the database
// rather than let the database compute it automatically from the default value
// as is the default.
func InsertUsePkey(opts *InsertOptions) {
	opts.UsePkey = true
}

type UpsertOpt func(opts *UpsertOptions)
type UpsertOptions struct {
	UsePkey bool
}

// UpsertUsePkey tells an upsert method to insert the primary key into the database
// rather than let the database compute it automatically from the default value
// as is the default.
func UpsertUsePkey(opts *UpsertOptions) {
	opts.UsePkey = true
}

type GetOpt func(opts *GetOptions)
type GetOptions struct {
}

type ListOpt func(opts *ListOptions)
type ListOptions struct {
}

type DeleteOpt func(opts *DeleteOptions)
type DeleteOptions struct {
	DoHardDelete bool
}

// DeleteDoHardDelete tells a delete method to delete the data from the database
// even if a `deleted_at` timestamp has been configured for soft deletes. If soft
// deletes have not been configured for the table (via the `deleted_at_field` config
// key), this flag has no effect.
func DeleteDoHardDelete(opts *DeleteOptions) {
	opts.DoHardDelete = true
}

type UpdateOpt func(opts *UpdateOptions)
type UpdateOptions struct {
}

type IncludeOpt func(opts *IncludeOptions)
type IncludeOptions struct {
}
