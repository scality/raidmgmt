package smartarray_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/smartarray"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)

	return arguments.Get(0).([]byte), arguments.Error(1)
}

var testDataPath = "./testdata/"

type UnitTestSuite struct {
	suite.Suite

	s   *smartarray.SSACLI
	mcr *MockCommandRunner
}

// TestRunSuite runs the test suite. It's called by go test.
func TestRunSuite(t *testing.T) {
	suite.Run(t, &UnitTestSuite{})
}

// SetupTest sets up the test suite. It's called before each test.
func (s *UnitTestSuite) SetupTest() {
	// s.T().Parallel()
	s.mcr = &MockCommandRunner{}

	s.s = smartarray.NewSSACLI(s.mcr)
}

func mockOutput(filename string) []byte {
	output, err := os.ReadFile(testDataPath + filename + ".txt")
	if err != nil {
		panic(err)
	}

	return output
}

// TestControllers tests the Controllers method.
func (s *UnitTestSuite) TestControllers() {
	tests := []struct {
		name          string
		mocking       []byte
		expectedError bool
	}{
		{
			name:          "nominal case",
			mocking:       mockOutput("controller/show/all_detail"),
			expectedError: false,
		},
		// TODO add more test cases
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.mcr.On("Run", []string{"controller", "all", "show", "detail"}).Return(tt.mocking, nil)

			controllers, err := s.s.Controllers()

			if tt.expectedError {
				s.Require().Error(err)
				s.Require().Nil(controllers)
			} else {
				s.Require().NoError(err)
				s.Require().NotEmpty(controllers)

				for _, controller := range controllers {
					s.Require().NotEmpty(controller.Name)
					s.Require().NotEmpty(controller.Serial)
				}

				for _, controller := range controllers {
					s.T().Logf("Controller %d: %+v", controller.ID, controller)
				}
			}
		})
	}
}

func (s *UnitTestSuite) TestController() {
	tests := []struct {
		name          string
		mocking       []byte
		id            int
		expectedError bool
	}{
		{
			name:          "nominal case",
			mocking:       mockOutput("controller/show/slot_0"),
			id:            0,
			expectedError: false,
		},
		// TODO add more test cases
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.mcr.On("Run", []string{"controller", "slot=" + strconv.Itoa(tt.id), "show", "detail"}).Return(tt.mocking, nil)

			metadata := &raidcontroller.Metadata{
				ID: tt.id,
			}

			controller, err := s.s.Controller(metadata)

			if tt.expectedError {
				s.Require().Error(err)
				s.Require().Nil(controller)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(controller)

				s.Assert().Equal(metadata.ID, controller.ID)
				s.Assert().Equal("P816i-a SR Gen10", controller.Name)
				s.Assert().Equal("PWXLA0CRHF10FM", controller.Serial)
			}
		})
	}
}

func (s *UnitTestSuite) TestPhysicalDrives() {
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
		// TODO add more test cases
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.mcr.On("Run", []string{
				"controller",
				"slot=" + strconv.Itoa(tt.id),
				"physicaldrive",
				"all",
				"show",
				"detail",
			}).Return(tt.mocking, nil)

			metadata := &raidcontroller.Metadata{
				ID: tt.id,
			}

			physicalDrives, err := s.s.PhysicalDrives(metadata)

			if tt.expectedError {
				s.Require().Error(err)
				s.Require().Nil(physicalDrives)
			} else {
				s.Require().NoError(err)
				s.Require().NotEmpty(physicalDrives)
				s.Require().Len(physicalDrives, 14)

				seen := make(map[string]bool)

				for _, pd := range physicalDrives {
					s.Require().NotEmpty(pd.Serial)
					s.Require().NotEmpty(pd.Model)
					s.Require().NotEmpty(pd.Vendor)

					if seen[pd.Serial] {
						s.T().Errorf("Duplicate physical drive: %s", pd.Serial)
					} else {
						seen[pd.Serial] = true
					}

					s.T().Logf("Physical Drive %s: %+v", pd.Serial, pd)
				}
			}
		})
	}
}

func (s *UnitTestSuite) TestPhysicalDrive() {
	tests := []struct {
		name          string
		mocking       []byte
		metadata      *physicaldrive.Metadata
		expected      *physicaldrive.PhysicalDrive
		expectedError bool
	}{
		{
			name:    "nominal case",
			mocking: mockOutput("physicaldrives/4I:6:1_detail"),
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				Slot: &physicaldrive.Slot{
					Port:      "4I",
					Enclosure: "6",
					Bay:       "1",
				},
			},
			expected: &physicaldrive.PhysicalDrive{
				Metadata: &physicaldrive.Metadata{
					CtrlMetadata: &raidcontroller.Metadata{
						ID: 0,
					},
					Slot: &physicaldrive.Slot{
						Port:      "4I",
						Enclosure: "6",
						Bay:       "1",
					},
				},
				Vendor: "HPE",
				Model:  "MO000800JXBEV",
				Serial: "W2X0751Y",
				ID:     "5000CCA0B8712794",
				Size:   858993459200,
				Status: physicaldrive.PDStatusUsed,
			},
			expectedError: false,
		},
		// TODO add more test cases
	}

	formatSlot := func(slot *physicaldrive.Slot) string {
		if slot == nil {
			return ""
		}

		return slot.Port + ":" + slot.Enclosure + ":" + slot.Bay
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			args := []string{
				"controller",
				"slot=" + fmt.Sprint(rune(tt.metadata.CtrlMetadata.ID)),
				"physicaldrive",
				formatSlot(tt.metadata.Slot),
				"show",
				"detail",
			}
			s.T().Logf("args: %v", args)

			s.mcr.On("Run", args).Return(tt.mocking, nil)

			metadata := &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: tt.expected.CtrlMetadata.ID,
				},
				Slot: &physicaldrive.Slot{
					Port:      "4I",
					Enclosure: "6",
					Bay:       "1",
				},
			}

			physicalDrive, err := s.s.PhysicalDrive(metadata)

			if tt.expectedError {
				s.Require().Error(err)
				s.Require().Nil(physicalDrive)
			} else {
				s.Require().NoError(err)
				s.Require().NotEmpty(physicalDrive)

				s.Equal(tt.expected.ID, physicalDrive.ID)
				s.Equal(tt.expected.Serial, physicalDrive.Serial)
				s.Equal(tt.expected.Model, physicalDrive.Model)
				s.Equal(tt.expected.Vendor, physicalDrive.Vendor)
			}

			s.T().Logf("Physical Drive %s: %+v", physicalDrive.Serial, physicalDrive)
		})
	}
}

func (s *UnitTestSuite) TestLogicalVolumes() {
	mapMockingStatusNominal := map[string][]byte{
		"1":  mockOutput("logicalvolumes/show/status/1"),
		"2":  mockOutput("logicalvolumes/show/status/2"),
		"3":  mockOutput("logicalvolumes/show/status/3"),
		"4":  mockOutput("logicalvolumes/show/status/4"),
		"5":  mockOutput("logicalvolumes/show/status/5"),
		"6":  mockOutput("logicalvolumes/show/status/6"),
		"7":  mockOutput("logicalvolumes/show/status/7"),
		"8":  mockOutput("logicalvolumes/show/status/8"),
		"9":  mockOutput("logicalvolumes/show/status/9"),
		"10": mockOutput("logicalvolumes/show/status/10"),
		"11": mockOutput("logicalvolumes/show/status/11"),
	}

	tests := []struct {
		name          string
		mockingDetail []byte
		mockingStatus map[string][]byte
		id            int
		expectedError bool
	}{
		{
			name:          "nominal case",
			mockingDetail: mockOutput("logicalvolumes/show/detail/all"),
			mockingStatus: mapMockingStatusNominal,
			id:            0,
			expectedError: false,
		},
		// TODO add more test cases
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.mcr.On("Run", []string{
				"controller",
				"slot=" + strconv.Itoa(tt.id),
				"logicaldrive",
				"all",
				"show",
				"detail",
			}).Return(tt.mockingDetail, nil)

			s.mcr.On("Run", []string{
				"controller",
				"slot=" + strconv.Itoa(tt.id),
				"show",
				"config",
			}).Return(mockOutput("controller/show/config"), nil)

			metadata := &raidcontroller.Metadata{
				ID: tt.id,
			}

			logicalVolumes, err := s.s.LogicalVolumes(metadata)

			seen := make(map[string]bool)

			if tt.expectedError {
				s.Require().Error(err)
				s.Require().Nil(logicalVolumes)
			} else {
				s.Require().NoError(err)
				s.Require().NotEmpty(logicalVolumes)
				s.Require().Len(logicalVolumes, 11)
				s.Require().Equal(logicalvolume.RAIDLevel1, logicalVolumes[0].RAIDLevel)
				s.Require().Equal(logicalvolume.RAIDLevel0, logicalVolumes[1].RAIDLevel)

				for _, lv := range logicalVolumes {
					if seen[lv.ID] {
						s.T().Errorf("Duplicate logical drive: %s", lv.ID)
					} else {
						seen[lv.ID] = true
					}

					s.T().Logf("Logical Drive %s: %+v", lv.ID, lv)
				}
			}
		})
	}
}

func (s *UnitTestSuite) TestLogicalVolume() {
	tests := []struct {
		name          string
		mockingDetail []byte
		metadata      *logicalvolume.Metadata
		expected      *logicalvolume.LogicalVolume
		expectedError bool
	}{
		{
			name:          "nominal case",
			mockingDetail: mockOutput("logicalvolumes/show/detail/1"),
			metadata: &logicalvolume.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "1",
			},
			expected: &logicalvolume.LogicalVolume{
				Metadata: &logicalvolume.Metadata{
					CtrlMetadata: &raidcontroller.Metadata{
						ID: 0,
					},
					ID: "1",
				},
			},
			expectedError: false,
		},
		// TODO add more test cases
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.mcr.On("Run", []string{
				"controller",
				"slot=" + strconv.Itoa(tt.metadata.CtrlMetadata.ID),
				"show",
				"config",
			}).Return(mockOutput("controller/show/config"), nil)

			s.mcr.On("Run", []string{
				"controller",
				"slot=" + strconv.Itoa(tt.metadata.CtrlMetadata.ID),
				"logicaldrive",
				tt.metadata.ID,
				"show",
				"detail",
			}).Return(tt.mockingDetail, nil)

			logicalVolume, err := s.s.LogicalVolume(tt.metadata)

			if tt.expectedError {
				s.Require().Error(err)
				s.Require().Nil(logicalVolume)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(logicalVolume)

				s.Assert().Equal(tt.metadata.ID, logicalVolume.ID)
				s.Assert().Equal(tt.metadata.CtrlMetadata.ID, logicalVolume.CtrlMetadata.ID)
			}
		})
	}
}
