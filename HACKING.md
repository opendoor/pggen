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
