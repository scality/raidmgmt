package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const SmartCTLBinaryPath = "/usr/sbin/smartctl"

type SmartCTL struct {
	cliPath string
}

var (
	_ CommandRunner = &SmartCTL{}

	//nolint:gochecknoglobals // Needed for mocking in tests
	SmartCTLExecCommand = exec.Command
)

func NewSmartCTL(path *string) *SmartCTL {
	cliPath := SmartCTLBinaryPath
	if path != nil && *path != "" {
		cliPath = *path
	}

	return &SmartCTL{
		cliPath: cliPath,
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
