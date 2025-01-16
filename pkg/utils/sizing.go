package utils

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	GB = 1 << 30
	TB = 1 << 40
	PB = 1 << 50
)

var (
	// mapSize is a map of size units to their respective bytes.
	mapSize = map[string]uint64{
		"GB": GB,
		"TB": TB,
		"PB": PB,
	}
)

// ConvertSizeBytes converts a size string to bytes.
func ConvertSizeBytes(size string) (uint64, error) {
	sizeSplit := strings.Split(size, " ")
	if len(sizeSplit) != 2 {
		return 0, fmt.Errorf("invalid size format: %s", size)
	}

	// Replace comma with dot for compatibility with ParseFloat
	normalized := strings.ReplaceAll(sizeSplit[0], ",", ".")

	// Parse the value
	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size value: %w", err)
	}

	sizeUnit := sizeSplit[1]

	unit, ok := mapSize[sizeUnit]
	if !ok {
		return 0, fmt.Errorf("invalid size unit: %s", sizeUnit)
	}

	// Calculate the size in bytes
	return uint64(value * float64(unit)), nil
}
