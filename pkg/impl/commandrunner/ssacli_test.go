package commandrunner_test

import (
	"testing"

	"github.com/scality/raidmgmt/commandrunner"
	"github.com/stretchr/testify/assert"
)

func TestMockSSACLIRun(t *testing.T) {
	// Replace the real exec.Command with the mock
	originalCommand := commandrunner.SSACLIExecCommand
	defer func() { commandrunner.SSACLIExecCommand = originalCommand }()

	commandrunner.SSACLIExecCommand = mockedExecCommand

	runner := commandrunner.NewSSACLI()

	// Run the function & assert the results
	output, err := runner.Run([]string{"mocked ssacli command"})
	assert.NoError(t, err)
	assert.Equal(t, []byte("PASS\n"), output)
}
