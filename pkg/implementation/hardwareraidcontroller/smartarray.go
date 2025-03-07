package hardwareraidcontroller

import (
	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/ports"
)

type SmartArray struct {
	ports.ControllersGetter
	ports.PhysicalDrivesGetter
	ports.LogicalVolumesGetter
	ports.LogicalVolumesManager
	ports.Blinker
}

var _ ports.HardwareRAIDController = &SmartArray{}

//nolint:revive // This wraps interfaces together.
func NewSmartArray(
	controllersGetter ports.ControllersGetter,
	physicalDrivesGetter ports.PhysicalDrivesGetter,
	logicalVolumesGetter ports.LogicalVolumesGetter,
	logicalVolumesManager ports.LogicalVolumesManager,
	blinker ports.Blinker,
) *SmartArray {
	return &SmartArray{
		ControllersGetter:     controllersGetter,
		PhysicalDrivesGetter:  physicalDrivesGetter,
		LogicalVolumesGetter:  logicalVolumesGetter,
		LogicalVolumesManager: logicalVolumesManager,
		Blinker:               blinker,
	}
}

func (*SmartArray) EnableJBOD(_ *physicaldrive.Metadata) error {
	panic("not implemented for smartarray hardware RAID controller")
}

func (*SmartArray) DisableJBOD(_ *physicaldrive.Metadata) error {
	panic("not implemented for smartarray hardware RAID controller")
}

func (*SmartArray) SetLVCacheOptions(*logicalvolume.Metadata, *logicalvolume.CacheOptions) error {
	panic("not implemented for smartarray hardware RAID controller")
}
