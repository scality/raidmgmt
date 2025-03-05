package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const LSBLKBinaryPath = "/usr/bin/lsblk"

type LSBLK struct {
	cliPath string
}

var (
	_ CommandRunner = &LSBLK{}
	//nolint:gochecknoglobals // Needed for mocking in tests
	LSBLKExecCommand = exec.Command
)

func NewLSBLK() *LSBLK {
	return &LSBLK{
		cliPath: LSBLKBinaryPath,
	}
}

func (l *LSBLK) Run(args []string) ([]byte, error) {
	cmd := LSBLKExecCommand(l.cliPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run lsblk command: %s", string(output))
	}

	return output, nil
}
