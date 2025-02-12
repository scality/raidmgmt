//nolint:lll // Structures with tags are too long for you, lll.
package physicaldrive

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
)

const (
	emptySlot string = "<empty>"
	nilSlot   string = "<nil>"
)

type (
	// PhysicalDrive represents a physical drive.
	PhysicalDrive struct {
		*Metadata `json:"metadata,omitempty"` // Metadata of the disk

		ID            string   `json:"id,omitempty"`             // ID
		Vendor        string   `json:"vendor,omitempty"`         // Vendor
		Model         string   `json:"model,omitempty"`          // Model
		Serial        string   `json:"serial,omitempty"`         // Serial number
		Size          uint64   `json:"size,omitempty"`           // Size in bytes
		Type          DiskType `json:"type,omitempty"`           // Type (e.g.: HDD, SSD)
		JBOD          bool     `json:"jbod,omitempty"`           // Is the disk in JBOD mode
		Status        PDStatus `json:"status,omitempty"`         // State (e.g.: Online, Offline, Failed)
		Reason        string   `json:"reason,omitempty"`         // Reason for the disk state
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
		DevicePath   string                   `json:"device_path,omitempty"`         // Device path of the disk
		Slot         *Slot                    `json:"slot,omitempty"`                // Slot
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

// Available checks if the PhysicalDrive Status is PDStatusUnassignedGood.
func (pd *PhysicalDrive) IsAvailable() bool {
	return pd.Status == PDStatusUnassignedGood
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

	// Detailed validation of the slot is done in each adapter
	// as the validation rules may vary between different adapters
	// Some fields may be optional or mandatory depending on the adapter
	if m.Slot == nil {
		return errors.New("slot is nil")
	}

	if m.Slot.String() == emptySlot {
		return errors.New("slot is empty")
	}

	return nil
}
