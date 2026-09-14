package physicaldrive_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
)

// A slot renders as a vendor CLI drive address, and ParseSlot reads that address
// back into the same parts. Printing a slot in one order and parsing it in
// another turned a drive id such as 1I:1:1 into 1:1:1I.
func TestSlotRendersTheAddressParseSlotReads(t *testing.T) {
	tests := []struct {
		name     string
		slot     *physicaldrive.Slot
		expected string
	}{
		{
			// ssacli addresses a drive as port:box:bay, and box is what the
			// entity calls the enclosure.
			name:     "ssacli, letter port",
			slot:     &physicaldrive.Slot{Port: "1I", Enclosure: "1", Bay: "1"},
			expected: "1I:1:1",
		},
		{
			name:     "ssacli, fourth port of a P816i-a",
			slot:     &physicaldrive.Slot{Port: "4I", Enclosure: "6", Bay: "1"},
			expected: "4I:6:1",
		},
		{
			// The HPE documentation also shows plain numeric ports.
			name:     "ssacli, numeric port",
			slot:     &physicaldrive.Slot{Port: "1", Enclosure: "1", Bay: "3"},
			expected: "1:1:3",
		},
		{
			// storcli2 and perccli2 address a drive as EID:Slt, with no port.
			name:     "storcli2 and perccli2, EID:Slt",
			slot:     &physicaldrive.Slot{Enclosure: "320", Bay: "11"},
			expected: "320:11",
		},
		{
			// The legacy megaraid adapter reads the same EID:Slt form.
			name:     "megaraid, EID:Slt",
			slot:     &physicaldrive.Slot{Enclosure: "251", Bay: "10"},
			expected: "251:10",
		},
		{
			// A drive behind no enclosure keeps its bay alone.
			name:     "bay alone",
			slot:     &physicaldrive.Slot{Bay: "4"},
			expected: "4",
		},
		{
			name:     "no part at all",
			slot:     &physicaldrive.Slot{},
			expected: "<empty>",
		},
		{
			// Software RAID never fills a slot, it addresses a drive by its
			// device path.
			name:     "software RAID, no slot at all",
			slot:     nil,
			expected: "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.slot.String())

			if tt.slot == nil || tt.expected == "<empty>" {
				return
			}

			parsed, err := physicaldrive.ParseSlot(tt.slot.String())
			require.NoError(t, err)
			assert.Equal(t, tt.slot, parsed)
		})
	}
}

// Rendering drops a missing part without leaving a marker, so a slot with a hole
// in the middle comes back from ParseSlot as another slot. No adapter builds
// one: ssacli fills the three parts, storcli2 and megaraid fill the last two.
func TestSlotWithAHoleDoesNotSurviveParseSlot(t *testing.T) {
	slot := &physicaldrive.Slot{Port: "1I", Bay: "3"}

	assert.Equal(t, "1I:3", slot.String())

	parsed, err := physicaldrive.ParseSlot(slot.String())
	require.NoError(t, err)
	assert.Equal(t, &physicaldrive.Slot{Enclosure: "1I", Bay: "3"}, parsed)
}
