package logicalvolume

import (
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

const (
	RAID0DiskRequirement  = 1
	RAID1DiskRequirement  = 2
	RAID10DiskRequirement = 4
)

type (
	// LogicalVolume represents a logical volume.
	LogicalVolume struct {
		CtrlMetadata    *raidcontroller.Metadata  // Controller of the logical volume
		ID              string                    // ID of the array
		DevicePath      string                    // Device path of the array (e.g.: /dev/sda)
		PermanentPath   string                    // Permanent path of the array (e.g.: /dev/disk/by-id/...)
		RAIDLevel       RAIDLevel                 // RAID level of the array (e.g.: RAID 0, RAID 1, RAID 10, ...)
		PDrivesMetadata []*physicaldrive.Metadata // Physical drives composing the array
		CacheOptions    *CacheOptions             // Cache options of the array
		Status          LVStatus                  // State of the array (e.g.: Online, Offline, Degraded)
		Reason          string                    // Reason for the logical volume state
	}

	// Metadata represents the metadata of a logical volume.
	Metadata struct {
		CtrlMetadata *raidcontroller.Metadata // Controller of the logical volume
		ID           string                   // ID of the array
	}

	// Request represents the request to create a logical volume.
	Request struct {
		CtrlMetadata    *raidcontroller.Metadata  // Controller of the logical volume
		RAIDLevel       RAIDLevel                 // RAID level of the array (e.g.: RAID 0, RAID 1, RAID 10, ...)
		PDrivesMetadata []*physicaldrive.Metadata // Physical drives composing the array
		CacheOptions    *CacheOptions             // Cache options of the array
	}

	// CacheOptions represents the cache options of a logical volume.
	CacheOptions struct {
		ReadPolicy  ReadPolicy  // Read policy of the cache (e.g.: ReadAhead, NoReadAhead)
		WritePolicy WritePolicy // Write policy of the cache (e.g.: WriteBack, WriteThrough)
		IOPolicy    IOPolicy    // IO policy of the cache (e.g.: Direct, Cached)
	}
)
