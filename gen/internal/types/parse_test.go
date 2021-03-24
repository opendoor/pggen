// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package types

import (
	"regexp"
	"testing"
)

func TestPgParseArrayTypeSuccess(t *testing.T) {
	type testCase struct {
		src    string
		result string
	}

	cases := []testCase{
		{
			src: "bigint[]",
		},
		{
			src: "bigint[][]",
		},
		{
			src: "character varying[][][][][]",
		},
	}

	for i, c := range cases {
		s, err := parsePgArray(c.src)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %s", i, err)
		}

		if len(c.result) == 0 {
			c.result = c.src
		}

		after := s.String()
		if after != c.result {
			t.Fatalf(
				"case %d: expected '%s' to become '%s', got '%s'",
				i,
				c.src,
				c.result,
				after,
			)
		}
	}
}

func TestPgParseArrayTypeErrors(t *testing.T) {
	type testCase struct {
		src string
		re  string
	}

	cases := []testCase{
		{
			src: "foos",
			re:  "tried to parse an array, but failed to",
		},
	}

	for i, c := range cases {
		s, err := parsePgArray(c.src)
		if err == nil {
			t.Fatalf("case %d: err == nil", i)
		}
		if s != nil {
			t.Fatalf("case %d: s != nil", i)
		}

		errTxt := err.Error()
		matches, err := regexp.Match(c.re, []byte(errTxt))
		if err != nil {
			t.Fatal(err)
		}
		if !matches {
			t.Fatalf("case %d: /%s/ failed to match '%s'", i, c.re, errTxt)
		}
	}
}
