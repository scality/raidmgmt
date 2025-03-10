package raidcontroller

import (
	"github.com/pkg/errors"

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

var _ ports.RAIDController = &SmartArray{}

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
	return errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cannot enable JBOD on SmartArray",
	)
}

func (*SmartArray) DisableJBOD(_ *physicaldrive.Metadata) error {
	return errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cant disable JBOD on SmartArray",
	)
}

func (*SmartArray) SetLVCacheOptions(*logicalvolume.Metadata, *logicalvolume.CacheOptions) error {
	return errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cannot set cache options on SmartArray",
	)
}
