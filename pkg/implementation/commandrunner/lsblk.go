package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

const LSBLKBinaryPath = "/usr/bin/lsblk"

type LSBLK struct {
	cliPath string
}

var (
	_ CommandRunner = &LSBLK{}
	//nolint:gochecknoglobals // Needed for mocking in tests
	LSBLKExecCommand = exec.Command
)

func NewLSBLK(path *string) *LSBLK {
	cliPath := LSBLKBinaryPath
	if path != nil && *path != "" {
		cliPath = *path
	}

	return &LSBLK{
		cliPath: cliPath,
	}
}

// CommandPath returns the path of the lsblk binary this runner invokes.
func (l *LSBLK) CommandPath() string {
	return l.cliPath
}

func (l *LSBLK) Run(args []string) ([]byte, error) {
	cmd := LSBLKExecCommand(l.cliPath, args...)

	// Only stdout is returned: lsblk can print warnings on stderr and still exit
	// 0, which would corrupt its JSON output.
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, errors.Wrapf(err, "failed to run lsblk command: %s", string(exitErr.Stderr))
		}

		return nil, errors.Wrap(err, "failed to run lsblk command")
	}

	return output, nil
}
