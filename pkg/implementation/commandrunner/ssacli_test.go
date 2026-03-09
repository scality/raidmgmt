package commandrunner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
