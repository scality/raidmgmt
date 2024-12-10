package raidcontroller

import (
	"fmt"
	"strconv"
)

// IDInt returns the ID of the RAID controller card as an integer.
func (m *Metadata) IDInt() (int, error) {
	idInt, err := strconv.Atoi(m.ID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", ErrParsingFailed, err)
	}

	return idInt, nil
}

func (m *Metadata) Validate() error {
	if m == nil {
		return ErrNil
	}

	if m.ID == "" {
		return ErrIDEmpty
	}

	id, err := m.IDInt()
	if err != nil {
		return fmt.Errorf("%s: %w", prefixErrMetadata, err)
	}

	if id < 0 {
		return ErrIDNegative
	}

	return nil
}
