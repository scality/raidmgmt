package core

import (
	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
)

type RAIDController struct {
	iface ports.RAIDController
}

var _ ports.RAIDController = &RAIDController{}

// New returns a new RAID controller.
func New(iface ports.RAIDController) *RAIDController {
	return &RAIDController{
		iface: iface,
	}
}

// Controllers returns a list of RAID controllers.
func (r *RAIDController) Controllers() ([]*raidcontroller.RAIDController, error) {
	controllers, err := r.iface.Controllers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get RAID controllers")
	}

	return controllers, nil
}

// Controller returns a RAID controller for a given metadata.
func (r *RAIDController) Controller(metadata *raidcontroller.Metadata) (
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
func (r *RAIDController) PhysicalDrives(metadata *raidcontroller.Metadata) (
	[]*physicaldrive.PhysicalDrive,
	error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, errors.Wrap(err, ErrInvalidRAIDControllerMetadata.Error())
	}

	physicalDrives, err := r.iface.PhysicalDrives(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get physical drives")
	}

	return physicalDrives, nil
}

// PhysicalDrive returns a physical drive for a given metadata.
func (r *RAIDController) PhysicalDrive(metadata *physicaldrive.Metadata) (
	*physicaldrive.PhysicalDrive,
	error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	physicalDrive, err := r.iface.PhysicalDrive(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get physical drive")
	}

	return physicalDrive, nil
}

// LogicalVolumes returns a list of logical volumes for a given RAID controller.
func (r *RAIDController) LogicalVolumes(metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume,
	error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, errors.Wrap(err, ErrInvalidRAIDControllerMetadata.Error())
	}

	logicalVolumes, err := r.iface.LogicalVolumes(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volumes")
	}

	return logicalVolumes, nil
}

// LogicalVolume returns a logical volume for a given metadata.
func (r *RAIDController) LogicalVolume(metadata *logicalvolume.Metadata) (
	*logicalvolume.LogicalVolume,
	error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	logicalVolume, err := r.iface.LogicalVolume(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volume")
	}

	return logicalVolume, nil
}

// EnableJBOD enables JBOD mode on a physical drive.
func (r *RAIDController) EnableJBOD(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	if err := r.iface.EnableJBOD(metadata); err != nil {
		return errors.Wrap(err, "failed to enable JBOD for physical drive")
	}

	return nil
}

// DisableJBOD disables JBOD mode on a physical drive.
func (r *RAIDController) DisableJBOD(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	if err := r.iface.DisableJBOD(metadata); err != nil {
		return errors.Wrap(err, "failed to disable JBOD for physical drive")
	}

	return nil
}

// SetLVCacheOptions sets cache options on a logical volume.
func (r *RAIDController) SetLVCacheOptions(
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
func (r *RAIDController) CreateLV(request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume,
	error,
) {
	if err := request.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid logical volume request")
	}

	newLV, err := r.iface.CreateLV(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logical volume")
	}

	return newLV, nil
}

// AddPDsToLV adds a physical drive to a logical volume.
func (r *RAIDController) AddPDsToLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	if err := lvMetadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	for _, pd := range pdsMetadata {
		if err := pd.Validate(); err != nil {
			return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
		}
	}

	if err := r.iface.AddPDsToLV(lvMetadata, pdsMetadata...); err != nil {
		return errors.Wrap(err, "failed to add physical drive to logical volume")
	}

	return nil
}

// DeleteLV deletes a logical volume.
func (r *RAIDController) DeleteLV(metadata *logicalvolume.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	if err := r.iface.DeleteLV(metadata); err != nil {
		return errors.Wrap(err, "failed to delete logical volume")
	}

	return nil
}

// DeletePDsFromLV deletes a physical drive from a logical volume.
func (r *RAIDController) DeletePDsFromLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	if err := lvMetadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	for _, pd := range pdsMetadata {
		if err := pd.Validate(); err != nil {
			return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
		}
	}

	if err := r.iface.DeletePDsFromLV(lvMetadata, pdsMetadata...); err != nil {
		return errors.Wrap(err, "failed to remove physical drive from logical volume")
	}

	return nil
}

// StartBlink starts blinking for a physical drive.
func (r *RAIDController) StartBlink(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	if err := r.iface.StartBlink(metadata); err != nil {
		return errors.Wrap(err, "failed to start blinking for physical drive")
	}

	return nil
}

// StopBlink stops blinking for a physical drive.
func (r *RAIDController) StopBlink(metadata *physicaldrive.Metadata) error {
	if err := metadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	if err := r.iface.StopBlink(metadata); err != nil {
		return errors.Wrap(err, "failed to stop blinking for physical drive")
	}

	return nil
}
