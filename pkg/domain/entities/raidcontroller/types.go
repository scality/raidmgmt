package raidcontroller

type (
	// RAIDController represents a RAID controller card.
	RAIDController struct {
		Metadata *Metadata
		Name     string // Name of the RAID controller card
		Serial   string // Serial number of the RAID controller card
	}

	// Metadata represents the metadata of a RAID controller card.
	Metadata struct {
		ID string // ID of the RAID controller card
	}
)
