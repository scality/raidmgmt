package physicaldrivegetter

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/utils"
)

// storcli2Fixture reads a storcli2 JSON fixture from the package testdata.
func storcli2Fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile("testdata/storcli2/" + name)
	require.NoError(t, err)

	return data
}

func TestStorCLI2PhysicalDrives(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/eall/sall", "show", "all"}).
		Return(storcli2Fixture(t, "show/all.json"), nil)

	s := NewStorCLI2(mockRunner)

	drives, err := s.PhysicalDrives(&raidcontroller.Metadata{ID: 0})
	require.NoError(t, err)
	require.Len(t, drives, 24)

	expectedSize, err := utils.ConvertSizeBytes("9.094 TiB")
	require.NoError(t, err)

	first := drives[0]
	assert.Equal(t, "306:0", first.ID)
	assert.Equal(t, 0, first.CtrlMetadata.ID)
	assert.Equal(t, "306", first.Slot.Enclosure)
	assert.Equal(t, "0", first.Slot.Bay)
	assert.Equal(t, "SEAGATE", first.Vendor)
	assert.Equal(t, "ST10000NM018B", first.Model)
	assert.Equal(t, "WP00MLCA0000E2426EZU", first.Serial)
	assert.Equal(t, "0x5000C500EF7DE7D4", first.WWN)
	assert.Equal(t, expectedSize, first.Size)
	assert.Equal(t, physicaldrive.DiskTypeHDD, first.Type)
	assert.Equal(t, physicaldrive.PDStatusUsed, first.Status)
	assert.False(t, first.JBOD)
}

func TestStorCLI2PhysicalDrive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		id             string
		selector       string
		fixture        string
		expectError    bool
		expectedStatus physicaldrive.PDStatus
	}{
		{
			name:           "configured drive",
			id:             "306:0",
			selector:       "/c0/e306/s0",
			fixture:        "show/e306s0.json",
			expectedStatus: physicaldrive.PDStatusUsed,
		},
		{
			name:           "unconfigured good drive",
			id:             "320:11",
			selector:       "/c0/e320/s11",
			fixture:        "show/e320s11_UGood.json",
			expectedStatus: physicaldrive.PDStatusUnassignedGood,
		},
		{
			name:        "drive not found",
			id:          "306:99",
			selector:    "/c0/e306/s99",
			fixture:     "show/e306s99_invalid.json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockRunner.On("Run", []string{tt.selector, "show", "all"}).
				Return(storcli2Fixture(t, tt.fixture), nil)

			s := NewStorCLI2(mockRunner)

			drive, err := s.PhysicalDrive(&physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{ID: 0},
				ID:           tt.id,
			})
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, drive)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, drive)
			assert.Equal(t, tt.id, drive.ID)
			assert.Equal(t, tt.expectedStatus, drive.Status)
			assert.Equal(t, physicaldrive.DiskTypeHDD, drive.Type)
		})
	}
}

// TestStorCLI2PhysicalDriveEmptyList covers the not-found guard reached when
// the command succeeds but reports no drive (distinct from a storcli2 failure
// payload, which is rejected earlier by Decode). Per the User Guide, showing a
// nonexistent object reports success, and the "Drives List" section may be
// present-but-empty or absent altogether.
func TestStorCLI2PhysicalDriveEmptyList(t *testing.T) {
	t.Parallel()

	payloads := map[string][]byte{
		"empty section": []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
			`"Response Data":{"Drives List":[]}}]}`),
		"absent section": []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
			`"Response Data":{}}]}`),
	}

	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockRunner.On("Run", []string{"/c0/e306/s0", "show", "all"}).Return(payload, nil)

			s := NewStorCLI2(mockRunner)

			drive, err := s.PhysicalDrive(&physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{ID: 0},
				ID:           "306:0",
			})
			require.Error(t, err)
			require.ErrorContains(t, err, "not found")
			assert.Nil(t, drive)
		})
	}
}

// TestStorCLI2PhysicalDrivesEmptyInventory pins the empty-inventory contract:
// a controller without drives yields an empty list rather than an error,
// whether the "Drives List" section is empty or absent.
func TestStorCLI2PhysicalDrivesEmptyInventory(t *testing.T) {
	t.Parallel()

	payloads := map[string][]byte{
		"empty section": []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
			`"Response Data":{"Drives List":[]}}]}`),
		"absent section": []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
			`"Response Data":{}}]}`),
	}

	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockRunner.On("Run", []string{"/c0/eall/sall", "show", "all"}).Return(payload, nil)

			s := NewStorCLI2(mockRunner)

			drives, err := s.PhysicalDrives(&raidcontroller.Metadata{ID: 0})
			require.NoError(t, err)
			assert.Empty(t, drives)
		})
	}
}

// TestStorCLI2PhysicalDrivesJBOD pins the JBOD mapping at the entity level
// with a synthetic payload (the captured fixtures contain no JBOD drive): a
// JBOD drive that is not functioning (here "Missing") keeps JBOD=true, maps to
// PDStatusFailed, and must not have its device paths resolved — ComputePaths
// would fail on its absent device node and abort the whole inventory.
func TestStorCLI2PhysicalDrivesJBOD(t *testing.T) {
	t.Parallel()

	const payload = `{"Controllers":[{"Command Status":{"Status":"Success"},` +
		`"Response Data":{"Drives List":[{` +
		`"Drive Information":{"EID:Slt":"306:4","Model":"ST10000NM018B","Med":"HDD",` +
		`"Size":"9.094 TiB","State":"JBOD","Status":"Missing"},` +
		`"Drive Detailed Information":{"Vendor":"SEAGATE","Serial Number":"WP00MLCA",` +
		`"WWN":"5000C500EF7DE7D4"}}]}}]}`

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/eall/sall", "show", "all"}).Return([]byte(payload), nil)

	s := NewStorCLI2(mockRunner)

	drives, err := s.PhysicalDrives(&raidcontroller.Metadata{ID: 0})
	require.NoError(t, err)
	require.Len(t, drives, 1)
	assert.True(t, drives[0].JBOD)
	assert.Equal(t, physicaldrive.PDStatusFailed, drives[0].Status)
	assert.Empty(t, drives[0].DevicePath)
	assert.Empty(t, drives[0].PermanentPath)
}

func TestStorCLI2PDStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		state    string
		status   string
		expected physicaldrive.PDStatus
	}{
		{"configured online", "Conf", "Online", physicaldrive.PDStatusUsed},
		{"configured rebuilding", "Conf", "Rebuild", physicaldrive.PDStatusUsed},
		{"configured degraded variant", "ConfDgrd", "Online", physicaldrive.PDStatusUsed},
		{"jbod", "JBOD", "Online", physicaldrive.PDStatusUsed},
		{"jbod sanitize variant", "JBODSntz", "Online", physicaldrive.PDStatusUsed},
		{"shielded jbod", "Shld", "Online", physicaldrive.PDStatusUsed},
		{"global hot spare", "GHS", "Online", physicaldrive.PDStatusUsed},
		{"dedicated hot spare shielded", "DHSShld", "Online", physicaldrive.PDStatusUsed},
		{"configured being replaced", "Conf", "Replace", physicaldrive.PDStatusUsed},
		{"unconfigured good", "UConf", "Good", physicaldrive.PDStatusUnassignedGood},
		{"unconfigured shielded", "UConfShld", "Good", physicaldrive.PDStatusUnassignedGood},
		{"unconfigured bad", "UConf", "Bad", physicaldrive.PDStatusUnassignedBad},
		{"unconfigured unusable", "UConf", "Unusable", physicaldrive.PDStatusUnassignedBad},
		{"unconfigured unsupported", "UConfUnsp", "Good", physicaldrive.PDStatusUnassignedBad},
		// A "Failed", "Offline" or "Missing" status wins over any state: a drive
		// that is not functioning must never be reported as in use or available.
		{"configured failed", "Conf", "Failed", physicaldrive.PDStatusFailed},
		{"configured offline", "Conf", "Offline", physicaldrive.PDStatusFailed},
		{"jbod missing", "JBOD", "Missing", physicaldrive.PDStatusFailed},
		{"unconfigured failed", "UConf", "Failed", physicaldrive.PDStatusFailed},
		{"failed state guard", "Failed", "Failed", physicaldrive.PDStatusFailed},
		{"unusable", "Unusbl", "Online", physicaldrive.PDStatusUnknown},
		{"unknown", "", "", physicaldrive.PDStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, pdStatus(tt.state, tt.status))
		})
	}
}

func TestStorCLI2IsJBODState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state    string
		expected bool
	}{
		{"JBOD", true},
		{"JBODDgrd", true},
		{"JBODSntz", true},
		{"Shld", true},
		{"Conf", false},
		{"UConf", false},
		{"GHSShld", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isJBODState(tt.state))
		})
	}
}

func TestStorCLI2DiskType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		intf     string
		med      string
		expected physicaldrive.DiskType
	}{
		{"sas hdd", "SAS", "HDD", physicaldrive.DiskTypeHDD},
		{"sata ssd", "SATA", "SSD", physicaldrive.DiskTypeSSD},
		// storcli2 reports NVMe drives with "SSD" media and the transport in
		// "Intf" (User Guide sample: Intf "NVMe", Med "SSD").
		{"nvme reported as ssd media", "NVMe", "SSD", physicaldrive.DiskTypeNVMe},
		{"nvme lowercase", "nvme", "ssd", physicaldrive.DiskTypeNVMe},
		{"nvme media guard", "", "NVMe", physicaldrive.DiskTypeNVMe},
		{"unknown", "SAS", "weird", physicaldrive.DiskTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, diskType(tt.intf, tt.med))
		})
	}
}

func TestStorCLI2FormatWWN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wwn      string
		expected string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"plain", "5000C500EF7DE7D4", "0x5000C500EF7DE7D4"},
		{"trimmed", " ABC ", "0xABC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatWWN(tt.wwn))
		})
	}
}

func TestStorCLI2SelectorPD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		id          string
		expected    string
		expectError bool
	}{
		{name: "with enclosure", id: "306:0", expected: "/c0/e306/s0"},
		{name: "without enclosure", id: "5", expected: "/c0/s5"},
		{name: "empty id", id: "", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selector, err := storcli2SelectorPD(&physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{ID: 0},
				ID:           tt.id,
			})
			if tt.expectError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, selector)
		})
	}
}
