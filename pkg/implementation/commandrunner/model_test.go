package commandrunner_test

import (
	"fmt"
	"os"
	"os/exec"
)

func mockedExecCommand(command string, args ...string) *exec.Cmd {
	fmt.Println("Mocked command:", command, args)
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}

	return cmd
}
