package core

import (
	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
)

type SoftwareRAIDController struct {
	iface ports.SoftwareRAIDController
}

var _ ports.SoftwareRAIDController = &SoftwareRAIDController{}

// NewSoftwareRAIDController returns a new software RAID controller.
func NewSoftwareRAIDController(iface ports.SoftwareRAIDController) *SoftwareRAIDController {
	return &SoftwareRAIDController{
		iface: iface,
	}
}

// PhysicalDrives returns a list of physical drives.
// Controller metadata aren't used as this is software RAID.
func (s *SoftwareRAIDController) PhysicalDrives(
	metadata *raidcontroller.Metadata,
) ([]*physicaldrive.PhysicalDrive, error) {
	err := metadata.Validate()
	if err != nil {
		return nil, errors.Wrap(err, ErrInvalidRAIDControllerMetadata.Error())
	}

	physicalDrives, err := s.iface.PhysicalDrives(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get physical drives")
	}

	return physicalDrives, nil
}

// PhysicalDrive returns a physical drive.
// Controller metadata aren't used as this is software RAID.
func (s *SoftwareRAIDController) PhysicalDrive(
	metadata *physicaldrive.Metadata,
) (*physicaldrive.PhysicalDrive, error) {
	err := metadata.Validate()
	if err != nil {
		return nil, errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	physicalDrive, err := s.iface.PhysicalDrive(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get physical drive")
	}

	return physicalDrive, nil
}

// LogicalVolumes returns a list of logical volumes.
// Controller metadata aren't used as this is software RAID.
func (s *SoftwareRAIDController) LogicalVolumes(
	metadata *raidcontroller.Metadata,
) ([]*logicalvolume.LogicalVolume, error) {
	err := metadata.Validate()
	if err != nil {
		return nil, errors.Wrap(err, ErrInvalidRAIDControllerMetadata.Error())
	}

	logicalVolumes, err := s.iface.LogicalVolumes(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volumes")
	}

	return logicalVolumes, nil
}

// LogicalVolume returns a logical volume.
// Controller metadata aren't used as this is software RAID.
func (s *SoftwareRAIDController) LogicalVolume(
	metadata *logicalvolume.Metadata,
) (*logicalvolume.LogicalVolume, error) {
	err := metadata.Validate()
	if err != nil {
		return nil, errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	logicalVolume, err := s.iface.LogicalVolume(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volume")
	}

	return logicalVolume, nil
}

// CreateLV creates a logical volume.
func (s *SoftwareRAIDController) CreateLV(
	request *logicalvolume.Request,
) (*logicalvolume.LogicalVolume, error) {
	err := request.Validate()
	if err != nil {
		return nil, errors.Wrap(err, ErrInvalidLogicalVolumeRequest.Error())
	}

	logicalVolume, err := s.iface.CreateLV(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logical volume")
	}

	return logicalVolume, nil
}

// DeleteLV deletes a logical volume.
func (s *SoftwareRAIDController) DeleteLV(
	metadata *logicalvolume.Metadata,
) error {
	err := metadata.Validate()
	if err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	err = s.iface.DeleteLV(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to delete logical volume")
	}

	return nil
}

// AddPDsToLV adds physical drives to a logical volume.
func (s *SoftwareRAIDController) AddPDsToLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	err := lvMetadata.Validate()
	if err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	for _, pdMetadata := range pdsMetadata {
		err := pdMetadata.Validate()
		if err != nil {
			return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
		}
	}

	err = s.iface.AddPDsToLV(lvMetadata, pdsMetadata...)
	if err != nil {
		return errors.Wrap(err, "failed to add physical drives to logical volume")
	}

	return nil
}

// DeletePDsFromLV deletes physical drives from a logical volume.
func (s *SoftwareRAIDController) DeletePDsFromLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	err := lvMetadata.Validate()
	if err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	for _, pdMetadata := range pdsMetadata {
		err := pdMetadata.Validate()
		if err != nil {
			return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
		}
	}

	err = s.iface.DeletePDsFromLV(lvMetadata, pdsMetadata...)
	if err != nil {
		return errors.Wrap(err, "failed to delete physical drives from logical volume")
	}

	return nil
}
