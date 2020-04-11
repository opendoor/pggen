#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

if [[ -n "${LINT+x}" ]] ; then
    echo "LINTING"
    golangci-lint run -E gofmt -E gosec -E gocyclo -E deadcode
    exit 0
fi

export DB_URL="postgres://postgres:test@${DB_HOST}/postgres?sslmode=disable"

# Wait until `postgres` start accepting connections. It seems really
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
createdb -h "${DB_HOST}" -W test -U postgres -w -e pggen_test || /bin/true

psql "$DB_URL" < cmd/pggen/test/db.sql

go generate ./...

go test ./...
