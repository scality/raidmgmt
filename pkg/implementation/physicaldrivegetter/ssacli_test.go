package physicaldrivegetter

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
)

type MockCommandRunner struct {
	mock.Mock
}

var testDataPath = "./"

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)

	return arguments.Get(0).([]byte), arguments.Error(1)
}

func TestSSACLIPhysicalDrives(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	s := &SSACLI{
		SSACLI: mockRunner,
		LSBLK:  mockRunner,
	}

	tests := []struct {
		name          string
		mocking       []byte
		id            int
		expectedError bool
	}{
		{
			name:          "nominal case",
			mocking:       mockOutput("physicaldrives/all_detail"),
			id:            0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup command runner expectations
			mockRunner.On("Run", []string{
				"controller",
				"slot=" + strconv.Itoa(tt.id),
				"physicaldrive",
				"all",
				"show",
				"detail",
			}).Return(tt.mocking, nil)

			// Mock the lsblk call for disk device path
			lsblkOutput := []byte(`NAME ROTA SIZE TYPE TRAN MOUNTPOINT FSTYPE PARTTYPE
/dev/sda    0 858993459200 disk sata                    `)
			mockRunner.On("Run", mock.AnythingOfType("[]string")).Return(lsblkOutput, nil)

			metadata := &raidcontroller.Metadata{
				ID: tt.id,
			}

			physicalDrives, err := s.PhysicalDrives(metadata)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, physicalDrives)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, physicalDrives)
				assert.Len(t, physicalDrives, 14)

				seen := make(map[string]bool)

				for _, pd := range physicalDrives {
					assert.NotEmpty(t, pd.Serial)
					assert.NotEmpty(t, pd.Model)
					assert.NotEmpty(t, pd.Vendor)

					if seen[pd.Serial] {
						t.Errorf("Duplicate physical drive: %s", pd.Serial)
					} else {
						seen[pd.Serial] = true
					}
				}
			}
		})
	}
}

func TestSSCALIPhysicalDrive(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	s := &SSACLI{
		SSACLI: mockRunner,
		LSBLK:  mockRunner,
	}

	tests := []struct {
		name          string
		mocking       []byte
		metadata      *physicaldrive.Metadata
		expected      *physicaldrive.PhysicalDrive
		expectedError bool
	}{
		{
			name:    "nominal case",
			mocking: mockOutput("physicaldrives/4I.6.1_detail"),
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "4I:6:1",
			},
			expected: &physicaldrive.PhysicalDrive{
				Metadata: &physicaldrive.Metadata{
					CtrlMetadata: &raidcontroller.Metadata{
						ID: 0,
					},
					ID: "4I:6:1",
				},
				Slot: &physicaldrive.Slot{
					Port:      "4I",
					Enclosure: "6",
					Bay:       "1",
				},
				Vendor: "HPE",
				Model:  "MO000800JXBEV",
				Serial: "W2X0751Y",
				Size:   858993459200,
				Status: physicaldrive.PDStatusUsed,
			},
			expectedError: false,
		},
		// TODO add more test cases
	}

	for _, tt := range tests {
		mockRunner.On("Run", []string{
			"controller",
			"slot=" + strconv.Itoa(tt.metadata.CtrlMetadata.ID),
			"physicaldrive",
			tt.metadata.ID,
			"show",
			"detail",
		}).Return(tt.mocking, nil)

		lsblkOutput := []byte(`NAME ROTA SIZE TYPE TRAN MOUNTPOINT FSTYPE PARTTYPE
/dev/sda    0 858993459200 disk sata                    `)
		mockRunner.On("Run", mock.AnythingOfType("[]string")).Return(lsblkOutput, nil)

		metadata := &physicaldrive.Metadata{
			CtrlMetadata: &raidcontroller.Metadata{
				ID: tt.metadata.CtrlMetadata.ID,
			},
			ID: "4I:6:1",
		}

		physicalDrive, err := s.PhysicalDrive(metadata)

		if tt.expectedError {
			assert.Error(t, err)
			assert.Nil(t, physicalDrive)
		} else {
			assert.NoError(t, err)
			assert.NotEmpty(t, physicalDrive)

			assert.Equal(t, tt.expected.ID, physicalDrive.ID)
			assert.Equal(t, tt.expected.Serial, physicalDrive.Serial)
			assert.Equal(t, tt.expected.Model, physicalDrive.Model)
			assert.Equal(t, tt.expected.Vendor, physicalDrive.Vendor)
		}
	}
}

func mockOutput(filename string) []byte {
	output, err := os.ReadFile(testDataPath + filename + ".txt")
	if err != nil {
		panic(err)
	}

	return output
}

func TestSSACLIPhysicalDriveStatus(t *testing.T) {
	// lsblk on a drive that carries nothing: no mountpoint, no filesystem, no
	// partition type.
	emptyDevice := []byte(`NAME ROTA SIZE TYPE TRAN MOUNTPOINT FSTYPE PARTTYPE
/dev/sda    0 858993459200 disk sata                    `)

	tests := []struct {
		name              string
		mocking           []byte
		lsblk             []byte
		metadata          *physicaldrive.Metadata
		expected          *physicaldrive.PhysicalDrive
		expectedAvailable bool
		expectedError     bool
	}{
		{
			// ssacli answers "Status: OK" for a healthy drive whether or not it
			// belongs to an array, so "Drive Type: Unassigned Drive" is what
			// makes the drive available. Without it every drive came back used
			// and CreateLV rejected every request with "unavailable drives".
			name:    "unassigned drive is available",
			mocking: mockOutput("physicaldrives/1I.1.1_detail"),
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "1I:1:1",
			},
			expected: &physicaldrive.PhysicalDrive{
				Status: physicaldrive.PDStatusUnassignedGood,
				Reason: "OK",
			},
			expectedAvailable: true,
			expectedError:     false,
		},
		{
			// A drive serving an array stays used.
			name:    "data drive is used",
			mocking: mockOutput("physicaldrives/4I.6.1_detail"),
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "4I:6:1",
			},
			expected: &physicaldrive.PhysicalDrive{
				Status: physicaldrive.PDStatusUsed,
				Reason: "OK",
			},
			expectedAvailable: false,
			expectedError:     false,
		},
		{
			// ssacli reports a predictive (SMART) failure on a free drive too,
			// and such a drive must not be offered for a new volume.
			name:    "predictive failure on an unassigned drive stays used",
			mocking: mockOutput("physicaldrives/predictive_failure_unassigned_detail"),
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "1I:1:1",
			},
			expected: &physicaldrive.PhysicalDrive{
				Status: physicaldrive.PDStatusUsed,
				Reason: "Predictive Failure",
			},
			expectedAvailable: false,
			expectedError:     false,
		},
		{
			// A free drive that still carries data is used: the lsblk verdict
			// wins over "Unassigned Drive", whatever the order of the fields.
			name:    "unassigned drive carrying a filesystem is used",
			mocking: mockOutput("physicaldrives/1I.1.1_detail"),
			lsblk: []byte(`NAME ROTA SIZE TYPE TRAN MOUNTPOINT FSTYPE PARTTYPE
/dev/sda    0 858993459200 disk sata               xfs      `),
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "1I:1:1",
			},
			expected: &physicaldrive.PhysicalDrive{
				Status: physicaldrive.PDStatusUsed,
				Reason: "OK",
			},
			expectedAvailable: false,
			expectedError:     false,
		},
		{
			// "Unassigned Drive" never promotes a drive ssacli called failed.
			name:    "failed unassigned drive stays failed",
			mocking: mockOutput("physicaldrives/failed_unassigned_detail"),
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "1I:1:1",
			},
			expected: &physicaldrive.PhysicalDrive{
				Status: physicaldrive.PDStatusFailed,
				Reason: "Failed",
			},
			expectedAvailable: false,
			expectedError:     false,
		},
		{
			// "Predictive Failure" must not abort the inventory: the drive is
			// still online and serving its array, so it is surfaced as used
			// with the raw ssacli label kept in Reason.
			name:    "predictive failure",
			mocking: mockOutput("physicaldrives/predictive_failure_detail"),
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "3I:3:2",
			},
			expected: &physicaldrive.PhysicalDrive{
				Status: physicaldrive.PDStatusUsed,
				Reason: "Predictive Failure",
			},
			expectedAvailable: false,
			expectedError:     false,
		},
		{
			// An unmodeled status (here "Rebuilding") soft-fails to
			// PDStatusUnknown rather than returning an error, so a future
			// ssacli label can never again take down the whole inventory call.
			name:    "unknown status soft-fails",
			mocking: mockOutput("physicaldrives/unknown_status_detail"),
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "3I:3:2",
			},
			expected: &physicaldrive.PhysicalDrive{
				Status: physicaldrive.PDStatusUnknown,
				Reason: "Rebuilding",
			},
			expectedAvailable: false,
			expectedError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockCommandRunner)

			s := &SSACLI{
				SSACLI: mockRunner,
				LSBLK:  mockRunner,
			}

			mockRunner.On("Run", []string{
				"controller",
				"slot=" + strconv.Itoa(tt.metadata.CtrlMetadata.ID),
				"physicaldrive",
				tt.metadata.ID,
				"show",
				"detail",
			}).Return(tt.mocking, nil)

			lsblkOutput := tt.lsblk
			if lsblkOutput == nil {
				lsblkOutput = emptyDevice
			}

			mockRunner.On("Run", mock.AnythingOfType("[]string")).Return(lsblkOutput, nil)

			physicalDrive, err := s.PhysicalDrive(tt.metadata)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, physicalDrive)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, physicalDrive)
				assert.Equal(t, tt.expected.Status, physicalDrive.Status)
				assert.Equal(t, tt.expected.Reason, physicalDrive.Reason)
				assert.Equal(t, tt.expectedAvailable, physicalDrive.IsAvailable())
			}
		})
	}
}
