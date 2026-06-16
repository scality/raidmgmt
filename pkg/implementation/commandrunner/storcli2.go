package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

// StorCLI2Path is the default path to the storcli2 binary.
const StorCLI2Path = "/opt/MegaRAID/storcli2/storcli2"

// jsonOutputFlag is appended to every storcli2/perccli2 invocation to request
// JSON output.
const jsonOutputFlag = "J"

type StorCLI2 struct {
	cliPath string
}

var (
	_ CommandRunner = &StorCLI2{}

	//nolint:gochecknoglobals // Needed for mocking in tests
	StorCLI2ExecCommand = exec.Command
)

func NewStorCLI2(path *string) *StorCLI2 {
	cliPath := StorCLI2Path
	if path != nil && *path != "" {
		cliPath = *path
	}

	return &StorCLI2{
		cliPath: cliPath,
	}
}

// Run appends the JSON output flag and returns the command's standard output.
// stdout is captured on its own (not combined with stderr) because the payload
// is JSON that must parse cleanly.
//
// storcli2 exits non-zero for some failures (e.g. an invalid drive) while still
// writing its JSON error payload to stdout. The exit code is not a reliable
// success signal — other failures (invalid controller, invalid VD) exit zero —
// so a non-zero exit with a non-empty stdout is returned as-is and left to the
// caller's storcli2.Decode to surface the in-JSON error. Only a non-zero exit
// with no payload (or any other exec failure) is treated as a hard error.
func (s *StorCLI2) Run(args []string) ([]byte, error) {
	argsJSON := make([]string, 0, len(args)+1)
	argsJSON = append(argsJSON, args...)
	argsJSON = append(argsJSON, jsonOutputFlag)

	cmd := StorCLI2ExecCommand(s.cliPath, argsJSON...)

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if len(output) > 0 && errors.As(err, &exitErr) {
			return output, nil
		}

		return nil, errors.Wrap(err, "failed to run storcli2 command")
	}

	return output, nil
}
