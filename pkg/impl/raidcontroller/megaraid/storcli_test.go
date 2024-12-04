package megaraid_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/megaraid"
	"github.com/scality/raidmgmt/megaraid/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var path = "./testdata/"

type UnitTestSuite struct {
	suite.Suite

	m             *megaraid.Adapter
	cmdRunnerMock *mocks.Runner
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, &UnitTestSuite{})
}

func (s *UnitTestSuite) SetupTest() {
	s.cmdRunnerMock = mocks.NewRunner(s.T())
	s.m = megaraid.New(s.cmdRunnerMock)
}

func mockOutput(filename string) *megaraid.CmdOutput {
	file, err := os.Open(path + filename + ".json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var data megaraid.CmdOutput

	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		panic(err)
	}

	return &data
}

// mockError checks if the command was successful
// It's copied from the megaraid package.
func mockError(out *megaraid.CmdOutput) error {
	// Check if there are any controllers
	if len(out.Controllers) == 0 {
		return megaraid.ErrNoControllersFound
	}

	// Check if the command was successful
	for _, controller := range out.Controllers {
		commandStatus := controller.CommandStatus

		if commandStatus.Status != "Success" {
			return megaraid.ParseError(commandStatus)
		}
	}

	return nil
}

func mockReturn(filename string) (*megaraid.CmdOutput, error) {
	out := mockOutput(filename)
	return out, mockError(out)
}

func (s *UnitTestSuite) controllersMockCalls() {
	s.cmdRunnerMock.On("Run", mock.AnythingOfType("[]string")).Return(
		func(args []string) (*megaraid.CmdOutput, error) {
			if args[0] == "show" {
				return mockReturn("controllers/all")
			}

			if args[0] == "/c0" {
				return mockReturn("controllers/c0")
			}

			return nil, fmt.Errorf("unexpected call to Run with args: %v", args)
		})
}

func (s *UnitTestSuite) setupMockCalls() {
	s.cmdRunnerMock.On("Run", mock.AnythingOfType("[]string")).Return(
		func(args []string) (*megaraid.CmdOutput, error) {
			if args[0] == "/c0" {
				return mockReturn("controllers/c0")
			}

			args0Split := strings.Split(args[0], "/")

			// physical drives calls
			if args0Split[2] == "e251" {
				filename := fmt.Sprintf("physicaldrives/show/e251%s", args0Split[3])
				return mockReturn(filename)
			}

			// logical volumes calls
			filename := fmt.Sprintf("logicalvolumes/show/%s", args0Split[2])

			return mockReturn(filename)
		})
}

func (s *UnitTestSuite) logicalVolumesMockCalls() {
	s.setupMockCalls()

	s.cmdRunnerMock.On("Run", mock.AnythingOfType("[]string")).Return(
		func(args []string) (*megaraid.CmdOutput, error) {
			args0Split := strings.Split(args[0], "/")

			filename := fmt.Sprintf("logicalvolumes/show/%s", args0Split[2])

			return mockReturn(filename)
		})
}

func (s *UnitTestSuite) TestControllers() {
	s.controllersMockCalls()
	controllers, err := s.m.Controllers()

	s.NoError(err)
	s.Len(controllers, 1)
	s.Equal("MegaRAID 9560-8i 4GB", controllers[0].Name)
	s.Equal("SKC5120859", controllers[0].Serial)
}

func (s *UnitTestSuite) TestPhysicalDrives() {
	s.setupMockCalls()

	metadata := &raidcontroller.Metadata{
		ID: "0",
	}

	pDrives, err := s.m.PhysicalDrives(metadata)

	s.NoError(err)
	s.Len(pDrives, 12)

	expectedSize := uint64(17999005346693)
	expectedStatus := physicaldrive.PDStatusUsed
	expectedType := physicaldrive.DiskTypeHDD

	pd0 := pDrives[0]
	s.Equal("0", pd0.ID)
	s.Equal("ZVT1LT8M0000G214048L", pd0.Serial)
	s.Equal(expectedSize, pd0.Size)
	s.Equal(expectedStatus, pd0.Status)
	s.Equal(expectedType, pd0.Type)

	pd5 := pDrives[5]
	s.Equal("5", pd5.ID)
	s.Equal("ZVT1VDZL0000C2438NHV", pd5.Serial)
	s.Equal(expectedSize, pd5.Size)
	s.Equal(expectedStatus, pd5.Status)
	s.Equal(expectedType, pd5.Type)
}

func (s *UnitTestSuite) TestLogicalVolumes() {
	s.setupMockCalls()

	metadata := &raidcontroller.Metadata{
		ID: "0",
	}

	lVolumes, err := s.m.LogicalVolumes(metadata)
	s.NoError(err)
	s.Len(lVolumes, 12)

	expectedRAIDLevel := logicalvolume.RAIDLevel0
	expectedStatus := logicalvolume.LVStatusOptimal
	expectedCacheOptions := &logicalvolume.CacheOptions{
		ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
		WritePolicy: logicalvolume.WritePolicyWriteThrough,
		IOPolicy:    logicalvolume.IOPolicyDirect,
	}

	lv0 := lVolumes[0]
	s.Equal("228", lv0.ID)
	s.Equal(expectedRAIDLevel, lv0.RAIDLevel)
	s.Len(lv0.PhysicalDrives, 1)
	s.Equal(expectedStatus, lv0.Status)
	s.Equal(expectedCacheOptions, lv0.CacheOptions)

	lv5 := lVolumes[5]
	s.Equal("233", lv5.ID)
	s.Equal(expectedRAIDLevel, lv5.RAIDLevel)
	s.Len(lv5.PhysicalDrives, 1)
	s.Equal(expectedStatus, lv5.Status)
	s.Equal(expectedCacheOptions, lv5.CacheOptions)
}

func (s *UnitTestSuite) TestEnableJBODFail() {
	metadata := &physicaldrive.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{
			ID: "0",
		},
		Slot: &physicaldrive.Slot{
			Enclosure: 251,
			Bay:       6,
		},
	}

	s.cmdRunnerMock.On("Run", []string{"/c0/e251/s6", "set", "jbod"}).
		Return(mockReturn("physicaldrives/jbod/enable/fail"))

	err := s.m.EnableJBOD(metadata)

	s.Error(err)
}

func (s *UnitTestSuite) TestDisableJBODFail() {
	s.cmdRunnerMock.On("Run", []string{"/c0/e251/s6", "delete", "jbod"}).
		Return(mockReturn("physicaldrives/jbod/disable/fail"))

	metadata := &physicaldrive.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{
			ID: "0",
		},
		Slot: &physicaldrive.Slot{
			Enclosure: 251,
			Bay:       6,
		},
	}

	err := s.m.DisableJBOD(metadata)

	s.Error(err)
}

func (s *UnitTestSuite) TestSetLVCacheOptionsSuccess() {
	s.logicalVolumesMockCalls()

	ctrlMetadata := &raidcontroller.Metadata{
		ID: "0",
	}

	// IO policy is not in the command since it's the same
	// Only different cache options are in the command
	s.cmdRunnerMock.On("Run", []string{"/c0/v228", "set", "rdcache=nra", "wrcache=wb"}).
		Return(mockReturn("logicalvolumes/cacheoptions/success"))

	lvMetadata := &logicalvolume.Metadata{
		CtrlMetadata: ctrlMetadata,
		ID:           "228",
	}

	err := s.m.SetLVCacheOptions(lvMetadata, &logicalvolume.CacheOptions{
		ReadPolicy:  logicalvolume.ReadPolicyNoReadAhead,
		WritePolicy: logicalvolume.WritePolicyWriteBack,
		IOPolicy:    logicalvolume.IOPolicyDirect,
	})

	s.NoError(err)
}

func (s *UnitTestSuite) TestSetLVCacheOptionsSameOptions() {
	s.logicalVolumesMockCalls()

	ctrlMetadata := &raidcontroller.Metadata{
		ID: "0",
	}

	lVolumes, _ := s.m.LogicalVolumes(ctrlMetadata)

	lvMetadata := &logicalvolume.Metadata{
		CtrlMetadata: ctrlMetadata,
		ID:           lVolumes[0].ID,
	}

	// Set the same cache options
	err := s.m.SetLVCacheOptions(lvMetadata, &logicalvolume.CacheOptions{
		ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
		WritePolicy: logicalvolume.WritePolicyWriteThrough,
		IOPolicy:    logicalvolume.IOPolicyDirect,
	})

	s.Error(err)
	s.ErrorAs(err, &megaraid.ErrNoCacheOptionsToUpdate)
}
