package ports

import (
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

type RAIDController interface {
	ControllersGetter
	PhysicalDrivesGetter
	LogicalVolumesGetter
	LogicalVolumesManager
	LVCacheSetter
	JBODSetter
	Blinker
}

type ControllersGetter interface {
	// Controllers returns a list of RAID controllers
	Controllers() ([]*raidcontroller.RAIDController, error)

	// Controller returns a RAID controller for a given metadata
	Controller(metadata *raidcontroller.Metadata) (*raidcontroller.RAIDController, error)
}

type PhysicalDrivesGetter interface {
	// PhysicalDrives returns a list of physical drives for a given RAID controller
	PhysicalDrives(metadata *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error)

	// PhysicalDrive returns a physical drive for a given metadata
	PhysicalDrive(metadata *physicaldrive.Metadata) (*physicaldrive.PhysicalDrive, error)
}

type LogicalVolumesGetter interface {
	// LogicalVolumes returns a list of logical volumes for a given RAID controller
	LogicalVolumes(metadata *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error)

	// LogicalVolume returns a logical volume for a given metadata
	LogicalVolume(metadata *logicalvolume.Metadata) (*logicalvolume.LogicalVolume, error)
}

type LogicalVolumesManager interface {
	// CreateLV creates a logical volume from a request
	CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error)

	// DeleteLV deletes a logical volume
	DeleteLV(metadata *logicalvolume.Metadata) error

	// AddPVToLV adds a physical drive to a logical volume
	AddPVToLV(lvMetadata *logicalvolume.Metadata, pvMetadata *physicaldrive.Metadata) error

	// DeletePVFromLV deletes a physical drive from a logical volume
	DeletePVFromLV(lvMetadata *logicalvolume.Metadata, pvMetadata *physicaldrive.Metadata) error
}

type JBODSetter interface {
	// EnableJBOD enables JBOD mode on a physical drive
	EnableJBOD(metadata *physicaldrive.Metadata) error

	// DisableJBOD disables JBOD mode on a physical drive
	DisableJBOD(metadata *physicaldrive.Metadata) error
}

type LVCacheSetter interface {
	// SetLVCacheOptions sets cache options on a logical volume
	SetLVCacheOptions(metadata *logicalvolume.Metadata, cacheOpts *logicalvolume.CacheOptions) error
}

type Blinker interface {
	// StartBlink starts blinking a physical drive
	StartBlink(metadata *physicaldrive.Metadata) error

	// StopBlink stops blinking a physical drive
	StopBlink(metadata *physicaldrive.Metadata) error
}
