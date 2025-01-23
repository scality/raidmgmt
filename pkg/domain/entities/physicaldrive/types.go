package physicaldrive

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

const (
	emptySlot string = "<empty>"
	nilSlot   string = "<nil>"
)

type (
	// PhysicalDrive represents a physical drive.
	PhysicalDrive struct {
		CtrlMetadata *raidcontroller.Metadata // Controller of the disk
		ID           string                   // ID
		Vendor       string                   // Vendor
		Model        string                   // Model
		Serial       string                   // Serial number
		Slot         *Slot                    // Slot
		Size         uint64                   // Size in bytes
		Type         DiskType                 // Type (e.g.: HDD, SSD)
		JBOD         bool                     // Is the disk in JBOD mode
		Status       PDStatus                 // State (e.g.: Online, Offline, Failed)
		Reason       *string                  // Reason for the disk state
	}

	// Slot identifies the slot of a disk.
	Slot struct {
		Port      string // Port number (if available)
		Enclosure string // Enclosure number (if available)
		Bay       string // Bay number (if available)
	}

	// Metadata represents the metadata of a physical drive.
	Metadata struct {
		CtrlMetadata *raidcontroller.Metadata // Controller metadata of the disk
		Slot         *Slot                    // Slot
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
func (pd *PhysicalDrive) Available() bool {
	return pd.Status == PDStatusUnassignedGood
}

// ToMetadata returns the Metadata instance of the PhysicalDrive.
func (pd *PhysicalDrive) ToMetadata() *Metadata {
	if pd == nil {
		return nil
	}

	return &Metadata{
		CtrlMetadata: pd.CtrlMetadata,
		Slot:         pd.Slot,
	}
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
