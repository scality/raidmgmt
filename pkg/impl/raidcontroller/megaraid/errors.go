package megaraid

import "errors"

var (
	ErrControllers       = errors.New("controllers listing failed")
	ErrPhysicalDrives    = errors.New("physical drives listing failed")
	ErrLogicalVolumes    = errors.New("logical volumes listing failed")
	ErrEnableJBOD        = errors.New("enable JBOD failed")
	ErrDisableJBOD       = errors.New("disable JBOD failed")
	ErrSetLVCacheOptions = errors.New("set logical volume cache options failed")
	ErrCreateLV          = errors.New("create logical volume failed")
	ErrAddPVToLV         = errors.New("add physical drive to logical volume failed")
	ErrDeleteLV          = errors.New("delete logical volume failed")
	ErrDeletePVFromLV    = errors.New("delete physical drive from logical volume failed")
	ErrStartBlink        = errors.New("start blinking failed")
	ErrStopBlink         = errors.New("stop blinking failed")

	ErrCommandFailed = errors.New("command failed")
	ErrUnmarshal     = errors.New("unmarshal failed")
	ErrKeyNotFound   = errors.New("key not found")

	ErrNoControllersFound = errors.New("no controllers found")
	ErrControllerNotFound = errors.New("controller not found")

	ErrInvalidSizeFormat = errors.New("invalid size format")
	ErrInvalidSizeUnit   = errors.New("invalid size unit")
	ErrInvalidSizeValue  = errors.New("invalid size value")

	ErrInvalidAction = errors.New("invalid action")

	ErrLogicalVolumeNotFound = errors.New("logical volume not found")

	ErrCacheOptionsNotProvided = errors.New("cache options not provided")
	ErrNoCacheOptionsToUpdate  = errors.New("no cache options to update")

	ErrRaid0RequiresAtLeast1Drive = errors.New("RAID 0 requires at least 1 physical drives")
	ErrRaid1Requires2Drives       = errors.New("RAID 1 requires exactly 2 physical drives")
	ErrRaid10RequiresAtLeast4     = errors.New("RAID 10 requires at least 4 physical drives")
	ErrInvalidRAIDLevel           = errors.New("invalid RAID level")

	ErrMatchPhysicalDrives = errors.New("match physical drives failed")

	ErrInvalidEnclosureID = errors.New("invalid enclosure ID")
	ErrInvalidSlotID      = errors.New("invalid slot ID")

	ErrPhysicalDriveNotAvailable      = errors.New("physical drive not available")
	ErrPhysicalDriveNotFound          = errors.New("physical drive not found")
	ErrMultipleNewLogicalVolumes      = errors.New("more than one new logical volume ID found")
	ErrMultipleEnclosuresNotSupported = errors.New("multiple enclosures not supported")

	ErrNewLogicalVolumeNotFound = errors.New("new logical volume not found")
)
