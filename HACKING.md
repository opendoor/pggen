# Development

To hack on `pggen` you will need a postgres database to test against.
Make sure you have postgres installed, then from the root of the repo run

```
> createdb pggen_development
> psql pggen_development < cmd/pggen/test/db.sql
```

You will need to re-run the command loading `cmd/pggen/test/db.sql` any time
you make a change to the test data or schema defined in that file. Once
the test database is set up you can run all tests by doing

```
> go generate ./...
> go test ./...
```

Most of the tests are defined in the `cmd/pggen/test` package, so if you
want to focus on a specific test, you will probably want to look there.

## With Docker

If you want, you can use almost the same docker set up that `pggen` uses for
continuous integration for local development. The setup for local development
differs in that it uses a docker volume so that changes on your local file system
are picked up and tested without needing to re-build the image each time.

First, build the development docker image:

```bash
> docker-compose build
```

once you have the image built, you can either run the tests
start to finish with

```
> docker-compose run test
```

or debug interactively by opening a bash shell with

```
> docker-compose run test bash
```

## CLI Tests

`pggen/test/cli_test.go` contains some end-to-end tests of the `pggen` command
line utility. These tests spawn subprocesses and then perform some assertions
about the stdout, stderr, and return code of the subprocesses. If one of them
fails, it will print the command used to execute the failing test, but you won't
be able to re-run the command because the test will have cleaned up after
itself. You can enable re-running the cli tests with

```bash
> cd pggen/test
> PGGEN_DEBUG_CLI=1 go test --run TestCLI
```

This will both prevent the cli tests from removing their scratch dir and
compile the binary under test in a debugger friendly format so that you can
immediately point your delve at it.

## Errors in the Generated Code

When modifying the output of the codegenerator, you are likely to introduce compile
errors in the generated code. Because we run the `go/format` package over our output
before landing it to a disk you won't be able to debug the issue by looking at the
generated file by default. In order to make this easier, you can modify the `writeGoFile`
routine in `gen/utils.go` to skip the formatting and just dump the code to disk.

```

diff --git a/gen/utils.go b/gen/utils.go
index dd7804d..77a437e 100644
--- a/gen/utils.go
+++ b/gen/utils.go
@@ -2,7 +2,7 @@ package gen

 import (
 	"fmt"
-	"go/format"
+	// "go/format"
 	"io"
 	"math/rand"
 	"os"
@@ -18,10 +18,13 @@ func writeGoFile(path string, src []byte) error {
 	}
 	defer outFile.Close()

+	/*
 	formattedSrc, err := format.Source(src)
 	if err != nil {
 		return fmt.Errorf("internal pggen error: %s", err.Error())
 	}
+	*/
+	formattedSrc := src

 	return writeCompletely(outFile, formattedSrc)
 }
```

Even without formatting it is actually pretty readable.
