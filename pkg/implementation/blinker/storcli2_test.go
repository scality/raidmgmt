package blinker_test

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/blinker"
)

type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)

	return arguments.Get(0).([]byte), arguments.Error(1)
}

// storcli2Fixture reads a storcli2 JSON fixture from the package testdata.
func storcli2Fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile("testdata/storcli2/" + name)
	require.NoError(t, err)

	return data
}

// storcli2PDMetadata builds physical-drive metadata with the given EID:Slt id.
func storcli2PDMetadata(id string) *physicaldrive.Metadata {
	return &physicaldrive.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{ID: 0},
		ID:           id,
	}
}

// TestStorCLI2Blink covers the start/stop locate happy paths, including the
// enclosure and no-enclosure selector forms.
func TestStorCLI2Blink(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       string
		selector string
		action   string
		fixture  string
		start    bool
	}{
		{name: "start with enclosure", id: "252:0", selector: "/c0/e252/s0", action: "start", fixture: "start.json", start: true},
		{name: "stop with enclosure", id: "252:0", selector: "/c0/e252/s0", action: "stop", fixture: "stop.json", start: false},
		{name: "start without enclosure", id: "5", selector: "/c0/s5", action: "start", fixture: "start.json", start: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockRunner.On("Run", []string{tt.selector, tt.action, "locate"}).
				Return(storcli2Fixture(t, tt.fixture), nil)

			b := blinker.NewStorCLI2(mockRunner)

			var err error
			if tt.start {
				err = b.StartBlink(storcli2PDMetadata(tt.id))
			} else {
				err = b.StopBlink(storcli2PDMetadata(tt.id))
			}

			require.NoError(t, err)
			mockRunner.AssertExpectations(t)
		})
	}
}

// TestStorCLI2BlinkCommandError pins that a runner failure is surfaced.
func TestStorCLI2BlinkCommandError(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/e252/s0", "start", "locate"}).
		Return([]byte(nil), errors.New("boom"))

	b := blinker.NewStorCLI2(mockRunner)

	err := b.StartBlink(storcli2PDMetadata("252:0"))
	require.Error(t, err)
}

// TestStorCLI2BlinkInvalidSlot pins that an unparseable drive id is rejected
// before any command is run.
func TestStorCLI2BlinkInvalidSlot(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)

	b := blinker.NewStorCLI2(mockRunner)

	err := b.StopBlink(storcli2PDMetadata(""))
	require.Error(t, err)
	mockRunner.AssertNotCalled(t, "Run")
}
