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
		ID              string                    // ID
		DevicePath      string                    // Device path of the array (e.g.: /dev/sda)
		PermanentPath   string                    // Permanent path of the array (e.g.: /dev/disk/by-id/...)
		RAIDLevel       RAIDLevel                 // RAID level of the array (e.g.: RAID 0, RAID 1, RAID 10, ...)
		PDrivesMetadata []*physicaldrive.Metadata // Physical drives composing the logical volume
		CacheOptions    *CacheOptions             // Cache options
		Status          LVStatus                  // State (e.g.: Online, Offline, Degraded)
		Reason          *string                   // Reason for the logical volume state
	}

	// Metadata represents the metadata of a logical volume.
	Metadata struct {
		CtrlMetadata *raidcontroller.Metadata // Controller of the logical volume
		// In the cases of RHEL8/mdadm based implementations
		// the ID is the name of the logical volume, like md0, md/0_0.
		// It will be appended to /dev/ to get the device path.
		ID string // ID
	}

	// Request represents the request to create a logical volume.
	Request struct {
		ID              string                    // ID
		CtrlMetadata    *raidcontroller.Metadata  // Controller of the logical volume
		RAIDLevel       RAIDLevel                 // RAID level of the array (e.g.: RAID 0, RAID 1, RAID 10, ...)
		PDrivesMetadata []*physicaldrive.Metadata // Physical drives composing the logical volume
		CacheOptions    *CacheOptions             // Cache options
		Name            string                    // Name of the logical volume
	}

	// CacheOptions represents the cache options of a logical volume.
	CacheOptions struct {
		ReadPolicy  ReadPolicy  // Read policy of the cache (e.g.: ReadAhead, NoReadAhead)
		WritePolicy WritePolicy // Write policy of the cache (e.g.: WriteBack, WriteThrough)
		IOPolicy    IOPolicy    // IO policy of the cache (e.g.: Direct, Cached)
	}
)

var RAIDLevelMap = map[string]RAIDLevel{ //nolint:gochecknoglobals // Will be fixed eventually
	"RAID0":  RAIDLevel0,
	"RAID1":  RAIDLevel1,
	"RAID10": RAIDLevel10,
}

var RAIDLevelMapToString = map[RAIDLevel]string{ //nolint:gochecknoglobals,lll // Will be fixed eventually
	RAIDLevel0:  "raid0",
	RAIDLevel1:  "raid1",
	RAIDLevel10: "raid10",
}
