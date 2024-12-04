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
)
