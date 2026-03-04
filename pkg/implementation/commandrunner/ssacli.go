package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const SSACLIPath = "ssacli"

type SSACLI struct {
	cliPath string
}

var (
	_ CommandRunner = &SSACLI{}

	//nolint:gochecknoglobals // Needed for mocking in tests
	SSACLIExecCommand = exec.Command
)

func NewSSACLI(path *string) *SSACLI {
	cliPath := SSACLIPath
	if path != nil {
		cliPath = *path
	}

	return &SSACLI{
		cliPath: cliPath,
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
