//nolint:lll,cyclop,gocognit // Structures with tags are too long for you, lll.
package physicaldrive

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/utils"
)

const (
	emptySlot string = "<empty>"
	nilSlot   string = "<nil>"
)

type (
	// PhysicalDrive represents a physical drive.
	PhysicalDrive struct {
		*Metadata // Metadata of the disk

		Slot          *Slot    `json:"slot,omitempty"`           // Slot
		Vendor        string   `json:"vendor,omitempty"`         // Vendor
		Model         string   `json:"model,omitempty"`          // Model
		Serial        string   `json:"serial,omitempty"`         // Serial number
		WWN           string   `json:"wwn,omitempty"`            // World Wide Name
		Size          uint64   `json:"size,omitempty"`           // Size in bytes
		Type          DiskType `json:"type,omitempty"`           // Type (e.g.: HDD, SSD)
		JBOD          bool     `json:"jbod,omitempty"`           // Is the disk in JBOD mode
		Status        PDStatus `json:"status,omitempty"`         // State (e.g.: Online, Offline, Failed)
		Reason        string   `json:"reason,omitempty"`         // Reason for the disk state
		DevicePath    string   `json:"device_path,omitempty"`    // Device path of the disk
		PermanentPath string   `json:"permanent_path,omitempty"` // Permanent path of the array (e.g.: /dev/disk/by-id/...)
	}

	// Slot identifies the slot of a disk.
	Slot struct {
		Port      string `json:"port,omitempty"`      // Port number (if available)
		Enclosure string `json:"enclosure,omitempty"` // Enclosure number (if available)
		Bay       string `json:"bay,omitempty"`       // Bay number (if available)
	}

	// Metadata represents the metadata of a physical drive.
	Metadata struct {
		CtrlMetadata *raidcontroller.Metadata `json:"controller_metadata,omitempty"` // Controller metadata of the disk
		ID           string                   `json:"id,omitempty"`                  // ID
	}
)

// String returns the string representation of the Slot instance.
func (s *Slot) String() string {
	if s == nil {
		return nilSlot
	}

	var parts []string

	if s.Enclosure != "" {
		parts = append(parts, s.Enclosure)
	}

	if s.Bay != "" {
		parts = append(parts, s.Bay)
	}

	if s.Port != "" {
		parts = append(parts, s.Port)
	}

	str := strings.Join(parts, ":")
	if str == "" {
		return emptySlot
	}

	return str
}

func (s *Slot) Format() string {
	if s == nil {
		return nilSlot
	}

	// Handle empty cases for all fields
	if s.Port == "" && s.Enclosure == "" && s.Bay == "" {
		return emptySlot
	}

	result := s.Port

	if s.Enclosure != "" {
		if result != "" {
			result += ":"
		}

		result += s.Enclosure
	}

	if s.Bay != "" {
		if result != "" {
			result += ":"
		}

		result += s.Bay
	}

	return result
}

// Available checks if the PhysicalDrive Status is PDStatusUnassignedGood.
func (pd *PhysicalDrive) IsAvailable() bool {
	return pd.Status == PDStatusUnassignedGood
}

// ComputePermanentPath computes the permanent path of the physical drive.
// NOTE: For physical drives backed by hardware RAID controllers this might not
// be available.
// nolint: funlen,nestif // This function is pretty simple to follow.
func (pd *PhysicalDrive) ComputePaths() error {
	if pd.PermanentPath == "" {
		if pd.Type == DiskTypeNVMe {
			if pd.Model != "" && pd.Serial != "" {
				permanentPath := fmt.Sprintf(
					"/dev/disk/by-id/nvme-%s_%s",
					strings.ReplaceAll(pd.Model, " ", "_"),
					strings.ReplaceAll(pd.Serial, " ", "_"),
				)
				if utils.FileExists(permanentPath) {
					pd.PermanentPath = permanentPath
				}
			}
		} else {
			if pd.WWN != "" {
				permanentPath := fmt.Sprintf("/dev/disk/by-id/wwn-%s", pd.WWN)
				if utils.FileExists(permanentPath) {
					pd.PermanentPath = permanentPath
				}
			}

			if pd.PermanentPath == "" && pd.Vendor != "" && pd.Model != "" && pd.Serial != "" {
				permanentPath := fmt.Sprintf(
					"/dev/disk/by-id/scsi-S%s_%s_%s",
					strings.ReplaceAll(pd.Vendor, " ", "_"),
					strings.ReplaceAll(pd.Model, " ", "_"),
					strings.ReplaceAll(pd.Serial, " ", "_"),
				)
				if utils.FileExists(permanentPath) {
					pd.PermanentPath = permanentPath
				}
			}
		}

		if pd.PermanentPath == "" {
			return errors.New("failed to compute permanent path")
		}
	}

	if pd.DevicePath == "" {
		resolvedPath, err := filepath.EvalSymlinks(pd.PermanentPath)
		if err != nil {
			return errors.Wrap(err, "failed to resolve permanent path symlink")
		}

		pd.DevicePath = resolvedPath
	} else {
		// Verify that device path matches the resolved symlink of permanent path
		resolvedPath, err := filepath.EvalSymlinks(pd.PermanentPath)
		if err != nil {
			return errors.Wrap(err, "failed to resolve permanent path symlink")
		}

		if resolvedPath != pd.DevicePath {
			return errors.New("device path does not match resolved permanent path")
		}
	}

	return nil
}

// IsEqualTo checks if the Slot instance is equal to another Slot instance.
func (s *Slot) IsEqualTo(other *Slot) bool {
	if s == nil && other == nil {
		return true
	}

	if s == nil || other == nil {
		return false
	}

	return s.Port == other.Port && s.Enclosure == other.Enclosure && s.Bay == other.Bay
}

// Validate checks if the Metadata instance is valid.
func (m *Metadata) Validate() error {
	if m == nil {
		return errors.New("metadata is nil")
	}

	if err := m.CtrlMetadata.Validate(); err != nil {
		return errors.Wrap(err, "controller metadata is invalid")
	}

	if m.ID == "" {
		return errors.New("ID is empty")
	}

	return nil
}

// ParseSlot parses a string into a Slot instance.
func ParseSlot(slot string) (*Slot, error) {
	if slot == "" {
		return nil, errors.New("slot is empty")
	}

	res := &Slot{}

	parts := strings.Split(slot, ":")
	if len(parts) > 3 { //nolint:mnd // just the number of parts
		return nil, errors.New("invalid slot format (too many parts)")
	}

	if len(parts) > 2 { //nolint:mnd // just the number of parts
		res.Port, parts = parts[0], parts[1:]
	}

	if len(parts) > 1 {
		res.Enclosure, parts = parts[0], parts[1:]
	}

	res.Bay = parts[0]

	return res, nil
}
