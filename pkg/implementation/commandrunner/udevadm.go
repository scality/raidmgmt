package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

type UDevADM struct {
	cliPath string
}

const (
	UDevADMBinaryPath = "/usr/bin/udevadm"
)

var (
	_ CommandRunner = &UDevADM{}
	//nolint:gochecknoglobals // Needed for mocking in tests
	UDevADMExecCommand = exec.Command
)

func NewUDevADM(path *string) *UDevADM {
	cliPath := UDevADMBinaryPath
	if path != nil {
		cliPath = *path
	}

	return &UDevADM{
		cliPath: cliPath,
	}
}

func (u *UDevADM) Run(args []string) ([]byte, error) {
	cmd := UDevADMExecCommand(u.cliPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run udevadm command: %s", string(output))
	}

	return output, nil
}
