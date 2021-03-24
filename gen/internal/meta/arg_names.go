// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package meta

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// argNamesToSlice parses an arg_names spec and returns an in-order
// slice of the names specified. It is given a number of target args
// and will fill in any missing args with a made up argument.
func argNamesToSlice(argNamesSpec string, targetNargs int) ([]string, error) {
	type entry struct {
		index int
		name  string
	}
	entries := make([]entry, 0, targetNargs)

	// slurp all the args
	pairs := strings.Split(argNamesSpec, " ")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}

		pairParts := strings.SplitN(pair, ":", 2)
		if len(pairParts) != 2 {
			return nil, fmt.Errorf("malformed arg_names spec: expected ':' seperated pair")
		}

		argNum, err := strconv.Atoi(pairParts[0])
		if err != nil {
			return nil, fmt.Errorf("malformed arg_names spec: pairs must start with a number: %s", err.Error())
		}
		if argNum > targetNargs {
			return nil, fmt.Errorf("malformed arg_names spec: %d out of range", argNum)
		}
		entries = append(entries, entry{index: argNum, name: pairParts[1]})
	}

	// put the args in the right order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].index < entries[j].index
	})

	if len(entries) > 0 && entries[0].index <= 0 {
		return nil, fmt.Errorf("malformed arg_names spec: arg numbers start at 1 not %d", entries[0].index)
	}

	// build the return slice, filling in any blanks
	cursor := 0
	args := make([]string, 0, targetNargs)
	for i := 1; i <= targetNargs; i++ {
		if cursor < len(entries) && entries[cursor].index == i {
			args = append(args, entries[cursor].name)
			cursor++
		} else {
			args = append(args, fmt.Sprintf("arg%d", i))
		}
	}

	return args, nil
}
