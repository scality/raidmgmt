package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const smartctlBinaryPath = "/usr/sbin/smartctl"

type SmartCTL struct {
	cliPath string
}

var (
	_ CommandRunner = &SmartCTL{}

	//nolint:gochecknoglobals // Needed for mocking in tests
	SmartCTLExecCommand = exec.Command
)

func NewSmartCTL() *SmartCTL {
	return &SmartCTL{
		cliPath: smartctlBinaryPath,
	}
}

func (s *SmartCTL) Run(args []string) ([]byte, error) {
	cmd := SmartCTLExecCommand(s.cliPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run smartctl command: %s", err)
	}

	return output, nil
}
