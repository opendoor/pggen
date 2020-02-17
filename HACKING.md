# Development

To hack on `pggen` you will need a postgres database to test against.
Make sure you have postgres installed, then from the root of the repo run

```
> createdb pggen_development
> psql pggen_development < pggen/test/db.sql
```

You will need to re-run the command loading `pggen/test/db.sql` any time
you make a change to the test data or schema defined in that file. Once
the test database is set up you can run all tests by doing

```
> go generate ./...
> go test ./...
```

Most of the tests are defined in the `pggen/test` package, so if you
want to focus on a specific test, you will probably want to look there.

## With Docker

If you want, you can use almost the same docker set up that `pggen` uses for
continuous integration. First, build the development docker image:

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
> docker-compose run test /bin/bash
```

# CLI Tests

`pggen/test/cli_test.go` contains some end-to-end tests of the `pggen` command
line utility. These tests spawn subprocesses and then perform some assertions
about the stdout, stderr, and return code of the subprocesses. If one of them
fails, it will print the command used to execute the failing test, but you won't
be able to re-run the command because the test will have cleaned up after
itself. You can re-run the cli tests with

```bash
> cd pggen/test
> PGGEN_DEBUG_CLI=1 go test --run TestCLI
```

This will both prevent the cli tests from removing their scratch dir and
compile the binary under test in a debugger friendly format so that you can
immediately point your debugger at it.
