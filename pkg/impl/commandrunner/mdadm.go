package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const mdadmBinaryPath = "/usr/sbin/mdadm"

type MDADM struct {
	cliPath string
}

var (
	_ CommandRunner = &MDADM{}
	//nolint:gochecknoglobals // Needed for mocking in tests
	MDADMExecCommand = exec.Command
)

func NewMDADM() *MDADM {
	return &MDADM{
		cliPath: mdadmBinaryPath,
	}
}

func (m *MDADM) Run(args []string) ([]byte, error) {
	cmd := MDADMExecCommand(m.cliPath, args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run mdadm command")
	}

	return output, nil
}
