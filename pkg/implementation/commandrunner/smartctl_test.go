package commandrunner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

func TestMockSmartCTLRun(t *testing.T) {
	// Replace the real exec.Command with the mock
	originalCommand := commandrunner.SmartCTLExecCommand
	defer func() { commandrunner.SmartCTLExecCommand = originalCommand }()

	commandrunner.SmartCTLExecCommand = mockedExecCommand

	runner := commandrunner.NewSmartCTL()

	// Run the function
	output, err := runner.Run([]string{"mocked smartctl command"})

	// Assert results
	assert.NoError(t, err)
	assert.Contains(t, string(output), "PASS")
}
