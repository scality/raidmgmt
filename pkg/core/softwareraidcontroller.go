package core

import (
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
	return physicalDrives(s.iface, metadata)
}

// PhysicalDrive returns a physical drive.
// Controller metadata aren't used as this is software RAID.
func (s *SoftwareRAIDController) PhysicalDrive(
	metadata *physicaldrive.Metadata,
) (*physicaldrive.PhysicalDrive, error) {
	return physicalDrive(s.iface, metadata)
}

// LogicalVolumes returns a list of logical volumes.
// Controller metadata aren't used as this is software RAID.
func (s *SoftwareRAIDController) LogicalVolumes(
	metadata *raidcontroller.Metadata,
) ([]*logicalvolume.LogicalVolume, error) {
	return logicalVolumes(s.iface, metadata)
}

// LogicalVolume returns a logical volume.
// Controller metadata aren't used as this is software RAID.
func (s *SoftwareRAIDController) LogicalVolume(
	metadata *logicalvolume.Metadata,
) (*logicalvolume.LogicalVolume, error) {
	return logicalVolume(s.iface, metadata)
}

// CreateLV creates a logical volume.
func (s *SoftwareRAIDController) CreateLV(
	request *logicalvolume.Request,
) (*logicalvolume.LogicalVolume, error) {
	return createLV(s.iface, request)
}

// DeleteLV deletes a logical volume.
func (s *SoftwareRAIDController) DeleteLV(
	metadata *logicalvolume.Metadata,
) error {
	return deleteLV(s.iface, metadata)
}

// AddPDsToLV adds physical drives to a logical volume.
func (s *SoftwareRAIDController) AddPDsToLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	return addPDsToLV(s.iface, lvMetadata, pdsMetadata...)
}

// DeletePDsFromLV deletes physical drives from a logical volume.
func (s *SoftwareRAIDController) DeletePDsFromLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	return deletePDsFromLV(s.iface, lvMetadata, pdsMetadata...)
}
