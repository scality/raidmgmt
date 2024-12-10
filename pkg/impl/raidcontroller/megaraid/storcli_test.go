package megaraid_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/scality/raidmgmt/megaraid"
	"github.com/scality/raidmgmt/megaraid/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var path = "./files/"

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

func (s *UnitTestSuite) TestControllers() {
	s.controllersMockCalls()
	controllers, err := s.m.Controllers()

	s.NoError(err)
	s.Len(controllers, 1)
	s.Equal("MegaRAID 9560-8i 4GB", controllers[0].Name)
	s.Equal("SKC5120859", controllers[0].Serial)
}
