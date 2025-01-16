package raidcontroller

import "errors"

type (
	// RAIDController represents a RAID controller card.
	RAIDController struct {
		Metadata *Metadata // Metadata of the RAID controller card

		Name            string // Name of the RAID controller card
		Serial          string // Serial number of the RAID controller card
		IsJBODSupported bool   // Can the RAID controller card be set in JBOD mode
		IsJBODEnabled   bool   // Is the RAID controller card in JBOD mode
	}

	// Metadata represents the metadata of a RAID controller card.
	Metadata struct {
		ID int // ID of the RAID controller card
	}
)

// Validate validates the Metadata instance.
func (m *Metadata) Validate() error {
	if m == nil {
		return errors.New("Metadata is nil")
	}

	if m.ID < 0 {
		return errors.New("ID is negative")
	}

	return nil
}
