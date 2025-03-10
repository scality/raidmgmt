package controllergetter_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/controllergetter"
)

type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)

	return arguments.Get(0).([]byte), arguments.Error(1)
}

var testDataPath = "./"

func mockOutput(filename string) []byte {
	output, err := os.ReadFile(testDataPath + filename + ".txt")
	if err != nil {
		panic(err)
	}

	return output
}

// TestControllers tests the Controllers method.
func TestControllers(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	s := &controllergetter.SSACLI{
		SSACLI: mockRunner,
	}

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
		mockRunner.On("Run", []string{"controller", "all", "show", "detail"}).Return(tt.mocking, nil)

		controllers, err := s.Controllers()
		if tt.expectedError {
			assert.Error(t, err)
			assert.Nil(t, controllers)
		} else {
			assert.NoError(t, err)
			assert.NotEmpty(t, controllers)

			for _, controller := range controllers {
				assert.NotEmpty(t, controller.Name)
				assert.NotEmpty(t, controller.Serial)
			}

			for _, controller := range controllers {
				t.Logf("Controller %d: %+v", controller.ID, controller)
			}
		}
	}
}

func TestController(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	s := &controllergetter.SSACLI{
		SSACLI: mockRunner,
	}

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
		mockRunner.On("Run", []string{"controller", "slot=" + strconv.Itoa(tt.id), "show", "detail"}).Return(tt.mocking, nil)

		metadata := &raidcontroller.Metadata{
			ID: tt.id,
		}

		controller, err := s.Controller(metadata)
		if tt.expectedError {
			assert.Error(t, err)
			assert.Nil(t, controller)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, controller)

			assert.Equal(t, metadata.ID, controller.ID)
			assert.Equal(t, "P816i-a SR Gen10", controller.Name)
			assert.Equal(t, "PWXLA0CRHF10FM", controller.Serial)
		}
	}
}
