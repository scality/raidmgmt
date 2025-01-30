//nolint:lll // Structures with tags are too long for you, lll.
package logicalvolume

import (
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
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
		*Metadata `json:"metadata,omitempty"` // Metadata of the logical volume

		PermanentPath   string                    `json:"permanent_path,omitempty"`           // Permanent path of the array (e.g.: /dev/disk/by-id/...)
		DevicePath      string                    `json:"device_path,omitempty"`              // Device path of the array (e.g.: /dev/sda)
		RAIDLevel       RAIDLevel                 `json:"raid_level,omitempty"`               // RAID level of the array
		PDrivesMetadata []*physicaldrive.Metadata `json:"physical_drives_metadata,omitempty"` // Physical drives composing the logical volume
		CacheOptions    *CacheOptions             `json:"cache_options,omitempty"`            // Cache options
		Status          LVStatus                  `json:"status,omitempty"`                   // State (e.g.: Online, Offline, Degraded)
		Reason          string                    `json:"reason,omitempty"`                   // Reason for the logical volume state
	}

	// Metadata represents the metadata of a logical volume.
	Metadata struct {
		CtrlMetadata *raidcontroller.Metadata `json:"controller_metadata,omitempty"` // Controller of the logical volume
		ID           string                   `json:"id,omitempty"`                  // ID
	}

	// Request represents the request to create a logical volume.
	Request struct {
		CtrlMetadata    *raidcontroller.Metadata  `json:"controller_metadata,omitempty"`      // Controller of the logical volume
		ID              string                    `json:"id,omitempty"`                       // ID
		Name            string                    `json:"name,omitempty"`                     // Name of the logical volume
		RAIDLevel       RAIDLevel                 `json:"raid_level,omitempty"`               // RAID level of the array
		PDrivesMetadata []*physicaldrive.Metadata `json:"physical_drives_metadata,omitempty"` // Physical drives composing the logical volume
		CacheOptions    *CacheOptions             `json:"cache_options,omitempty"`            // Cache options
	}

	// CacheOptions represents the cache options of a logical volume.
	CacheOptions struct {
		ReadPolicy  ReadPolicy  `json:"read_policy,omitempty"`  // Read policy
		WritePolicy WritePolicy `json:"write_policy,omitempty"` // Write policy
		IOPolicy    IOPolicy    `json:"io_policy,omitempty"`    // IO policy
	}
)
