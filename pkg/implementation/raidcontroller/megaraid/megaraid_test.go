package megaraid_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	megaraid2 "github.com/scality/raidmgmt/pkg/implementation/raidcontroller/megaraid"
	mocks2 "github.com/scality/raidmgmt/pkg/implementation/raidcontroller/megaraid/mocks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var pathTestData = "./testdata/"

type UnitTestSuite struct {
	suite.Suite

	a                *megaraid2.Adapter
	mockRunner       *mocks2.Runner
	mockPathResolver *mocks2.PathResolver

	wasCreateLVCalledOnce bool

	// fileExistsFunc is the original FileExists function
	// It's used to restore the original function after mocking it
	fileExistsFunc func(string) bool
	// evalSymlinksFunc is the original EvalSymlinks function
	// It's used to restore the original function after mocking it
	evalSymlinksFunc func(string) (string, error)
}

// TestRunSuite runs the test suite. It's called by go test.
func TestRunSuite(t *testing.T) {
	suite.Run(t, &UnitTestSuite{})
}

// SetupTest sets up the test suite. It's called before each test.
func (s *UnitTestSuite) SetupTest() {
	s.mockRunner = mocks2.NewRunner(s.T())
	s.a = megaraid2.New(s.mockRunner)

	s.mockPathResolver = mocks2.NewPathResolver(s.T())
}

// mockOutput reads the output from a file and returns it.
func mockOutput(filename string) *megaraid2.CmdOutput {
	file, err := os.Open(pathTestData + filename + ".json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var data megaraid2.CmdOutput

	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		panic(err)
	}

	return &data
}

// parseError parses the error from the command status.
// It's copied from the megaraid package.
func parseError(commandStatus megaraid2.CommandStatus) error {
	if len(commandStatus.DetailedStatus) > 0 {
		for _, ds := range commandStatus.DetailedStatus {
			if ds.Status != "Success" {
				if ds.Description != nil {
					return errors.Errorf("%s: %s", ds.ErrMsg, *ds.Description)
				}

				return errors.New(ds.ErrMsg)
			}
		}
	}

	return errors.New(commandStatus.Description)
}

// mockError checks if the command was successful.
// It's copied from the megaraid package.
func mockError(out *megaraid2.CmdOutput) error {
	// Check if there are any controllers
	if len(out.Controllers) == 0 {
		return errors.New("no controllers found")
	}

	// Check if the command was successful
	for _, controller := range out.Controllers {
		commandStatus := controller.CommandStatus

		if commandStatus.Status != "Success" {
			return parseError(commandStatus)
		}
	}

	return nil
}

// mockReturn returns the output and the error.
func mockReturn(filename string) (*megaraid2.CmdOutput, error) {
	out := mockOutput(filename)

	return out, mockError(out)
}

// createLVMockCalls mocks the calls for the createLV function.
// It's used to test the createLV function.
func (s *UnitTestSuite) createLVMockCalls(args []string) (*megaraid2.CmdOutput, error) {
	// controllers calls
	if args[0] == "show" {
		return mockReturn("controllers/all")
	}

	if args[0] == "/c0" {
		if args[1] == "show" {
			if !s.wasCreateLVCalledOnce {
				return mockReturn("controllers/c0_s12_UGood")
			}

			return mockReturn("controllers/c0")
		}

		if args[1] == "add" {
			s.wasCreateLVCalledOnce = true

			return mockReturn("logicalvolumes/create/success")
		}
	}

	args0Split := strings.Split(args[0], "/")

	// physical drives calls
	if args0Split[2] == "e251" {
		filename := fmt.Sprintf("physicaldrives/show/e251%s", args0Split[3])

		if !s.wasCreateLVCalledOnce {
			// Slot 12 is unconfigured and good
			if args0Split[3] == "s12" {
				filename = fmt.Sprintf("physicaldrives/show/e251%s_UGood", args0Split[3])
			}
		}

		return mockReturn(filename)
	}

	// logical volumes calls
	filename := fmt.Sprintf("logicalvolumes/show/%s", args0Split[2])

	return mockReturn(filename)
}

// generalMockCalls mocks the calls for the all the other functions.
func (s *UnitTestSuite) generalMockCalls(args []string) (*megaraid2.CmdOutput, error) {
	switch args[0] {
	case "show":
		return mockReturn("controllers/all")
	case "/c0":
		return mockReturn("controllers/c0")
	case "/c5":
		return mockReturn("controllers/c5_invalid")
	}

	args0Split := strings.Split(args[0], "/")

	// physical drives calls
	if args0Split[2] == "e251" {
		filename := fmt.Sprintf("physicaldrives/show/e251%s", args0Split[3])

		if args0Split[3] == "s99" {
			return mockReturn("physicaldrives/show/e251s99_invalid")
		}

		return mockReturn(filename)
	}

	// logical volumes calls
	if args0Split[2] == "v999" {
		return mockReturn("logicalvolumes/show/v999_invalid")
	}

	filename := fmt.Sprintf("logicalvolumes/show/%s", args0Split[2])

	return mockReturn(filename)
}

// restoreCustomFileExists restores the custom FileExists function
// to the original one.
// It's used to mock the FileExists function.
func (s *UnitTestSuite) restoreCustomFileExists() {
	megaraid2.CustomFileExists = s.fileExistsFunc
}

// setupCustomFileExists sets up the custom FileExists function
// to be used in the tests.
// It's used to mock the FileExists function.
func (s *UnitTestSuite) setupCustomFileExists() {
	s.fileExistsFunc = megaraid2.CustomFileExists
	megaraid2.CustomFileExists = s.mockPathResolver.FileExists
}

// restoreCustomEvalSymlinks restores the custom EvalSymlinks function
// to the original one.
// It's used to mock the EvalSymlinks function.
func (s *UnitTestSuite) restoreCustomEvalSymlinks() {
	megaraid2.CustomEvalSymlinks = s.evalSymlinksFunc
}

// setupCustomEvalSymlinks sets up the custom EvalSymlinks function
// to be used in the tests.
// It's used to mock the EvalSymlinks function.
func (s *UnitTestSuite) setupCustomEvalSymlinks() {
	s.evalSymlinksFunc = megaraid2.CustomEvalSymlinks
	megaraid2.CustomEvalSymlinks = s.mockPathResolver.EvalSymlinks
}

// setupMockCallsCreateLV sets up the mock calls for the createLV function.
func (s *UnitTestSuite) setupMockCallsCreateLV() {
	s.mockRunner.On("Run", mock.AnythingOfType("[]string")).Return(s.createLVMockCalls)
}

// setupMockCalls sets up the mock calls for the general functions.
func (s *UnitTestSuite) setupMockCalls() {
	s.mockRunner.On("Run", mock.AnythingOfType("[]string")).Return(s.generalMockCalls)
}

func (s *UnitTestSuite) TestControllers() {
	s.setupMockCalls()

	tests := []struct {
		controllersExpected int
		errExpected         bool
		err                 string
	}{
		{
			controllersExpected: 1,
			errExpected:         false,
			err:                 "",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		controllers, err := s.a.Controllers()

		if tt.errExpected {
			s.Nil(controllers)
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
			s.Len(controllers, tt.controllersExpected)
		}
	}
}

func (s *UnitTestSuite) TestController() {
	s.setupMockCalls()

	tests := []struct {
		metadata    *raidcontroller.Metadata
		errExpected bool
		err         string
	}{
		{
			metadata: &raidcontroller.Metadata{
				ID: 0,
			},
			errExpected: false,
			err:         "",
		},
		{
			metadata: &raidcontroller.Metadata{
				ID: 5,
			},
			errExpected: true,
			err:         "Controller 5 not found",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		controller, err := s.a.Controller(tt.metadata)

		if tt.errExpected {
			s.Nil(controller)
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
			s.Equal("MegaRAID 9560-8i 4GB", controller.Name)
			s.Equal("SKC5120859", controller.Serial)
		}
	}
}

func (s *UnitTestSuite) TestPhysicalDrives() {
	s.setupMockCalls()

	tests := []struct {
		metadata    *raidcontroller.Metadata
		drivesCount int
		errExpected bool
		err         string
	}{
		{
			metadata: &raidcontroller.Metadata{
				ID: 0,
			},
			drivesCount: 12,
			errExpected: false,
			err:         "",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		drives, err := s.a.PhysicalDrives(tt.metadata)

		if tt.errExpected {
			s.Nil(drives)
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
			s.Len(drives, tt.drivesCount)
		}
	}
}

func (s *UnitTestSuite) TestPhysicalDrive() {
	s.setupMockCalls()

	tests := []struct {
		metadata    *physicaldrive.Metadata
		errExpected bool
		err         string
	}{
		{
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "251:6",
			},
			errExpected: false,
			err:         "",
		},
		{
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "251:99",
			},
			errExpected: true,
			err:         "Drive not found",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		pDrive, err := s.a.PhysicalDrive(tt.metadata)

		if tt.errExpected {
			s.Nil(pDrive)
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
			s.Equal("251:6", pDrive.ID)
			s.Equal("ZVT2DBEW0000C24112G0", pDrive.Serial)
			s.Equal(uint64(17999005346693), pDrive.Size)
			s.Equal(physicaldrive.PDStatusUsed, pDrive.Status)
			s.Equal(physicaldrive.DiskTypeHDD, pDrive.Type)
		}
	}
}

func (s *UnitTestSuite) TestLogicalVolumes() {
	s.setupMockCalls()
	s.mockPathResolver.On("FileExists", mock.Anything).Return(true)
	s.mockPathResolver.On("EvalSymlinks", "/dev/disk/by-id/wwn-0x600062b212da5d402bd3b493e1699377").Return("/dev/sda", nil)

	s.setupCustomFileExists()
	defer s.restoreCustomFileExists()

	s.setupCustomEvalSymlinks()
	defer s.restoreCustomEvalSymlinks()

	tests := []struct {
		metadata    *raidcontroller.Metadata
		lvCount     int
		errExpected bool
		err         string
	}{
		{
			metadata: &raidcontroller.Metadata{
				ID: 0,
			},
			lvCount:     12,
			errExpected: false,
			err:         "",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		lVolumes, err := s.a.LogicalVolumes(tt.metadata)

		if tt.errExpected {
			s.Nil(lVolumes)
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
			s.Len(lVolumes, tt.lvCount)
			s.Equal("228", lVolumes[0].ID)
			s.Equal(logicalvolume.RAIDLevel0, lVolumes[0].RAIDLevel)
			s.Len(lVolumes[0].PDrivesMetadata, 1)
			s.Equal(logicalvolume.LVStatusOptimal, lVolumes[0].Status)
			s.Equal(&logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteThrough,
				IOPolicy:    logicalvolume.IOPolicyDirect,
			}, lVolumes[0].CacheOptions)

			s.Equal("233", lVolumes[5].ID)
			s.Equal(logicalvolume.RAIDLevel0, lVolumes[5].RAIDLevel)
			s.Len(lVolumes[5].PDrivesMetadata, 1)
			s.Equal(logicalvolume.LVStatusOptimal, lVolumes[5].Status)
			s.Equal(&logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteThrough,
				IOPolicy:    logicalvolume.IOPolicyDirect,
			}, lVolumes[5].CacheOptions)
		}
	}
}

func (s *UnitTestSuite) TestLogicalVolume() {
	s.setupMockCalls()
	s.mockPathResolver.On("FileExists", "/dev/disk/by-id/wwn-0x600062b212da5d402bd3b493e1699377").Return(true)
	s.mockPathResolver.On("EvalSymlinks", "/dev/disk/by-id/wwn-0x600062b212da5d402bd3b493e1699377").Return("/dev/sda", nil)

	s.setupCustomFileExists()
	defer s.restoreCustomFileExists()

	s.setupCustomEvalSymlinks()
	defer s.restoreCustomEvalSymlinks()

	tests := []struct {
		metadata    *logicalvolume.Metadata
		errExpected bool
		err         string
	}{
		{
			metadata: &logicalvolume.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "228",
			},
			errExpected: false,
			err:         "",
		},
		{
			metadata: &logicalvolume.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "999",
			},
			errExpected: true,
			err:         "Invalid VD number",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		lv, err := s.a.LogicalVolume(tt.metadata)

		if tt.errExpected {
			s.Nil(lv)
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
			s.Equal("228", lv.ID)
			s.Equal(logicalvolume.RAIDLevel0, lv.RAIDLevel)
			s.Len(lv.PDrivesMetadata, 1)
			s.Equal(logicalvolume.LVStatusOptimal, lv.Status)
			s.Equal(&logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteThrough,
				IOPolicy:    logicalvolume.IOPolicyDirect,
			}, lv.CacheOptions)
		}
	}
}

func (s *UnitTestSuite) TestEnableJBOD() {
	s.mockRunner.On("Run", []string{"/c0/e251/s6", "set", "jbod"}).
		Return(mockReturn("physicaldrives/jbod/enable/fail"))

	tests := []struct {
		metadata    *physicaldrive.Metadata
		errExpected bool
		err         string
	}{
		{
			metadata: &physicaldrive.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "251:6",
			},
			errExpected: true,
			err:         "device state doesn't support requested command",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		err := s.a.EnableJBOD(tt.metadata)

		if tt.errExpected {
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
		}
	}
}

func (s *UnitTestSuite) TestDisableJBOD() {
	s.mockRunner.On("Run", []string{"/c0/e251/s6", "delete", "jbod"}).
		Return(mockReturn("physicaldrives/jbod/disable/fail"))

	metadata := &physicaldrive.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{
			ID: 0,
		},
		ID: "251:6",
	}

	err := s.a.DisableJBOD(metadata)

	s.Error(err)
	s.ErrorContains(err, "Operation not allowed")
}

func (s *UnitTestSuite) TestSetLVCacheOptions() {
	s.setupMockCalls()

	s.mockPathResolver.On("FileExists", "/dev/disk/by-id/wwn-0x600062b212da5d402bd3b493e1699377").Return(true)
	s.mockPathResolver.On("EvalSymlinks", "/dev/disk/by-id/wwn-0x600062b212da5d402bd3b493e1699377").Return("/dev/sda", nil)

	s.setupCustomFileExists()
	defer s.restoreCustomFileExists()

	s.setupCustomEvalSymlinks()
	defer s.restoreCustomEvalSymlinks()

	tests := []struct {
		cacheOptions *logicalvolume.CacheOptions
		errExpected  bool
		err          string
	}{
		{
			cacheOptions: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyNoReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicyDirect,
			},
			errExpected: false,
			err:         "",
		},
		// Same options
		{
			cacheOptions: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteThrough,
				IOPolicy:    logicalvolume.IOPolicyDirect,
			},
			errExpected: false,
			err:         "",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		ctrlMetadata := &raidcontroller.Metadata{
			ID: 0,
		}

		lvMetadata := &logicalvolume.Metadata{
			CtrlMetadata: ctrlMetadata,
			ID:           "228",
		}

		err := s.a.SetLVCacheOptions(lvMetadata, tt.cacheOptions)

		if tt.errExpected {
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
		}
	}
}

func (s *UnitTestSuite) TestCreateLV() {
	s.setupMockCallsCreateLV()
	s.mockPathResolver.On("FileExists", mock.Anything).Return(true)
	s.mockPathResolver.On("EvalSymlinks", "/dev/disk/by-id/wwn-0x600062b212da5d402bd3b493e1699377").Return("/dev/sda", nil)

	s.setupCustomFileExists()
	defer s.restoreCustomFileExists()

	s.setupCustomEvalSymlinks()
	defer s.restoreCustomEvalSymlinks()

	tests := []struct {
		request     *logicalvolume.Request
		errExpected bool
		err         string
	}{
		{
			request: &logicalvolume.Request{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				RAIDLevel: logicalvolume.RAIDLevel0,
				PDrivesMetadata: []*physicaldrive.Metadata{
					{
						CtrlMetadata: &raidcontroller.Metadata{
							ID: 0,
						},
						ID: "251:12",
					},
				},
				CacheOptions: &logicalvolume.CacheOptions{
					ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
					WritePolicy: logicalvolume.WritePolicyWriteThrough,
					IOPolicy:    logicalvolume.IOPolicyDirect,
				},
			},
			errExpected: false,
			err:         "",
		},
		{
			request: &logicalvolume.Request{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				RAIDLevel: logicalvolume.RAIDLevel0,
				PDrivesMetadata: []*physicaldrive.Metadata{
					{
						CtrlMetadata: &raidcontroller.Metadata{
							ID: 0,
						},
						ID: "251:12",
					},
				},
				CacheOptions: &logicalvolume.CacheOptions{
					ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
					WritePolicy: logicalvolume.WritePolicyWriteThrough,
					IOPolicy:    logicalvolume.IOPolicyDirect,
				},
			},
			errExpected: true,
			err:         "unavailable drives: 251:12",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		newLv, err := s.a.CreateLV(tt.request)

		if tt.errExpected {
			s.Nil(newLv)
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
			s.Equal("228", newLv.ID)
			s.Equal("/dev/sda", newLv.DevicePath)
			s.Equal(logicalvolume.RAIDLevel0, newLv.RAIDLevel)
			s.Equal(logicalvolume.LVStatusOptimal, newLv.Status)
			s.Len(newLv.PDrivesMetadata, 1)
			s.Equal("251:12", newLv.PDrivesMetadata[0].ID)
			s.Equal(logicalvolume.ReadPolicyReadAhead, newLv.CacheOptions.ReadPolicy)
			s.Equal(logicalvolume.WritePolicyWriteThrough, newLv.CacheOptions.WritePolicy)
			s.Equal(logicalvolume.IOPolicyDirect, newLv.CacheOptions.IOPolicy)
			s.Equal("/dev/disk/by-id/wwn-0x600062b212da5d402bd3b493e1699377", newLv.PermanentPath)
		}
	}
}

func (s *UnitTestSuite) TestDeleteLV() {
	s.mockRunner.On("Run", []string{"/c0/v228", "delete"}).
		Return(mockReturn("logicalvolumes/delete/success"))

	s.mockRunner.On("Run", []string{"/c0/v299", "delete"}).
		Return(mockReturn("logicalvolumes/delete/fail_invalid"))

	tests := []struct {
		metadata    *logicalvolume.Metadata
		errExpected bool
		err         string
	}{
		{
			metadata: &logicalvolume.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "228",
			},
			errExpected: false,
			err:         "",
		},
		{
			metadata: &logicalvolume.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "299",
			},
			errExpected: true,
			err:         "Invalid VD number",
		},
		// TODO : complete the test cases
	}

	for _, tt := range tests {
		err := s.a.DeleteLV(tt.metadata)

		if tt.errExpected {
			s.Error(err)
			s.ErrorContains(err, tt.err)
		} else {
			s.NoError(err)
		}
	}
}

func (s *UnitTestSuite) TestStartBlink() {
	s.mockRunner.On("Run", []string{"/c0/e251/s9", "start", "locate"}).
		Return(mockReturn("physicaldrives/blink/start"))

	metadata := &physicaldrive.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{
			ID: 0,
		},
		ID: "251:9",
	}

	err := s.a.StartBlink(metadata)

	s.NoError(err)
}

func (s *UnitTestSuite) TestStopBlink() {
	s.mockRunner.On("Run", []string{"/c0/e251/s9", "stop", "locate"}).
		Return(mockReturn("physicaldrives/blink/stop"))

	metadata := &physicaldrive.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{
			ID: 0,
		},
		ID: "251:9",
	}

	err := s.a.StopBlink(metadata)

	s.NoError(err)
}

// TODO get test data.
func (s *UnitTestSuite) TestAddPDToLV() {
	s.T().Skip("not implemented")
}

// TODO get test data.
func (s *UnitTestSuite) TestDeletePDFromLV() {
	s.T().Skip("not implemented")
}
