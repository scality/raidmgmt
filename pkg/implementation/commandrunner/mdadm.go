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

func NewMDADM(path *string) *MDADM {
	cliPath := MDADMBinaryPath
	if path != nil && *path != "" {
		cliPath = *path
	}

	return &MDADM{
		cliPath: cliPath,
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
