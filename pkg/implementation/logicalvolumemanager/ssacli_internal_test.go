package logicalvolumemanager

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
)

// ssacli takes the RAID level as a number in its create grammar, and spells
// RAID 10 as 1+0. A level it cannot express has to fail closed here rather than
// reach the command line, where it would only earn an exit status.
func TestSSACLIRAIDLevelToken(t *testing.T) {
	tests := []struct {
		name          string
		level         logicalvolume.RAIDLevel
		expected      string
		expectedFound bool
	}{
		{
			name:          "RAID 0",
			level:         logicalvolume.RAIDLevel0,
			expected:      "0",
			expectedFound: true,
		},
		{
			name:          "RAID 1",
			level:         logicalvolume.RAIDLevel1,
			expected:      "1",
			expectedFound: true,
		},
		{
			name:          "RAID 10 spells 1+0",
			level:         logicalvolume.RAIDLevel10,
			expected:      "1+0",
			expectedFound: true,
		},
		{
			name:          "an unknown level is not expressible",
			level:         logicalvolume.RAIDLevelUnknown,
			expected:      "",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, found := ssacliRAIDLevelToken(tt.level)

			assert.Equal(t, tt.expectedFound, found)
			assert.Equal(t, tt.expected, token)
		})
	}
}
