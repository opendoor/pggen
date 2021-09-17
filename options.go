package pggen

// options.go contains functional options that can be passed to generated code.

type InsertOpt func(opts *InsertOptions)
type InsertOptions struct {
	UsePkey       bool
	DefaultFields FieldSet
	DisableTimestamps bool
}

// InsertDisableTimestamps tells an insert method to not
// set the timestamp fields
func InsertDisableTimestamps(opts *InsertOptions) {
	opts.DisableTimestamps = true
}

// InsertUsePkey tells an insert method to insert the primary key into the database
// rather than let the database compute it automatically from the default value
// as is the default.
func InsertUsePkey(opts *InsertOptions) {
	opts.UsePkey = true
}

// Set the fields that will be generated from the default values stored in the database.
// By default, all field values are inserted based on the provided struct.
// If all fields are specified, only those fields which actually have a default in the
// database are defaulted, other fields are inserted as normal.
func InsertDefaultFields(fieldSet FieldSet) InsertOpt {
	return func(opts *InsertOptions) {
		opts.DefaultFields = fieldSet
	}
}

type UpsertOpt func(opts *UpsertOptions)
type UpsertOptions struct {
	UsePkey       bool
	DefaultFields FieldSet
	DisableTimestamps bool
}

// UpsertDisableTimestamps tells an upsert method to not
// set the timestamp fields
func UpsertDisableTimestamps(opts *UpsertOptions) {
	opts.DisableTimestamps = true
}

// UpsertUsePkey tells an upsert method to insert the primary key into the database
// rather than let the database compute it automatically from the default value
// as is the default.
func UpsertUsePkey(opts *UpsertOptions) {
	opts.UsePkey = true
}

// Set the fields that will be generated from the default values stored in the database.
// By default, all field values are inserted based on the provided struct.
// If all fields are specified, only those fields which actually have a default in the
// database are defaulted, other fields are inserted as normal.
func UpsertDefaultFields(fieldSet FieldSet) UpsertOpt {
	return func(opts *UpsertOptions) {
		opts.DefaultFields = fieldSet
	}
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
	DisableTimestamps bool
}

// UpdateDisableTimestamps tells an update method to not
// set the timestamp fields
func UpdateDisableTimestamps(opts *UpdateOptions) {
	opts.DisableTimestamps = true
}

type IncludeOpt func(opts *IncludeOptions)
type IncludeOptions struct {
}
