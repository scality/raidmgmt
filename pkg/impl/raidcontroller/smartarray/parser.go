package smartarray

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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

//nolint:gocognit,cyclop // Parser functions are complicated by essence.
func ParseLSBLKOutput(output []byte) ([]BlockDevice, error) {
	lines := strings.Split(string(output), "\n")
	//nolint:mnd // No need for a constant here.
	if len(lines) < 2 { // Check if there's at least a header and one device
		return nil, nil
	}

	header := strings.Fields(lines[0]) // Split the header line
	devices := []BlockDevice{}

	for _, line := range lines[1:] { // Skip the header line
		line = strings.TrimSpace(line) // remove leading/trailing spaces
		if line == "" {                // skip empty lines
			continue
		}

		fields := strings.Fields(line)

		device := BlockDevice{}

		for i, field := range fields {
			if header[i] != "" && len(header) > i {
				switch header[i] {
				case "NAME":
					device.DevicePath = field
				case "SIZE":
					size, err := strconv.ParseUint(field, 10, 64)
					if err != nil {
						return nil, errors.Wrap(err, "failed to parse size")
					}

					device.Size = size
				case "ROTA":
					device.Rotational = field
				case "TYPE":
					device.Type = field
				case "TRAN":
					device.Tran = field
				case "MOUNTPOINT":
					device.MountPoint = field
				case "FSTYPE":
					device.FilesystemType = field
				case "PARTTYPE":
					device.PartitionType = field
				}
			}
		}

		// Skip non-disk and non-part devices
		if device.Type != "disk" && device.Type != "part" {
			continue
		}

		devices = append(devices, device)
	}

	return devices, nil
}
