package raidcontroller

import (
	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
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

var _ ports.RAIDController = &RHEL8{}

// NewRHEL8 creates a new RHEL8 instance
//
//	using compatible implementations of the required interfaces
//	(MDADM for logical volumes, UDevADM and LSBLK for physical drives)
func NewRHEL8(
	physicalDriveGetter *physicaldrivegetter.RHEL8,
	logicalVolumesGetter *logicalvolumegetter.MDADM,
	logicalVolumesManager *logicalvolumemanager.MDADM,
) *RHEL8 {
	return &RHEL8{
		PhysicalDrivesGetter:  physicalDriveGetter,
		LogicalVolumesGetter:  logicalVolumesGetter,
		LogicalVolumesManager: logicalVolumesManager,
	}
}

func (*RHEL8) Controllers() ([]*raidcontroller.RAIDController, error) {
	return nil, errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cannot list hardware RAID controllers on RHEL8",
	)
}

func (*RHEL8) Controller(_ *raidcontroller.Metadata) (*raidcontroller.RAIDController, error) {
	return nil, errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cannot get hardware RAID controller on RHEL8",
	)
}

func (*RHEL8) SetLVCacheOptions(_ *logicalvolume.Metadata, _ *logicalvolume.CacheOptions) error {
	return errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cannot set cache options on RHEL8",
	)
}

func (*RHEL8) EnableJBOD(_ *physicaldrive.Metadata) error {
	return errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cannot enable JBOD on RHEL8",
	)
}

func (*RHEL8) DisableJBOD(_ *physicaldrive.Metadata) error {
	return errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cannot disable JBOD on RHEL8",
	)
}

func (*RHEL8) StartBlink(_ *physicaldrive.Metadata) error {
	return errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cannot start blinking on RHEL8",
	)
}

func (*RHEL8) StopBlink(_ *physicaldrive.Metadata) error {
	return errors.Wrap(
		ports.ErrFunctionNotSupportedByImplementation,
		"cannot stop blinking on RHEL8",
	)
}
