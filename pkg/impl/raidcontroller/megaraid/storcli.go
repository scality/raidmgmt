package megaraid

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/domain/ports"
)

var _ ports.RAIDController = &Adapter{}

const (
	GB = 1 << 30
	TB = 1 << 40
	PB = 1 << 50
)

// mapSize is a map of size units to their respective bytes.
var mapSize = map[string]uint64{
	"GB": GB,
	"TB": TB,
	"PB": PB,
}

type Adapter struct {
	cmd Runner
}

func New(cmdRunner Runner) *Adapter {
	return &Adapter{
		cmd: cmdRunner,
	}
}

// Controllers returns a list of RAID controllers.
func (a *Adapter) Controllers() ([]*raidcontroller.RAIDController, error) {
	controllers, err := a.controllers()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrControllers, err)
	}

	return controllers, nil
}

// PhysicalDrives returns all physical drives for a given controller.
func (a *Adapter) PhysicalDrives(
	metadata *raidcontroller.Metadata) (
	[]*physicaldrive.PhysicalDrive, error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPhysicalDrives, err)
	}

	physicalDrives, err := a.physicaldrives(metadata)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPhysicalDrives, err)
	}

	return physicalDrives, nil
}

// LogicalVolumes returns all logical volumes for a given controller.
func (a *Adapter) LogicalVolumes(
	metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume, error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLogicalVolumes, err)
	}

	logicalVolumes, err := a.logicalvolumes(metadata)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLogicalVolumes, err)
	}

	return logicalVolumes, nil
}

// EnableJBOD enables JBOD mode on a physical drive.
func (a *Adapter) EnableJBOD(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrEnableJBOD, err)
	}

	if err := a.enableJBOD(metadata); err != nil {
		return fmt.Errorf("%w: %w", ErrEnableJBOD, err)
	}

	return nil
}

// DisableJBOD disables JBOD mode on a physical drive.
func (a *Adapter) DisableJBOD(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrDisableJBOD, err)
	}

	if err := a.disableJBOD(metadata); err != nil {
		return fmt.Errorf("%w: %w", ErrDisableJBOD, err)
	}

	return nil
}

// SetLVCacheOptions sets cache options on a logical volume.
func (a *Adapter) SetLVCacheOptions(
	metadata *logicalvolume.Metadata,
	cacheOpts *logicalvolume.CacheOptions,
) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrSetLVCacheOptions, err)
	}

	if err := cacheOpts.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrSetLVCacheOptions, err)
	}

	if err := a.setLVCacheOptions(metadata, cacheOpts); err != nil {
		return fmt.Errorf("%w: %w", ErrSetLVCacheOptions, err)
	}

	return nil
}

// CreateLV creates a logical volume.
func (a *Adapter) CreateLV(
	request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume, error,
) {
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateLV, err)
	}

	newLv, err := a.createLV(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateLV, err)
	}

	return newLv, nil
}

// AddPVToLV adds a physical drive to a logical volume.
func (a *Adapter) AddPVToLV(
	lvMetadata *logicalvolume.Metadata,
	pdMetadata *physicaldrive.Metadata,
) error {
	if err := lvMetadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrAddPVToLV, err)
	}

	if err := pdMetadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrAddPVToLV, err)
	}

	if err := a.addPVToLV(lvMetadata, pdMetadata); err != nil {
		return fmt.Errorf("%w: %w", ErrAddPVToLV, err)
	}

	return nil
}

// DeleteLV deletes a logical volume.
func (a *Adapter) DeleteLV(metadata *logicalvolume.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrDeleteLV, err)
	}

	if err := a.deleteLV(metadata); err != nil {
		return fmt.Errorf("%w: %w", ErrDeleteLV, err)
	}

	return nil
}

// DeletePVFromLV deletes a physical drive from a logical volume.
func (a *Adapter) DeletePVFromLV(
	lvMetadata *logicalvolume.Metadata,
	pdMetadata *physicaldrive.Metadata,
) error {
	if err := lvMetadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrDeletePVFromLV, err)
	}

	if err := pdMetadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrDeletePVFromLV, err)
	}

	if err := a.deletePVFromLV(lvMetadata, pdMetadata); err != nil {
		return fmt.Errorf("%w: %w", ErrDeletePVFromLV, err)
	}

	return nil
}

// StartBlink starts the blinking of the given physical drive.
func (a *Adapter) StartBlink(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrStartBlink, err)
	}

	if err := a.startBlink(metadata); err != nil {
		return fmt.Errorf("%w: %w", ErrStartBlink, err)
	}

	return nil
}

// StopBlink stops the blinking of the given physical drive.
func (a *Adapter) StopBlink(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrStopBlink, err)
	}

	if err := a.stopBlink(metadata); err != nil {
		return fmt.Errorf("%w: %w", ErrStopBlink, err)
	}

	return nil
}

// unmarshalToSlice unmarshals a JSON response data to a slice.
func unmarshalToSlice[T any](responseData json.RawMessage, key string) ([]T, error) {
	data, found := searchForKey(responseData, key)
	if !found {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}

	var slice []T

	if err := json.Unmarshal(data, &slice); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshal, key)
	}

	return slice, nil
}

// unmashalToPointer unmarshals a JSON response data to a pointer.
func unmarshalToPointer[T any](responseData json.RawMessage, key string) (*T, error) {
	data, found := searchForKey(responseData, key)
	if !found {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}

	var t T

	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshal, key)
	}

	return &t, nil
}

// convertSizeBytes converts a size string to bytes.
func convertSizeBytes(size string) (uint64, error) {
	sizeSplit := strings.Split(size, " ")
	if len(sizeSplit) != 2 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidSizeFormat, size)
	}

	// Replace comma with dot for compatibility with ParseFloat
	normalized := strings.ReplaceAll(sizeSplit[0], ",", ".")

	// Parse the value
	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidSizeValue, sizeSplit[0])
	}

	sizeUnit := sizeSplit[1]

	unit, ok := mapSize[sizeUnit]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrInvalidSizeUnit, sizeUnit)
	}

	// Calculate the size in bytes
	return uint64(value * float64(unit)), nil
}
