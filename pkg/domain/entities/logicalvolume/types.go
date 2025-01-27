package logicalvolume

import (
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

const (
	RAID0DiskRequirement  int = 1
	RAID1DiskRequirement  int = 2
	RAID10DiskRequirement int = 4

	sizeTolerancePercent uint64 = 5
	percent              uint64 = 100

	ErrUnavailableDrives = "unavailable drives: %s"
)

type (
	// LogicalVolume represents a logical volume.
	LogicalVolume struct {
		*Metadata // Metadata of the logical volume

		//nolint:lll // The comment is long but it is necessary to explain the field
		PermanentPath   string                    // Permanent path of the array (e.g.: /dev/disk/by-id/...)
		DevicePath      string                    // Device path of the array (e.g.: /dev/sda)
		RAIDLevel       RAIDLevel                 // RAID level of the array
		PDrivesMetadata []*physicaldrive.Metadata // Physical drives composing the logical volume
		CacheOptions    *CacheOptions             // Cache options
		Status          LVStatus                  // State (e.g.: Online, Offline, Degraded)
		Reason          string                    // Reason for the logical volume state
	}

	// Metadata represents the metadata of a logical volume.
	Metadata struct {
		CtrlMetadata *raidcontroller.Metadata // Controller of the logical volume
		ID           string                   // ID
	}

	// Request represents the request to create a logical volume.
	Request struct {
		CtrlMetadata    *raidcontroller.Metadata  // Controller of the logical volume
		ID              string                    // ID
		Name            string                    // Name of the logical volume
		RAIDLevel       RAIDLevel                 // RAID level of the array
		PDrivesMetadata []*physicaldrive.Metadata // Physical drives composing the logical volume
		CacheOptions    *CacheOptions             // Cache options
	}

	// CacheOptions represents the cache options of a logical volume.
	CacheOptions struct {
		ReadPolicy  ReadPolicy  // Read policy
		WritePolicy WritePolicy // Write policy
		IOPolicy    IOPolicy    // IO policy
	}
)
