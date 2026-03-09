package commandrunner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

func TestMockMDADMRun(t *testing.T) {
	// Replace the real exec.Command with the mock
	originalCommand := commandrunner.MDADMExecCommand
	defer func() { commandrunner.MDADMExecCommand = originalCommand }()

	commandrunner.MDADMExecCommand = mockedExecCommand

	runner := commandrunner.NewMDADM(nil)

	// Run the function & assert the results
	output, err := runner.Run([]string{"mocked mdadm command"})
	assert.NoError(t, err)
	assert.Contains(t, string(output), "PASS")
}
