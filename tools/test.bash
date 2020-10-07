#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

if [[ ! -x "${DB_URL+x}" ]] ; then
    export DB_URL="postgres://postgres:test@${DB_HOST}/postgres?sslmode=disable"
fi

if [[ "${DB_HOST+x}" == "x" ]] ; then
    # Wait until `postgres` starts accepting connections. It seems really
    # silly that we need to do this.
    ticks=0
    while ! echo exit | nc "${DB_HOST}" 5432
    do
        echo "failed to connect to postgres trying again in 5 seconds"
        sleep 5

        ticks=$((ticks + 1))
        if (( $ticks > 30 ))
        then
            echo "timed out after $ticks ticks"
            exit 1
        fi
    done

    # If the database already exists, don't bring the script down.
    createdb -h "${DB_HOST}" -W test -U postgres -w -e pggen_test 2>/dev/null || /bin/true
fi

go generate ./...

psql "$DB_URL" < cmd/pggen/test/db.sql

if [[ -n "${LINT+x}" ]] ; then
    golangci-lint run -E gofmt -E gosec -E gocyclo -E deadcode
elif go version | grep '1.13' 2>&1 >/dev/null ; then
    # for some reason the race detector acts weird with go 1.13
    go test -p 1 ./...
    DB_DRIVER=pgx go test -p 1 ./cmd/pggen/test # re-run using jackc/pgx as the driver
else
    # We have to serialize the tests because the example tests will re-write the database
    # schema dynamically. We could fix this by creating a dedicated database for the example tests.
    go test -race -p 1 ./...
    DB_DRIVER=pgx go test -race -p 1 ./cmd/pggen/test # re-run using jackc/pgx as the driver
fi
