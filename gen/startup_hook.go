package gen

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// run the given command
func runStartupHook(hook string) error {
	if len(hook) == 0 {
		return nil
	}

	cmdArgs, err := parseCmdLine(hook)
	if err != nil {
		return err
	}
	if len(cmdArgs) < 1 {
		return fmt.Errorf("startup hook must contain a command")
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...) // nolint: gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// parse a string into an array of command line arguments, respecting
// ' and " quoting as well as \ escapes
func parseCmdLine(cmd string) ([]string, error) {
	if len(strings.TrimSpace(cmd)) == 0 {
		return nil, fmt.Errorf("blank cmd string")
	}

	chunks := []string{}

	inChunk := false
	quoteChar := 'X'
	chunk := strings.Builder{}
	for i, r := range cmd {
		if !inChunk {
			if quoteChar == 'X' {
				if (i == 0 || cmd[i-1] != '\\') && isQuoteRune(r) {
					quoteChar = r
					continue
				}
			} else {
				if r == quoteChar && (i == 0 || cmd[i-1] != '\\') {
					chunks = append(chunks, chunk.String())
					chunk.Reset()
					quoteChar = 'X'
				} else if !(r == '\\' && i+1 < len(cmd) && cmd[i+1] == byte(quoteChar)) {
					chunk.WriteRune(r)
				}

				// we are in a quote, so skip the standard chunking
				// logic below
				continue
			}
		}

		if r == ' ' && (i == 0 || cmd[i-1] != '\\') {
			if inChunk {
				chunks = append(chunks, chunk.String())
				chunk.Reset()
				inChunk = false
			}
			continue
		} else if !inChunk {
			inChunk = true
		}

		if !(r == '\\' && i+1 < len(cmd) && (isQuoteByte(cmd[i+1]) || cmd[i+1] == ' ')) {
			chunk.WriteRune(r)
		}
	}

	if quoteChar != 'X' {
		return nil, fmt.Errorf(
			"unmatched quote char: %s", string([]rune{quoteChar}))
	}

	if inChunk {
		chunks = append(chunks, chunk.String())
	}

	return chunks, nil
}

func isQuoteRune(r rune) bool {
	return r == '"' || r == '\''
}

func isQuoteByte(b byte) bool {
	return b == '"' || b == '\''
}
