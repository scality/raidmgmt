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

// mapSize maps size unit labels to their multiplier in bytes.
//
// All supported RAID CLIs report binary quantities. ssacli and megaraid print
// decimal-style labels (KB/MB/GB/TB/PB) for binary values — e.g. ssacli "800 GB"
// is 858993459200 bytes (= 800 GiB, matching the lsblk byte count) and megaraid
// "16.370 TB" is 17999005346693 bytes (= 16.370 TiB). storcli2 prints proper IEC
// labels (KiB/MiB/GiB/TiB/PiB), e.g. "9.094 TiB". Both label families therefore
// map to the same 1024-based multipliers; this is intentional, not a typo.
//
//nolint:gochecknoglobals // This map is used to convert size units to bytes.
var mapSize = map[string]uint64{
	"KB": KB,
	"MB": MB,
	"GB": GB,
	"TB": TB,
	"PB": PB,

	"KiB": KB,
	"MiB": MB,
	"GiB": GB,
	"TiB": TB,
	"PiB": PB,
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
