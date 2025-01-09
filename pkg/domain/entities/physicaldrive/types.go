package physicaldrive

import (
	"fmt"

	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

type (
	// PhysicalDrive represents a physical drive.
	PhysicalDrive struct {
		CtrlMetadata *raidcontroller.Metadata // Controller of the disk
		ID           string                   // ID of the disk
		Vendor       string                   // Vendor of the disk
		Model        string                   // Model of the disk
		Serial       string                   // Serial number of the disk
		Slot         *Slot                    // Slot of the disk
		Size         uint64                   // Size of the disk in bytes
		Type         DiskType                 // Type of the disk (e.g.: HDD, SSD)
		JBOD         bool                     // Is the disk in JBOD mode
		Status       PDStatus                 // State of the disk (e.g.: Online, Offline, Failed)
		Reason       string                   // Reason for the disk state
	}

	// Slot identifies the slot of a disk.
	Slot struct {
		Port      string // Port number of the disk (if available)
		Enclosure string // Enclosure number of the disk (if available)
		Bay       string // Bay number of the disk (if available)
	}

	// Metadata represents the metadata of a physical drive.
	Metadata struct {
		CtrlMetadata *raidcontroller.Metadata // Controller metadata of the disk
		Slot         *Slot                    // Slot of the disk
	}
)

func (s *Slot) String() string {
	if s == nil {
		return ""
	}

	if s.Enclosure == "" {
		return s.Bay
	}

	return fmt.Sprintf("%s:%s", s.Enclosure, s.Bay)
}

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
		return ErrNil
	}

	if err := m.CtrlMetadata.Validate(); err != nil {
		return fmt.Errorf("%s: %w", prefixErr, err)
	}

	return nil
}
