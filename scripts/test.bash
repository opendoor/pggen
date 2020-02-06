#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

# Wait until `postgres` start accepting connections. It seems really
# silly that we need to do this.
ticks=0
while ! echo exit | nc postgres 5432
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
createdb -h postgres -U postgres -w -e pggen_test || /bin/true

psql $DB_URL < pggen/test/db.sql

golangci-lint run -E gofmt -E gosec -E gocyclo -E deadcode

go generate ./...

go test ./...
