// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package gen

import (
	"os"
	"strings"
)

func anyVarPatternMatches(patterns []string) bool {
	for _, p := range patterns {
		if varPatternMatches(p) {
			return true
		}
	}

	return false
}

func allVarPatternsMatch(patterns []string) bool {
	for _, p := range patterns {
		if !varPatternMatches(p) {
			return false
		}
	}

	return true
}

func varPatternMatches(pattern string) bool {
	eqIdx := strings.Index(pattern, "=")
	if eqIdx == -1 {
		_, inEnv := os.LookupEnv(pattern)
		if inEnv {
			return true
		}
	} else {
		expectedValue := pattern[eqIdx+1:]
		value := os.Getenv(pattern[:eqIdx])
		if value == expectedValue {
			return true
		}
	}

	return false
}
