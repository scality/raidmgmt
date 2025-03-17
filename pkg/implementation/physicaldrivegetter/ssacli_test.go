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
