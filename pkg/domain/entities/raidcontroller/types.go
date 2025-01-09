package raidcontroller

type (
	// RAIDController represents a RAID controller card.
	RAIDController struct {
		Metadata *Metadata // Metadata of the RAID controller card

		Name   string // Name of the RAID controller card
		Serial string // Serial number of the RAID controller card
		JBOD   bool   // Can the RAID controller card be set in JBOD mode
	}

	// Metadata represents the metadata of a RAID controller card.
	Metadata struct {
		ID int // ID of the RAID controller card
	}
)

// ToMetadata returns the Metadata instance of the RAIDController.
func (rc *RAIDController) ToMetadata() *Metadata {
	if rc == nil {
		return nil
	}

	return rc.Metadata
}

// Validate validates the Metadata instance.
func (m *Metadata) Validate() error {
	if m == nil {
		return ErrNil
	}

	if m.ID < 0 {
		return ErrIDNegative
	}

	return nil
}

func (rc *RAIDController) HasJBOD() bool {
	return rc.JBOD
}
