// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package utils

import (
	"fmt"
	"go/format"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func WriteGoFile(path string, rawSrc []byte) error {
	outFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var src []byte
	if os.Getenv("PGGEN_GOFMT") == "off" {
		src = rawSrc
	} else {
		src, err = format.Source(rawSrc)
		if err != nil {
			return fmt.Errorf("internal pggen error: %s", err.Error())
		}
	}

	return WriteCompletely(outFile, src)
}

func WriteCompletely(w io.Writer, data []byte) error {
	for len(data) > 0 {
		nbytes, err := w.Write(data)
		if err != nil {
			return err
		}
		data = data[nbytes:]
	}
	return nil
}

// dirOf returns the name of the directory that `path` is contained by
func DirOf(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return filepath.Base(filepath.Dir(abs)), nil
}

func RandomName(base string) string {
	return fmt.Sprintf("%s_%d", base, rand.Int63()) // nolint: gosec
}

// nullOutArgs takes a string containing an SQL query and replaces
// all substrings which match the regex `\$[0-9]+` outside of quotes
// with the string "NULL".
func NullOutArgs(query string) string {
	lastChunkEnd := 0
	chunks := []string{}
	quoteRune := 'X'
	argStart := -1

	for i, r := range query {
		if argStart >= 0 {
			if unicode.IsDigit(r) && r <= 127 { // is it an ascii digit
				continue
			}
			if i > argStart+1 { // we have seen at least one digit past the $
				chunks = append(chunks, query[lastChunkEnd:argStart])
				chunks = append(chunks, "NULL")
				lastChunkEnd = i
				argStart = -1
			}
		}

		switch r {
		case '"':
			fallthrough
		case '\'':
			realQuoteChar := true
			if i > 0 && query[i-1] == '\\' {
				realQuoteChar = false
			}

			if realQuoteChar {
				if quoteRune == 'X' {
					quoteRune = r
				} else if r == quoteRune {
					quoteRune = 'X'
				}
			}
		case '$':
			if quoteRune == 'X' {
				argStart = i
			}
		}
	}

	if argStart >= 0 {
		// the very last bit was an escape sequence
		chunks = append(chunks, query[lastChunkEnd:argStart])
		chunks = append(chunks, "NULL")
	} else {
		chunks = append(chunks, query[lastChunkEnd:])
	}

	return strings.Join(chunks, "")
}
