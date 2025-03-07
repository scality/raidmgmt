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

func (s *SmartArray) EnableJBOD(metadata *physicaldrive.Metadata) error {
	panic("not implemented for smartarray hardware RAID controller")
}

func (s *SmartArray) DisableJBOD(metadata *physicaldrive.Metadata) error {
	panic("not implemented for smartarray hardware RAID controller")
}

func (s *SmartArray) SetLVCacheOptions(*logicalvolume.Metadata, *logicalvolume.CacheOptions) error {
	panic("not implemented for smartarray hardware RAID controller")
}
