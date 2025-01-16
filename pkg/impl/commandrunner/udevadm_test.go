package commandrunner_test

import (
	"commandrunner"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockUDevADMRun(t *testing.T) {
	// Replace the real exec.Command with the mock
	originalCommand := commandrunner.UDevADMExecCommand
	defer func() { commandrunner.UDevADMExecCommand = originalCommand }()

	commandrunner.UDevADMExecCommand = mockedExecCommand

	runner := commandrunner.NewUDevADM()

	// Run the function
	output, err := runner.Run([]string{"mocked udevadm command"})
	fmt.Println("output: ", string(output))

	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, []byte("PASS\n"), output)
}
