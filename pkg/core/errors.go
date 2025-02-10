package core

import "github.com/pkg/errors"

var (
	ErrInvalidRAIDControllerMetadata = errors.New("invalid RAID controller metadata")
	ErrInvalidPhysicalDriveMetadata  = errors.New("invalid physical drive metadata")
	ErrInvalidLogicalVolumeMetadata  = errors.New("invalid logical volume metadata")

	ErrInvalidLogicalVolumeRequest = errors.New("invalid logical volume request")
)
