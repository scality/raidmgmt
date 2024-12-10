package megaraid

import (
	"encoding/json"
	"fmt"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/domain/ports"
)

var _ ports.RAIDController = &Adapter{}

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

	panic("not implemented")
}

// LogicalVolumes returns all logical volumes for a given controller.
func (a *Adapter) LogicalVolumes(
	metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume, error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLogicalVolumes, err)
	}

	panic("not implemented")
}

// EnableJBOD enables JBOD mode on a physical drive.
func (a *Adapter) EnableJBOD(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrEnableJBOD, err)
	}

	panic("not implemented")
}

// DisableJBOD disables JBOD mode on a physical drive.
func (a *Adapter) DisableJBOD(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrDisableJBOD, err)
	}

	panic("not implemented")
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

	panic("not implemented")
}

// CreateLV creates a logical volume.
func (a *Adapter) CreateLV(
	request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume, error,
) {
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateLV, err)
	}

	panic("not implemented")
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

	panic("not implemented")
}

// DeleteLV deletes a logical volume.
func (a *Adapter) DeleteLV(metadata *logicalvolume.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrDeleteLV, err)
	}

	panic("not implemented")
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

	panic("not implemented")
}

// StartBlink starts the blinking of the given physical drive.
func (a *Adapter) StartBlink(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrStartBlink, err)
	}

	panic("not implemented")
}

// StopBlink stops the blinking of the given physical drive.
func (a *Adapter) StopBlink(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrStopBlink, err)
	}

	panic("not implemented")
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
