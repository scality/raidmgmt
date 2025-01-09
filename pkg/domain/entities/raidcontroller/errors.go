package raidcontroller

import "errors"

const prefixErrMetadata = "RAIDController.Metadata.Validate: "

var (
	ErrNil        = errors.New(prefixErrMetadata + "Metadata is nil")
	ErrIDNegative = errors.New(prefixErrMetadata + "Metadata ID is negative")
)
