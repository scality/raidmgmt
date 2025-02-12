package core

import (
	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
)

type HardwareRAIDController struct {
	iface ports.HardwareRAIDController
}

var _ ports.HardwareRAIDController = &HardwareRAIDController{}

// New returns a new RAID controller.
func New(iface ports.HardwareRAIDController) *HardwareRAIDController {
	return &HardwareRAIDController{
		iface: iface,
	}
}

// Controllers returns a list of RAID controllers.
func (r *HardwareRAIDController) Controllers() ([]*raidcontroller.RAIDController, error) {
	controllers, err := r.iface.Controllers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get RAID controllers")
	}

	return controllers, nil
}

// Controller returns a RAID controller for a given metadata.
func (r *HardwareRAIDController) Controller(metadata *raidcontroller.Metadata) (
	*raidcontroller.RAIDController,
	error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, errors.Wrap(err, ErrInvalidRAIDControllerMetadata.Error())
	}

	controller, err := r.iface.Controller(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get RAID controller")
	}

	return controller, nil
}

// PhysicalDrives returns a list of physical drives for a given RAID controller.
func (r *HardwareRAIDController) PhysicalDrives(metadata *raidcontroller.Metadata) (
	[]*physicaldrive.PhysicalDrive,
	error,
) {
	return physicalDrives(r.iface, metadata)
}

// PhysicalDrive returns a physical drive for a given metadata.
func (r *HardwareRAIDController) PhysicalDrive(metadata *physicaldrive.Metadata) (
	*physicaldrive.PhysicalDrive,
	error,
) {
	return physicalDrive(r.iface, metadata)
}

// LogicalVolumes returns a list of logical volumes for a given RAID controller.
func (r *HardwareRAIDController) LogicalVolumes(metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume,
	error,
) {
	return logicalVolumes(r.iface, metadata)
}

// LogicalVolume returns a logical volume for a given metadata.
func (r *HardwareRAIDController) LogicalVolume(metadata *logicalvolume.Metadata) (
	*logicalvolume.LogicalVolume,
	error,
) {
	return logicalVolume(r.iface, metadata)
}

// EnableJBOD enables JBOD mode on a physical drive.
func (r *HardwareRAIDController) EnableJBOD(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	if err := r.iface.EnableJBOD(metadata); err != nil {
		return errors.Wrap(err, "failed to enable JBOD for physical drive")
	}

	return nil
}

// DisableJBOD disables JBOD mode on a physical drive.
func (r *HardwareRAIDController) DisableJBOD(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	if err := r.iface.DisableJBOD(metadata); err != nil {
		return errors.Wrap(err, "failed to disable JBOD for physical drive")
	}

	return nil
}

// SetLVCacheOptions sets cache options on a logical volume.
func (r *HardwareRAIDController) SetLVCacheOptions(
	metadata *logicalvolume.Metadata,
	cacheOpts *logicalvolume.CacheOptions,
) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	if err := cacheOpts.Validate(); err != nil {
		return errors.Wrap(err, "invalid cache options")
	}

	if err := r.iface.SetLVCacheOptions(metadata, cacheOpts); err != nil {
		return errors.Wrap(err, "failed to set cache options for logical volume")
	}

	return nil
}

// CreateLV creates a logical volume from a request.
func (r *HardwareRAIDController) CreateLV(request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume,
	error,
) {
	return createLV(r.iface, request)
}

// AddPDsToLV adds a physical drive to a logical volume.
func (r *HardwareRAIDController) AddPDsToLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	return addPDsToLV(r.iface, lvMetadata, pdsMetadata...)
}

// DeleteLV deletes a logical volume.
func (r *HardwareRAIDController) DeleteLV(metadata *logicalvolume.Metadata) error {
	return deleteLV(r.iface, metadata)
}

// DeletePDsFromLV deletes a physical drive from a logical volume.
func (r *HardwareRAIDController) DeletePDsFromLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	return deletePDsFromLV(r.iface, lvMetadata, pdsMetadata...)
}

// StartBlink starts blinking for a physical drive.
func (r *HardwareRAIDController) StartBlink(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	if err := r.iface.StartBlink(metadata); err != nil {
		return errors.Wrap(err, "failed to start blinking for physical drive")
	}

	return nil
}

// StopBlink stops blinking for a physical drive.
func (r *HardwareRAIDController) StopBlink(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	if err := r.iface.StopBlink(metadata); err != nil {
		return errors.Wrap(err, "failed to stop blinking for physical drive")
	}

	return nil
}
