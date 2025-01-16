package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const mdadmBinaryPath = "" // FIXME Figure out where the binary lives

type MDADM struct {
	cliPath string
}

var (
	_                CommandRunner = &MDADM{}
	MDADMExecCommand               = exec.Command
)

func NewMDADM() *MDADM {
	return &MDADM{
		cliPath: mdadmBinaryPath,
	}
}

func (m *MDADM) Run(args []string) ([]byte, error) {
	cmd := MDADMExecCommand(m.cliPath, args...) //nolint:gosec // FIXME Temporary nolint

	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run mdadm command")
	}

	return output, nil
}
