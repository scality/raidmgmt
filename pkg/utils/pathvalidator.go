package utils

import (
	"os"

	"github.com/pkg/errors"
)

// ValidatePath checks if the path is a file and exists.
func ValidatePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Wrapf(err, "path does not exist: %s", path)
		}

		return errors.Wrap(err, "error getting path info")
	}

	if info.IsDir() {
		return errors.Wrapf(err, "path is a directory: %s", path)
	}

	return nil
}
