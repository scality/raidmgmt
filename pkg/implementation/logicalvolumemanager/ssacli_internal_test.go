package logicalvolumemanager

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
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

// ssacliConfig reads a "controller show config" fixture.
func ssacliConfig(t *testing.T, name string) []byte {
	t.Helper()

	output, err := os.ReadFile("testdata/ssacli/controller/show/" + name)
	require.NoError(t, err)

	return output
}

// A "show config" output lists an array as a block opening with its logical
// drives and closing with its member drives, so the drive belongs to the
// logical drive declared in its own block. Reading it the other way round made
// every freshly created volume look like a drive shared by several volumes.
func TestGetLogicalDriveID(t *testing.T) {
	tests := []struct {
		name          string
		fixture       string
		driveID       string
		expected      string
		expectedError string
	}{
		{
			name:     "the drive belongs to the volume of its array",
			fixture:  "config.txt",
			driveID:  "1I:1:3",
			expected: "2",
		},
		{
			name:     "a volume further down the output",
			fixture:  "config.txt",
			driveID:  "3I:3:4",
			expected: "11",
		},
		{
			name:          "an unassigned drive belongs to none",
			fixture:       "config.txt",
			driveID:       "1I:1:1",
			expectedError: "physical drive 1I:1:1 not found in any logical drive",
		},
		{
			name:          "a drive absent from the controller",
			fixture:       "config.txt",
			driveID:       "9I:9:9",
			expectedError: "physical drive 9I:9:9 not found in any logical drive",
		},
		{
			// A substring match would read bay 11 as bay 1 and hand out the
			// volume of another drive.
			name:          "a two digit bay is not confused with its prefix",
			fixture:       "config-two-digit-bay.txt",
			driveID:       "1I:1:1",
			expectedError: "physical drive 1I:1:1 not found in any logical drive",
		},
		{
			name:     "the two digit bay finds its own volume",
			fixture:  "config-two-digit-bay.txt",
			driveID:  "1I:1:11",
			expected: "7",
		},
		{
			name:          "an array declaring two volumes is ambiguous",
			fixture:       "config-array-with-two-logicaldrives.txt",
			driveID:       "1I:1:1",
			expectedError: "physical drive 1I:1:1 found in multiple logical drives: 1, 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &logicalvolume.Request{
				PDrivesMetadata: []*physicaldrive.Metadata{{ID: tt.driveID}},
			}

			id, err := getLogicalDriveID(request, ssacliConfig(t, tt.fixture))

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				assert.Empty(t, id)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, id)
		})
	}
}
