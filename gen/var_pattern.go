package gen

import (
	"os"
	"strings"
)

func anyVarPatternMatches(patterns []string) bool {
	for _, p := range patterns {
		eqIdx := strings.Index(p, "=")
		if eqIdx == -1 {
			_, inEnv := os.LookupEnv(p)
			if inEnv {
				return true
			}
		} else {
			expectedValue := p[eqIdx+1:]
			value := os.Getenv(p[:eqIdx])
			if value == expectedValue {
				return true
			}
		}
	}

	return false
}
