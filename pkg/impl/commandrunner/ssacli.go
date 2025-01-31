package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const SSACLIBinaryPath = "/opt/smartstorageadmin/ssacli/bin/ssacli"

type SSACLI struct {
	cliPath string
}

var (
	_ CommandRunner = &SSACLI{}
	//nolint:gochecknoglobals // Needed for mocking in tests
	SSACLIExecCommand = exec.Command
)

func NewSSACLI() *SSACLI {
	return &SSACLI{
		cliPath: SSACLIBinaryPath,
	}
}

func (m *SSACLI) Run(args []string) ([]byte, error) {
	cmd := SSACLIExecCommand(m.cliPath, args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run mdadm command")
	}

	return output, nil
}
