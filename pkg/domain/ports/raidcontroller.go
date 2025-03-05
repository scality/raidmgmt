package ports

import (
	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
)

type (
	ControllersGetter interface {
		// Controllers returns a list of RAID controllers
		Controllers() ([]*raidcontroller.RAIDController, error)

		// Controller returns a RAID controller for a given metadata
		Controller(*raidcontroller.Metadata) (*raidcontroller.RAIDController, error)
	}

	PhysicalDrivesGetter interface {
		// PhysicalDrives returns a list of physical drives for a given RAID controller
		PhysicalDrives(*raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error)

		// PhysicalDrive returns a physical drive for a given metadata
		PhysicalDrive(*physicaldrive.Metadata) (*physicaldrive.PhysicalDrive, error)
	}

	LogicalVolumesGetter interface {
		// LogicalVolumes returns a list of logical volumes for a given RAID controller
		LogicalVolumes(*raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error)

		// LogicalVolume returns a logical volume for a given metadata
		LogicalVolume(*logicalvolume.Metadata) (*logicalvolume.LogicalVolume, error)
	}

	LogicalVolumesManager interface {
		// CreateLV creates a logical volume from a request
		CreateLV(*logicalvolume.Request) (*logicalvolume.LogicalVolume, error)

		// DeleteLV deletes a logical volume
		DeleteLV(*logicalvolume.Metadata) error

		// AddPDsToLV adds a physical drive to a logical volume
		AddPDsToLV(*logicalvolume.Metadata, ...*physicaldrive.Metadata) error

		// DeletePDsFromLV deletes a physical drive from a logical volume
		DeletePDsFromLV(*logicalvolume.Metadata, ...*physicaldrive.Metadata) error
	}

	JBODSetter interface {
		// EnableJBOD enables JBOD mode on a physical drive
		EnableJBOD(*physicaldrive.Metadata) error

		// DisableJBOD disables JBOD mode on a physical drive
		DisableJBOD(*physicaldrive.Metadata) error
	}

	LVCacheSetter interface {
		// SetLVCacheOptions sets cache options on a logical volume
		SetLVCacheOptions(*logicalvolume.Metadata, *logicalvolume.CacheOptions) error
	}

	Blinker interface {
		// StartBlink starts blinking a physical drive
		StartBlink(*physicaldrive.Metadata) error

		// StopBlink stops blinking a physical drive
		StopBlink(*physicaldrive.Metadata) error
	}
)
