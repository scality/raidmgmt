package raidcontroller

import "errors"

const prefixErrMetadata = "RAIDController.Metadata.Validate: "

var (
	ErrParsingFailed = errors.New("failed to parse RAID controller ID as integer")
	ErrNil           = errors.New(prefixErrMetadata + "Metadata is nil")
	ErrIDEmpty       = errors.New(prefixErrMetadata + "Metadata ID is empty")
	ErrIDNegative    = errors.New(prefixErrMetadata + "Metadata ID is negative")
)
