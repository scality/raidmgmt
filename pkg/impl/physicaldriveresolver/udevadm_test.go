package physicaldriveresolver_test

import (
	"physicaldriveresolver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type (
	MockCommandRunner struct {
		mock.Mock
	}
)

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)
	return arguments.Get(0).([]byte), arguments.Error(1)
}

func (m *MockCommandRunner) RunWithCombinedOutput(args []string) ([]byte, error) {
	arguments := m.Called(args)
	return arguments.Get(0).([]byte), arguments.Error(1)
}

func TestUDevADMResolvePhysicalDriveDeviceNameFromID(t *testing.T) {
	// // Create a mock object
	mockRunner := &MockCommandRunner{}

	deviceName := "nvme1n1"

	// Set up expected behavior of the mock
	mockRunner.On("Run", []string{"info", "--query=name", "--name=/dev/disk/by-id/nvme-nvme.1d0f-766f6c3062343563353662356465616535663665-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001"}).Return([]byte(deviceName), nil)

	// Use the mock object in your test
	udevadm := &physicaldriveresolver.UDevADM{CommandRunner: mockRunner}
	outputDeviceName, err := udevadm.ResolvePhysicalDriveDeviceNameFromID("nvme-nvme.1d0f-766f6c3062343563353662356465616535663665-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001")

	assert.Equal(t, deviceName, outputDeviceName)
	assert.Nil(t, err)
}
