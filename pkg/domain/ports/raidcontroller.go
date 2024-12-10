package ports

import (
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

type RAIDController interface {
	// Controllers returns a list of RAID controllers
	Controllers() ([]*raidcontroller.RAIDController, error)

	// PhysicalDrives returns a list of physical drives for a given RAID controller
	PhysicalDrives(metadata *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error)

	// LogicalVolumes returns a list of logical volumes for a given RAID controller
	LogicalVolumes(metadata *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error)

	// EnableJBOD enables JBOD mode on a physical drive
	EnableJBOD(metadata *physicaldrive.Metadata) error

	// DisableJBOD disables JBOD mode on a physical drive
	DisableJBOD(metadata *physicaldrive.Metadata) error

	// SetLVCacheOptions sets cache options on a logical volume
	SetLVCacheOptions(metadata *logicalvolume.Metadata, cacheOpts *logicalvolume.CacheOptions) error

	// CreateLV creates a logical volume from a request
	CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error)

	// AddPVToLV adds a physical drive to a logical volume
	AddPVToLV(lvMetadat *logicalvolume.Metadata, pvMetadata *physicaldrive.Metadata) error

	// DeleteLV deletes a logical volume
	DeleteLV(metadata *logicalvolume.Metadata) error

	// DeletePVFromLV deletes a physical drive from a logical volume
	DeletePVFromLV(lvMetadata *logicalvolume.Metadata, pvMetadata *physicaldrive.Metadata) error

	// StartBlink starts blinking a physical drive
	StartBlink(metadata *physicaldrive.Metadata) error

	// StopBlink stops blinking a physical drive
	StopBlink(metadata *physicaldrive.Metadata) error
}
