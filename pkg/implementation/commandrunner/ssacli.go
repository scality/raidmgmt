package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const SSACLIPath = "/opt/hp/hp_ssacli/bin/hp_ssacli"

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
		cliPath: SSACLIPath,
	}
}

func (s *SSACLI) Run(args []string) ([]byte, error) {
	cmd := SSACLIExecCommand(s.cliPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run ssacli command: %s", err)
	}

	return output, nil
}
