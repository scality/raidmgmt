package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const MDADMBinaryPath = "/usr/sbin/mdadm"

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
		cliPath: MDADMBinaryPath,
	}
}

func (m *MDADM) Run(args []string) ([]byte, error) {
	cmd := MDADMExecCommand(m.cliPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run mdadm command: %s", string(output))
	}

	return output, nil
}
