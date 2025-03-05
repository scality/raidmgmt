package commandrunner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

func TestMockUDevADMRun(t *testing.T) {
	// Replace the real exec.Command with the mock
	originalCommand := commandrunner.UDevADMExecCommand
	defer func() { commandrunner.UDevADMExecCommand = originalCommand }()

	commandrunner.UDevADMExecCommand = mockedExecCommand

	runner := commandrunner.NewUDevADM()

	// Run the function
	output, err := runner.Run([]string{"mocked udevadm command"})

	// Assert results
	assert.NoError(t, err)
	assert.Contains(t, string(output), "PASS")
}
