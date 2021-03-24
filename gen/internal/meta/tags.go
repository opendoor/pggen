// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package meta

import (
	"fmt"
	"strconv"
	"strings"
)

// mergeTags combines two strings containing golang struct tags
//
// The rules are:
//    - gorm tags get combined into a single tag with semicolon seperated values
//    - only the first occurence of a duplicate tag is kept (unless that tag is 'gorm').
//      This means that that this function is not communitive because tags in t1 take
//      precedence over tags in t2.
//    - orthoginal tags are joined with spaces
//    - if exactly one tag string does not parse, it is just appened to the end with a seperating
//      space
//    - if both tag strings do not parse, mergeTags returns "$t1 $t2"
func mergeTags(t1 string, t2 string) string {
	// combine all the tag pairs into a single list
	pairs, err1 := parseTags(t1)
	pairs2, err2 := parseTags(t2)
	if err1 == nil && err2 == nil {
		pairs = append(pairs, pairs2...)
	} else if err2 == nil {
		pairs = pairs2
	}

	// combine gorm tags, seperating with ';'
	var gormValue strings.Builder
	seenGormTag := false
	for _, pair := range pairs {
		if pair.key == "gorm" {
			if seenGormTag {
				gormValue.WriteRune(';')
			}
			gormValue.WriteString(pair.value)
			seenGormTag = true
		}
	}

	// filter out duplicates and 'gorm' tags, keeping only the first
	seen := map[string]bool{}
	cursor := 0
	for _, pair := range pairs {
		if !(pair.key == "gorm" || seen[pair.key]) {
			pairs[cursor] = pair
			cursor++
		}
		seen[pair.key] = true
	}
	pairs = pairs[:cursor]

	// assemble the result
	writtenTag := false
	var res strings.Builder
	gormValueStr := gormValue.String()
	if gormValueStr != "" {
		res.WriteString(`gorm:`)
		res.WriteString(strconv.Quote(gormValueStr))
		writtenTag = true
	}
	for _, pair := range pairs {
		if writtenTag {
			res.WriteRune(' ')
		}
		res.WriteString(pair.key)
		res.WriteRune(':')
		res.WriteString(strconv.Quote(pair.value))
		writtenTag = true
	}

	if err1 != nil {
		if writtenTag {
			res.WriteRune(' ')
		}
		res.WriteString(t1)
		writtenTag = true
	}
	if err2 != nil {
		if writtenTag {
			res.WriteRune(' ')
		}
		res.WriteString(t2)
	}

	return res.String()
}

type tagPair struct {
	key   string
	value string
}

// parseTags splits a conventially formated golang struct tag into key value pairs
//
// This is a modified version of the reflect.StructTag.Lookup routine. Internal
// to this file.
func parseTags(tag string) (pairs []tagPair, err error) {
	pairs = []tagPair{}
	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			return nil, fmt.Errorf("incomplete tag")
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			return nil, fmt.Errorf("unclosed quoted value")
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		value, err := strconv.Unquote(qvalue)
		if err != nil {
			return nil, fmt.Errorf("unquoting string: %s", err.Error())
		}
		pairs = append(pairs, tagPair{
			key:   name,
			value: value,
		})
	}

	return pairs, nil
}
