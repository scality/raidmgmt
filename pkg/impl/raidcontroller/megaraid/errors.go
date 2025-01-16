package megaraid

import "github.com/pkg/errors"

var (
	ErrInvalidRAIDControllerMetadata = errors.New("invalid RAID controller metadata")
	ErrInvalidPhysicalDriveMetadata  = errors.New("invalid physical drive metadata")
	ErrInvalidLogicalVolumeMetadata  = errors.New("invalid logical volume metadata")

	ErrCommandFailed = errors.New("command failed")
)

const (
	ErrUnrecognizedCacheOptions = "unrecognized cache options: %s"
	ErrUnavailableDrives        = "unavailable drives: %s"
	ErrInvalidEnclosureID       = "invalid enclosure ID: %s"
	ErrInvalidBayID             = "invalid bay ID: %s"
)
