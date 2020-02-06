FROM golang:1.12-alpine

# we need postgresql-client so we can set up our database with psql
# for testing and `go` uses `git` to fetch deps for us. musl-dev
# and gcc are needed for cgo support.
RUN apk add --no-cache postgresql-client git musl-dev gcc bash

# install golangci-lint
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.23.3

WORKDIR /pggen

COPY go.sum .
COPY go.mod .
RUN go mod download

# volumes don't work well in circle, so just copy all the source code
# into the image itself.
COPY . ./

CMD bash ./scripts/test.bash
