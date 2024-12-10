package physicaldrive

import "errors"

const prefixErr = "PhysicalDrive.Metadata.Validate: "

var (
	ErrNil         = errors.New(prefixErr + "Metadata is nil")
	ErrCtrlMetaNil = errors.New(prefixErr + "ControllerMetadata is nil")
)
