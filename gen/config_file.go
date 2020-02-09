package gen

// The configuration file format used to specify the database objects
// to generate code for.
type dbConfig struct {
	StoredFunctions []storedFuncConfig `toml:"stored_function"`
	Queries         []queryConfig      `toml:"query"`
	Stmts           []stmtConfig       `toml:"statement"`
	Tables          []tableConfig      `toml:"table"`
}

// Stored functions are a special case of queries. The main advantage
// they have over queries is that the names of the arguments to the
// generated function will be better, as they are derived from the argument
// names in postgres rather than being `arg0`, `arg1`...
type storedFuncConfig struct {
	// The name of the stored function in postgres
	Name string `toml:"name"`
	// See the field of the same name on `queryConfig`
	NullFlags string `toml:"null_flags"`
	// See the field of the same name on `queryConfig`
	NotNullFields []string `toml:"not_null_fields"`
	// See the field of the same name on `queryConfig`
	ReturnType string `toml:"return_type"`
}

// Queries registered in the config file represent arbitrary bits of
// SQL, possibly parameterized by $N arguments. The generated code
// will use `sql.QueryContext` and marshal the results into a list of
// rows returned.
type queryConfig struct {
	// The name that should be used to identify this query in generated go
	// code.
	Name string `toml:"name"`
	// The actual text of the query.
	Body string `toml:"body"`
	// A string consisting of the runes '-' and 'n' to indicate the
	// nullability of return columns. '-' indicates that the column is
	// not nullable (NOT NULL), while 'n' indicates that it is nullable.
	// These need to be specified manually because postgres does not expose
	// a mechanism for infering the nullability of query results that I
	// could discover. The flags string must be exactly as long as the
	// result set is wide.
	NullFlags string `toml:"null_flags"`
	// A long-form way of specifying the same thing as `NullFlags`. Only one
	// of the two options should be provided. Any fields appearing in this list
	// will be treated as not nullable, with all other fields being considered
	// nullable as is the default.
	NotNullFields []string `toml:"not_null_fields"`
	// The name that should be used for this query's return type.
	// This is useful because it allows multiple queries to return
	// values of the same type so that client code does not have to
	// perform a series of endless conversions. If two queries which
	// return different types are given the same name to use for their
	// return type, it is an error.
	ReturnType string `toml:"return_type"`
}

// Statements are like queries but they are executed for side effects
// and therefore return `(sql.Result, error)` rather than a set of
// rows. Statements should be used for INSERT, UPDATE, and DELETE
// operations.
type stmtConfig struct {
	// The name that should be used to identify this statement in generated
	// go code.
	Name string `toml:"name"`
	// The actual text of this statement.
	Body string `toml:"body"`
}

type tableConfig struct {
	// The name of the table in the database
	Name string `toml:"name"`
	// If true, pggen will not infer a relationship between this table
	// and any owning tables based on any foreign keys in this table.
	NoInferBelongsTo bool `toml:"no_infer_belongs_to"`
	// A list of tables that this table belongs to
	BelongsTo []belongsTo `toml:"belongs_to"`
}

// An explicitly configured foreign key relationship which can be attached
// to a table's config.
type belongsTo struct {
	// The table that this table belongs to
	Table string `toml:"table"`
	// The name of the foreign key which points to the table this table
	// belongs to.
	KeyField string `toml:"key_field"`
	// If true the owning table has at most one of this table
	OneToOne bool `toml:"one_to_one"`
}
