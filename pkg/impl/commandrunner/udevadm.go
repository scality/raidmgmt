package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

type UDevADM struct {
	cliPath string
}

const (
	uDevADMBinaryPath = "/usr/bin/udevadm"
)

var (
	_                  CommandRunner = &UDevADM{}
	UDevADMExecCommand               = exec.Command
)

func NewUDevADM() *UDevADM {
	return &UDevADM{
		cliPath: uDevADMBinaryPath,
	}
}

func (u *UDevADM) Run(args []string) ([]byte, error) {
	cmd := UDevADMExecCommand(u.cliPath, args...) //nolint:gosec // FIXME Temporary nolint

	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run udevadm command")
	}

	return output, nil
}
