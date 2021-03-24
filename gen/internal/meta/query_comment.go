// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package meta

// file: query_comment.go
// This file is concerned with convering a text comment from the config file
// into a golang comment.

import (
	"strings"
	"unicode"
)

// configCommentToGoComment converts a comment provided in the config file to
// a golang comment. The rules are:
// - the empty string maps to the empty string
// - if blank, the very first line is dropped before common prefix detection runs
// - if blank, the very last line is dropped before common prefix detection runs
// - all common leading whitespace at the start of each line is stripped
// - the characters `// ` are added to each nonblank line
// - each blank line is converted to `//`
func configCommentToGoComment(comment string) string {
	// short circuit on the empty string
	if comment == "" {
		return ""
	}

	lines := strings.Split(comment, "\n")

	// drop leading/trailing blank line which is often the result of having the quotes
	// for the string be on their own line(s).
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		if len(lines) == 1 {
			lines = []string{}
		} else {
			lines = lines[:len(lines)-1]
		}
	}

	// strip common leading whitespace
	first := true
	common := []rune{}
	for _, line := range lines {
		i := 0
		for _, r := range line {
			if unicode.IsSpace(r) {
				if first {
					common = append(common, r)
				} else if i >= len(common) || r != common[i] {
					// this line has more spaces than the current known common prefix
					// or it has a different type of whitespace
					break
				}
			} else {
				// we are no longer at a space
				break
			}
			i++
		}
		if !first {
			if i < len(common) {
				common = common[:i]
			}
		}
		first = false
	}
	// now that we've found the prefix, we can snip everything
	prefixByteLen := len(string(common))
	for i := range lines {
		lines[i] = lines[i][prefixByteLen:]
	}

	for i := range lines {
		if strings.TrimSpace(lines[i]) == "" {
			lines[i] = "//"
		} else {
			lines[i] = "// " + lines[i]
		}
	}

	return strings.Join(lines, "\n")
}
