package rhel8

import (
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumemanager"
	"github.com/scality/raidmgmt/pkg/implementation/physicaldrivegetter"
)

type RHEL8 struct {
	ports.PhysicalDrivesGetter
	ports.LogicalVolumesGetter
	ports.LogicalVolumesManager
}

var _ ports.SoftwareRAIDController = &RHEL8{}

// NewRHEL8 creates a new RHEL8 instance
//
//	using compatible implementations of the required interfaces
//	(MDADM for logical volumes, UDevADM and LSBLK for physical drives)
func NewRHEL8(
	physicalDriveGetter physicaldrivegetter.RHEL8,
	logicalVolumesGetter logicalvolumegetter.MDADM,
	logicalVolumesManager logicalvolumemanager.MDADM,
) *RHEL8 {
	return &RHEL8{
		PhysicalDrivesGetter:  &physicalDriveGetter,
		LogicalVolumesGetter:  &logicalVolumesGetter,
		LogicalVolumesManager: &logicalVolumesManager,
	}
}
