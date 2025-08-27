package utils //nolint:revive // This is a utility package.

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	KB = 1 << 10
	MB = 1 << 20
	GB = 1 << 30
	TB = 1 << 40
	PB = 1 << 50

	ErrInvalidSizeFormat = "invalid size format: %s"
	ErrInvalidSizeUnit   = "invalid size unit: %s"
)

// mapSize is a map of size units to their respective bytes.
//
//nolint:gochecknoglobals // This map is used to convert size units to bytes.
var mapSize = map[string]uint64{
	"KB": KB,
	"MB": MB,
	"GB": GB,
	"TB": TB,
	"PB": PB,
}

// ConvertSizeBytes converts a size string to bytes.
func ConvertSizeBytes(size string) (uint64, error) {
	splitParts := 2

	sizeSplit := strings.Split(size, " ")

	if len(sizeSplit) != splitParts {
		return 0, errors.Errorf(ErrInvalidSizeFormat, size)
	}

	// Replace comma with dot for compatibility with ParseFloat
	normalized := strings.ReplaceAll(sizeSplit[0], ",", ".")

	// Parse the value
	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse size value")
	}

	sizeUnit := sizeSplit[1]

	unit, ok := mapSize[sizeUnit]
	if !ok {
		return 0, errors.Errorf(ErrInvalidSizeUnit, sizeUnit)
	}

	// Calculate the size in bytes
	return uint64(value * float64(unit)), nil
}
