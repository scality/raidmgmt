package logicalvolumegetter

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/utils"
)

// MockCommandRunner is a manual testify mock of commandrunner.CommandRunner,
// returning canned storcli2 fixtures. It mirrors the mock used by the storcli2
// physical-drive getter.
type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)

	return arguments.Get(0).([]byte), arguments.Error(1)
}

// storcli2Fixture reads a storcli2 JSON fixture from the package testdata.
func storcli2Fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile("testdata/storcli2/" + name)
	require.NoError(t, err)

	return data
}

func TestStorCLI2LogicalVolumes(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/vall", "show", "all"}).
		Return(storcli2Fixture(t, "show/all.json"), nil)

	s := NewStorCLI2(mockRunner)

	volumes, err := s.LogicalVolumes(&raidcontroller.Metadata{ID: 0})
	require.NoError(t, err)
	require.Len(t, volumes, 24)

	expectedSize, err := utils.ConvertSizeBytes("9.094 TiB")
	require.NoError(t, err)

	first := volumes[0]
	assert.Equal(t, "1", first.ID)
	assert.Equal(t, 0, first.CtrlMetadata.ID)
	assert.Equal(t, logicalvolume.RAIDLevel0, first.RAIDLevel)
	assert.Equal(t, logicalvolume.LVStatusOptimal, first.Status)
	assert.Equal(t, expectedSize, first.Size)
	assert.Equal(t, "/dev/sdb", first.DevicePath)
	assert.Equal(t, "/dev/disk/by-id/wwn-0x600062b22066d54069faf124ced57e62", first.PermanentPath)

	require.NotNil(t, first.CacheOptions)
	assert.Equal(t, logicalvolume.ReadPolicyNoReadAhead, first.CacheOptions.ReadPolicy)
	assert.Equal(t, logicalvolume.WritePolicyWriteBack, first.CacheOptions.WritePolicy)

	require.Len(t, first.PDrivesMetadata, 1)
	assert.Equal(t, "306:0", first.PDrivesMetadata[0].ID)
	assert.Equal(t, 0, first.PDrivesMetadata[0].CtrlMetadata.ID)
}

func TestStorCLI2LogicalVolume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		id          string
		selector    string
		fixture     string
		expectError bool
		errContains string
	}{
		{
			name:     "nominal case",
			id:       "1",
			selector: "/c0/v1",
			fixture:  "show/v1.json",
		},
		{
			// storcli2 reports an out-of-range VD as a command-level failure
			// payload (exit code 0), which storcli2.Decode surfaces as the
			// in-JSON error message.
			name:        "invalid volume",
			id:          "999",
			selector:    "/c0/v999",
			fixture:     "show/v999_invalid.json",
			expectError: true,
			errContains: "Invalid VD number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockRunner.On("Run", []string{tt.selector, "show", "all"}).
				Return(storcli2Fixture(t, tt.fixture), nil)

			s := NewStorCLI2(mockRunner)

			volume, err := s.LogicalVolume(&logicalvolume.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{ID: 0},
				ID:           tt.id,
			})
			if tt.expectError {
				require.Error(t, err)

				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}

				assert.Nil(t, volume)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, volume)
			assert.Equal(t, tt.id, volume.ID)
			assert.Equal(t, 0, volume.CtrlMetadata.ID)
			assert.Equal(t, logicalvolume.RAIDLevel0, volume.RAIDLevel)
			assert.Equal(t, logicalvolume.LVStatusOptimal, volume.Status)
		})
	}
}

// TestStorCLI2LogicalVolumeEmptyList covers the not-found guard reached when
// the command succeeds but reports no virtual drive (distinct from a storcli2
// failure payload, which is rejected earlier by Decode). Per the User Guide,
// showing a nonexistent object reports success, and the "Virtual Drives"
// section may be present-but-empty or absent altogether.
func TestStorCLI2LogicalVolumeEmptyList(t *testing.T) {
	t.Parallel()

	payloads := map[string][]byte{
		"empty section": []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
			`"Response Data":{"Virtual Drives":[]}}]}`),
		"absent section": []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
			`"Response Data":{}}]}`),
	}

	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockRunner.On("Run", []string{"/c0/v1", "show", "all"}).Return(payload, nil)

			s := NewStorCLI2(mockRunner)

			volume, err := s.LogicalVolume(&logicalvolume.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{ID: 0},
				ID:           "1",
			})
			require.Error(t, err)
			require.ErrorContains(t, err, "not found")
			assert.Nil(t, volume)
		})
	}
}

// TestStorCLI2LogicalVolumesEmptyInventory pins the zero-VD contract: on a
// controller without virtual drives (e.g. all drives exposed as JBOD, the
// deployment this adapter targets), the inventory is empty rather than an
// error, whether the "Virtual Drives" section is empty or absent.
func TestStorCLI2LogicalVolumesEmptyInventory(t *testing.T) {
	t.Parallel()

	payloads := map[string][]byte{
		"empty section": []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
			`"Response Data":{"Virtual Drives":[]}}]}`),
		"absent section": []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
			`"Response Data":{}}]}`),
	}

	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockRunner.On("Run", []string{"/c0/vall", "show", "all"}).Return(payload, nil)

			s := NewStorCLI2(mockRunner)

			volumes, err := s.LogicalVolumes(&raidcontroller.Metadata{ID: 0})
			require.NoError(t, err)
			assert.Empty(t, volumes)
		})
	}
}

func TestStorCLI2LVStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state    string
		expected logicalvolume.LVStatus
	}{
		{"Optl", logicalvolume.LVStatusOptimal},
		{"Dgrd", logicalvolume.LVStatusDegraded},
		{"Pdgd", logicalvolume.LVStatusDegraded},
		{"Rec", logicalvolume.LVStatusDegraded},
		// The documented terminal VD state; the User Guide spells it both
		// "OfLn" (table legend) and "Ofln" (property description).
		{"OfLn", logicalvolume.LVStatusFailed},
		{"Ofln", logicalvolume.LVStatusFailed},
		{"Fail", logicalvolume.LVStatusFailed},
		{"weird", logicalvolume.LVStatusUnknown},
		{"", logicalvolume.LVStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, lvStatus(tt.state))
		})
	}
}

func TestStorCLI2ParseCacheOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cache       string
		expectedRP  logicalvolume.ReadPolicy
		expectedWP  logicalvolume.WritePolicy
		expectedIOP logicalvolume.IOPolicy
	}{
		{
			name:        "no read-ahead write-back",
			cache:       "NR,WB",
			expectedRP:  logicalvolume.ReadPolicyNoReadAhead,
			expectedWP:  logicalvolume.WritePolicyWriteBack,
			expectedIOP: logicalvolume.IOPolicyUnknown,
		},
		{
			name:        "read-ahead write-through direct",
			cache:       "R,WT,D",
			expectedRP:  logicalvolume.ReadPolicyReadAhead,
			expectedWP:  logicalvolume.WritePolicyWriteThrough,
			expectedIOP: logicalvolume.IOPolicyDirect,
		},
		{
			name:        "always write-back cached with spaces",
			cache:       "R, AWB, C",
			expectedRP:  logicalvolume.ReadPolicyReadAhead,
			expectedWP:  logicalvolume.WritePolicyAlwaysWriteBack,
			expectedIOP: logicalvolume.IOPolicyCached,
		},
		{
			name:        "empty leaves everything unknown",
			cache:       "",
			expectedRP:  logicalvolume.ReadPolicyUnknown,
			expectedWP:  logicalvolume.WritePolicyUnknown,
			expectedIOP: logicalvolume.IOPolicyUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			options := parseCacheOptions(tt.cache)
			assert.Equal(t, tt.expectedRP, options.ReadPolicy)
			assert.Equal(t, tt.expectedWP, options.WritePolicy)
			assert.Equal(t, tt.expectedIOP, options.IOPolicy)
		})
	}
}

func TestStorCLI2ParseVDID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dgvd        string
		expected    string
		expectError bool
	}{
		{name: "nominal", dgvd: "0/1", expected: "1"},
		{name: "multi digit", dgvd: "10/24", expected: "24"},
		{name: "missing separator", dgvd: "1", expectError: true},
		{name: "empty volume", dgvd: "0/", expectError: true},
		{name: "empty", dgvd: "", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := parseVDID(tt.dgvd)
			if tt.expectError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, id)
		})
	}
}

func TestStorCLI2PermanentPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		naaID    string
		expected string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"plain", "600062b22066d54069faf124ced57e62", "/dev/disk/by-id/wwn-0x600062b22066d54069faf124ced57e62"},
		{"trimmed", " abc ", "/dev/disk/by-id/wwn-0xabc"},
		// udev wwn- links are lowercase; the firmware is case-inconsistent.
		{"uppercase id is lowercased", "600062B22066D540", "/dev/disk/by-id/wwn-0x600062b22066d540"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, permanentPath(tt.naaID))
		})
	}
}
