package commandrunner_test

import (
	"commandrunner"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockMDADMRun(t *testing.T) {
	// Replace the real exec.Command with the mock
	originalCommand := commandrunner.MDADMExecCommand
	defer func() { commandrunner.MDADMExecCommand = originalCommand }()

	commandrunner.MDADMExecCommand = mockedExecCommand

	runner := commandrunner.NewMDADM()

	// Run the function
	output, err := runner.Run([]string{"mocked mdadm command"})
	fmt.Println("output: ", string(output))

	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, []byte("PASS\n"), output)
}
