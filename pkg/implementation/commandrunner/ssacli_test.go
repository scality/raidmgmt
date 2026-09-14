package commandrunner_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

func TestMockSSACLIRun(t *testing.T) {
	originalCommand := commandrunner.SSACLIExecCommand
	defer func() { commandrunner.SSACLIExecCommand = originalCommand }()

	commandrunner.SSACLIExecCommand = mockedExecCommand

	runner := commandrunner.NewSSACLI(nil)

	output, err := runner.Run([]string{"mocked ssacli command"})
	assert.NoError(t, err)
	assert.Contains(t, string(output), "PASS")
}

// ssacli exits 1 when the controller has no logical drive, printing only that
// message. The runner reports it through ErrNoLogicalDrives so the caller can
// read an empty inventory; any other non-zero exit stays a hard failure.
func TestSSACLIRunNonZeroExit(t *testing.T) {
	tests := []struct {
		name             string
		message          string
		expectedSentinel error
	}{
		{
			name:             "no logical drive",
			message:          "Error: The specified device does not have any logical drives.",
			expectedSentinel: commandrunner.ErrNoLogicalDrives,
		},
		{
			name:             "any other failure",
			message:          "Error: The specified controller does not exist.",
			expectedSentinel: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := commandrunner.SSACLIExecCommand
			defer func() { commandrunner.SSACLIExecCommand = original }()

			commandrunner.SSACLIExecCommand = func(_ string, _ ...string) *exec.Cmd {
				return exec.Command("sh", "-c", "printf '%s\\n' '"+tt.message+"'; exit 1")
			}

			output, err := commandrunner.NewSSACLI(nil).Run(
				[]string{"controller", "slot=0", "logicaldrive", "all", "show", "detail"},
			)
			require.Error(t, err)

			if tt.expectedSentinel == nil {
				require.NotErrorIs(t, err, commandrunner.ErrNoLogicalDrives)
				assert.Nil(t, output)

				return
			}

			require.ErrorIs(t, err, tt.expectedSentinel)
			assert.Contains(t, string(output), tt.message)
		})
	}
}
