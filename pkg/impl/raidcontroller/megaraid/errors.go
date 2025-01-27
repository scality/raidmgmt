package megaraid

import "github.com/pkg/errors"

const (
	ErrUnrecognizedCacheOptions = "unrecognized cache options: %s"
	ErrInvalidEnclosureID       = "invalid enclosure ID: %s"
	ErrInvalidBayID             = "invalid bay ID: %s"
)

var ErrCommandFailed = errors.New("command failed")
