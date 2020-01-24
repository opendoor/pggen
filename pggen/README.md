# pggen

Package `"github.com/opendoor-labs/pggen/pggen"` contains the command line
tool for invoking the `pggen` library. It generates type safe SQL
database call shims based on the objects stored in a postgres database.
This allows you to define the schema for your database objects only once,
and in the language most natural for working with relational data: SQL.

See `$CODE/go/lib/pggen/README.md` for more details on the features
and configuration for pggen.

## TODO

There are a few more features remaining before `pggen` is ready for the
prime time. Here is a list in rough order of the priority that I think
they have.

1. Add support for array arguments to queries
2. Spend some effort making sure that `pggen` is well documented
3. Split the output into multiple files
4. Write some tests invoking the pggen program at the command line
    to make sure we reasonably report errors and whatnot.
5. If a foreign key is constrained to be UNIQUE, infer a `has_one` relationship
   instead of a `has_many` relationship.
6. Recognize join tables as indicating a `many_to_many` relationship. Expose a
   configuration option to allow people to explicitly specify join tables (if
   they have additional fields in the join tables).

## Alternatives

`xo` and `gnorm` are both existing "database first" code generation tools for
go, but they both seem to be pretty lightly maintained.

I've spent the most time examining xo because it seems like a more mature
solution.

Some issues I found with xo:
  - For stored functions, it generates verbose return type names without
    actually defining the return type. You can override the return type
    name manually, but the lack of actually defining the return types
    is pretty sad.
  - When operating in query introspection mode, it generates return types
    which can't properly handle nulls (nullable strings get represented
    with `string` instead of `sql.NullString` or `*string` for example).
    This is a pretty big show stopper in my opinion.
  - It doesn't know how to infer argument types for queries (you have to
    explicitly provide them using a baroque xo-specific syntax instead
    of being able to use a postgres-native query).
  - It uses command line flags rather than a single configuration file
    for configuration, which I find more confusing.

