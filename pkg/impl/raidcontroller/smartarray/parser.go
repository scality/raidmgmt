package smartarray

import (
	"bytes"
	"regexp"
)

// splitOutput splits the output into blocks based on the regular expression.
// TODO add tests.
func splitOutput(regularExpression *regexp.Regexp, output []byte) [][]byte {
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
