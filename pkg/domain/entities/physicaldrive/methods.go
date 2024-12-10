package physicaldrive

import (
	"fmt"
)

// Validate checks if the Metadata instance is valid.
func (m *Metadata) Validate() error {
	if m == nil {
		return ErrNil
	}

	if m.CtrlMetadata == nil {
		return ErrCtrlMetaNil
	}

	if err := m.CtrlMetadata.Validate(); err != nil {
		return fmt.Errorf("%s: %w", prefixErr, err)
	}

	return nil
}

func AreSlotsEqual(s1 *Slot, s2 *Slot) bool {
	if s1 == nil && s2 == nil {
		return true
	}

	if s1 == nil || s2 == nil {
		return false
	}

	return s1.Port == s2.Port && s1.Enclosure == s2.Enclosure && s1.Bay == s2.Bay
}
