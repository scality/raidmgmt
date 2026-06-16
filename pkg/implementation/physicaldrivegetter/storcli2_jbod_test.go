package physicaldrivegetter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
)

// storcli2Success is a minimal success envelope; the captured JBOD fixtures are
// failures only (a success needs hardware, see ARTESCA-17649).
const storcli2Success = `{"Controllers":[{"Command Status":{"Status":"Success"}}]}`

func storcli2PDMetadata() *physicaldrive.Metadata {
	return &physicaldrive.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{ID: 0},
		ID:           "306:0",
	}
}

func TestStorCLI2EnableJBOD(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/e306/s0", "set", "jbod"}).
		Return([]byte(storcli2Success), nil)

	s := NewStorCLI2(mockRunner)

	require.NoError(t, s.EnableJBOD(storcli2PDMetadata()))
	mockRunner.AssertExpectations(t)
}

func TestStorCLI2DisableJBOD(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/e306/s0", "set", "uconf"}).
		Return([]byte(storcli2Success), nil)

	s := NewStorCLI2(mockRunner)

	require.NoError(t, s.DisableJBOD(storcli2PDMetadata()))
	mockRunner.AssertExpectations(t)
}

// TestStorCLI2EnableJBODFailure pins that the in-JSON failure payload reported
// regardless of exit code is surfaced as an error.
func TestStorCLI2EnableJBODFailure(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/e306/s0", "set", "jbod"}).
		Return(storcli2Fixture(t, "jbod/enable/fail.json"), nil)

	s := NewStorCLI2(mockRunner)

	err := s.EnableJBOD(storcli2PDMetadata())
	require.Error(t, err)
	require.ErrorContains(t, err, "wrong state")
}

// TestStorCLI2JBODSelectorError pins that an unparseable slot aborts before any
// command is run.
func TestStorCLI2JBODSelectorError(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	s := NewStorCLI2(mockRunner)

	err := s.EnableJBOD(&physicaldrive.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{ID: 0},
		ID:           "",
	})
	require.Error(t, err)
	mockRunner.AssertNotCalled(t, "Run")
}
