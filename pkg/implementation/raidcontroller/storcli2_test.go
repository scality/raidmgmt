package raidcontroller_test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	rcentity "github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/raidcontroller"
)

type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)

	return arguments.Get(0).([]byte), arguments.Error(1)
}

const storcli2LocateSuccess = `{
"Controllers":[
{
	"Command Status" : {
		"Status" : "Success",
		"Description" : "Start PD Locate Succeeded."
	}
}
]
}`

// TestStorCLI2Composition is a smoke test: it wires the full adapter on a mocked
// runner, asserts the composition satisfies ports.RAIDController, and drives one
// operation end to end to prove the components are reachable through the runner.
func TestStorCLI2Composition(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/e252/s0", "start", "locate"}).
		Return([]byte(storcli2LocateSuccess), nil)

	var adapter ports.RAIDController = raidcontroller.NewStorCLI2(mockRunner)

	err := adapter.StartBlink(&physicaldrive.Metadata{
		CtrlMetadata: &rcentity.Metadata{ID: 0},
		ID:           "252:0",
	})
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}
