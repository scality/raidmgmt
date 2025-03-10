package megaraid

import (
	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
)

var _ ports.RAIDController = &Adapter{}

type Adapter struct {
	runner Runner
}

func New(runner Runner) *Adapter {
	return &Adapter{
		runner: runner,
	}
}

// Controllers returns a list of RAID controllers.
func (a *Adapter) Controllers() ([]*raidcontroller.RAIDController, error) {
	controllers, err := a.controllers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get controllers")
	}

	return controllers, nil
}

// PhysicalDrives returns all physical drives for a given controller.
func (a *Adapter) PhysicalDrives(
	metadata *raidcontroller.Metadata) (
	[]*physicaldrive.PhysicalDrive, error,
) {
	physicalDrives, err := a.physicaldrives(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get physical drives")
	}

	return physicalDrives, nil
}

// LogicalVolumes returns all logical volumes for a given controller.
func (a *Adapter) LogicalVolumes(
	metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume, error,
) {
	logicalVolumes, err := a.logicalvolumes(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volumes")
	}

	return logicalVolumes, nil
}

// EnableJBOD enables JBOD mode on a physical drive.
func (a *Adapter) EnableJBOD(metadata *physicaldrive.Metadata) error {
	if err := a.setJBOD(metadata, "set"); err != nil {
		return errors.Wrap(err, "failed to enable JBOD")
	}

	return nil
}

// DisableJBOD disables JBOD mode on a physical drive.
func (a *Adapter) DisableJBOD(metadata *physicaldrive.Metadata) error {
	if err := a.setJBOD(metadata, "delete"); err != nil {
		return errors.Wrap(err, "failed to disable JBOD")
	}

	return nil
}

// SetLVCacheOptions sets cache options on a logical volume.
func (a *Adapter) SetLVCacheOptions(
	metadata *logicalvolume.Metadata,
	cacheOpts *logicalvolume.CacheOptions,
) error {
	if err := a.setLVCacheOptions(metadata, cacheOpts); err != nil {
		return errors.Wrap(err, "failed to set cache options")
	}

	return nil
}

// CreateLV creates a logical volume.
func (a *Adapter) CreateLV(
	request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume, error,
) {
	newLv, err := a.createLV(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logical volume")
	}

	return newLv, nil
}

// AddPDToLV adds a physical drive to a logical volume.
func (a *Adapter) AddPDsToLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	if err := a.migrate("add", lvMetadata, pdsMetadata...); err != nil {
		return errors.Wrap(err, "failed to add physical drive to logical volume")
	}

	return nil
}

// DeleteLV deletes a logical volume.
func (a *Adapter) DeleteLV(metadata *logicalvolume.Metadata) error {
	if err := a.deleteLV(metadata); err != nil {
		return errors.Wrap(err, "failed to delete logical volume")
	}

	return nil
}

// DeletePDFromLV deletes a physical drive from a logical volume.
func (a *Adapter) DeletePDsFromLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	if err := a.migrate("remove", lvMetadata, pdsMetadata...); err != nil {
		return errors.Wrap(err, "failed to delete physical drive from logical volume")
	}

	return nil
}

// StartBlink starts the blinking of the given physical drive.
func (a *Adapter) StartBlink(metadata *physicaldrive.Metadata) error {
	if err := a.blink(metadata, "start"); err != nil {
		return errors.Wrap(err, "failed to start blinking")
	}

	return nil
}

// StopBlink stops the blinking of the given physical drive.
func (a *Adapter) StopBlink(metadata *physicaldrive.Metadata) error {
	if err := a.blink(metadata, "stop"); err != nil {
		return errors.Wrap(err, "failed to stop blinking")
	}

	return nil
}

// Controller returns a RAID controller for a given metadata.
func (a *Adapter) Controller(metadata *raidcontroller.Metadata) (
	*raidcontroller.RAIDController, error,
) {
	controller, err := a.controller(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get controller %d", metadata.ID)
	}

	return controller, nil
}

// PhysicalDrive returns a physical drive for a given metadata.
func (a *Adapter) PhysicalDrive(metadata *physicaldrive.Metadata) (
	*physicaldrive.PhysicalDrive,
	error,
) {
	pd, err := a.physicalDrive(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get physical drive %s", metadata.Slot.String())
	}

	return pd, nil
}

// LogicalVolume returns a logical volume for a given metadata.
func (a *Adapter) LogicalVolume(metadata *logicalvolume.Metadata) (
	*logicalvolume.LogicalVolume,
	error,
) {
	lv, err := a.logicalVolume(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get logical volume %s", metadata.ID)
	}

	return lv, nil
}
