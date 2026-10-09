package commandrunner_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

func TestMockLSBLKRun(t *testing.T) {
	// Replace the real exec.Command with the mock
	originalCommand := commandrunner.LSBLKExecCommand
	defer func() { commandrunner.LSBLKExecCommand = originalCommand }()

	commandrunner.LSBLKExecCommand = mockedExecCommand

	runner := commandrunner.NewLSBLK(nil)

	// Run the function
	output, err := runner.Run([]string{"mocked lsblk command"})

	// Assert results
	assert.NoError(t, err)
	assert.Contains(t, string(output), "PASS")
}

func TestLSBLKRun_IgnoresStderrOnSuccess(t *testing.T) {
	originalCommand := commandrunner.LSBLKExecCommand
	defer func() { commandrunner.LSBLKExecCommand = originalCommand }()

	// lsblk can warn on stderr and still exit 0 with a valid JSON document.
	commandrunner.LSBLKExecCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("sh", "-c",
			`echo "lsblk: dm-3: failed to get device path" >&2; echo '{"blockdevices": []}'`)
	}

	output, err := commandrunner.NewLSBLK(nil).Run([]string{"--json"})

	assert.NoError(t, err)
	assert.JSONEq(t, `{"blockdevices": []}`, string(output))
}

func TestLSBLKRun_StderrInError(t *testing.T) {
	originalCommand := commandrunner.LSBLKExecCommand
	defer func() { commandrunner.LSBLKExecCommand = originalCommand }()

	commandrunner.LSBLKExecCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("sh", "-c", `echo "lsblk: /dev/sdz: not a block device" >&2; exit 32`)
	}

	output, err := commandrunner.NewLSBLK(nil).Run([]string{"/dev/sdz"})

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "/dev/sdz: not a block device")
}
