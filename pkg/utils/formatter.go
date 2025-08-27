package utils //nolint:revive // This is a utility package.

import (
	"bytes"
	"regexp"
	"strings"
)

const keyValueParts = 2

// splitOutput splits the output into blocks based on the regular expression.
// TODO add tests.
func SplitOutput(regularExpression *regexp.Regexp, output []byte) [][]byte {
	indices := regularExpression.FindAllIndex(output, -1)
	if indices == nil {
		return nil // No matches found
	}

	var blocks [][]byte

	start := 0

	for i, match := range indices {
		if i == 0 {
			continue // Skip the first match
		}

		block := output[start:match[0]] // everything before the match
		if len(block) > 0 {             // avoid empty blocks
			blocks = append(blocks, bytes.TrimSpace(block)) // trim space here
		}

		start = match[0] // Start of the next block is the current match
	}
	// Add the last block if any
	if start < len(output) {
		blocks = append(blocks, bytes.TrimSpace(output[start:]))
	}

	return blocks
}

// FIXME Might go in another file
// ParseLineDetail parses a line of the show detail command and returns the key and value.
func ParseLineDetail(line string) (key, value string) {
	if line == "" {
		return "", ""
	}

	splitParts := strings.Split(line, ":")

	if len(splitParts) != keyValueParts {
		return "", ""
	}

	key = strings.TrimSpace(splitParts[0])
	value = strings.TrimSpace(splitParts[1])

	return key, value
}
