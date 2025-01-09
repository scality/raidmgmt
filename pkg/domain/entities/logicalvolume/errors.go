package logicalvolume

import "errors"

const (
	prefixCacheOptionsErr = "CacheOptions.Validate: "
	prefixMetadataErr     = "LogicalVolume.Metadata.Validate: "
	prefixRequestErr      = "LogicalVolume.Request.Validate: "
)

var (
	ErrCacheOptionsNil    = errors.New(prefixCacheOptionsErr + "CacheOptions is nil")
	ErrUnknownReadPolicy  = errors.New(prefixCacheOptionsErr + "ReadPolicy is unknown")
	ErrUnknownWritePolicy = errors.New(prefixCacheOptionsErr + "WritePolicy is unknown")
	ErrUnknownIOPolicy    = errors.New(prefixCacheOptionsErr + "IOPolicy is unknown")

	ErrMetadataNil       = errors.New(prefixMetadataErr + "Metadata is nil")
	ErrControllerMetaNil = errors.New(prefixMetadataErr + "ControllerMetadata is nil")

	ErrRequestNil          = errors.New(prefixRequestErr + "Request is nil")
	ErrUnknownRAIDLevel    = errors.New(prefixRequestErr + "RAIDLevel is unknown")
	ErrEmptyPhysicalDrives = errors.New(prefixRequestErr + "PhysicalDrives is empty")

	ErrNotEnoughPhysicalDrives   = errors.New("not enough physical drives")
	ErrOddNumberOfPhysicalDrives = errors.New("odd number of physical drives")
)
