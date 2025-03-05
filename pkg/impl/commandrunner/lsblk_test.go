package commandrunner_test

import (
	"testing"

	"github.com/scality/raidmgmt/commandrunner"
	"github.com/stretchr/testify/assert"
)

func TestMockLSBLKRun(t *testing.T) {
	// Replace the real exec.Command with the mock
	originalCommand := commandrunner.LSBLKExecCommand
	defer func() { commandrunner.LSBLKExecCommand = originalCommand }()

	commandrunner.LSBLKExecCommand = mockedExecCommand

	runner := commandrunner.NewLSBLK()

	// Run the function
	output, err := runner.Run([]string{"mocked lsblk command"})

	// Assert results
	assert.NoError(t, err)
	assert.Contains(t, string(output), "PASS")
}
