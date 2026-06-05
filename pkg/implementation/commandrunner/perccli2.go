package commandrunner

import (
	"os/exec"

	"github.com/pkg/errors"
)

// PercCLI2Path is the default path to the perccli2 binary.
const PercCLI2Path = "/opt/MegaRAID/perccli2/perccli2"

type PercCLI2 struct {
	cliPath string
}

var (
	_ CommandRunner = &PercCLI2{}

	//nolint:gochecknoglobals // Needed for mocking in tests
	PercCLI2ExecCommand = exec.Command
)

func NewPercCLI2(path *string) *PercCLI2 {
	cliPath := PercCLI2Path
	if path != nil && *path != "" {
		cliPath = *path
	}

	return &PercCLI2{
		cliPath: cliPath,
	}
}

// Run appends the JSON output flag and returns the command's standard output.
// perccli2 emits the same JSON envelope as storcli2; stdout is captured on its
// own (not combined with stderr) so the payload parses cleanly.
func (p *PercCLI2) Run(args []string) ([]byte, error) {
	argsJSON := make([]string, 0, len(args)+1)
	argsJSON = append(argsJSON, args...)
	argsJSON = append(argsJSON, jsonOutputFlag)

	cmd := PercCLI2ExecCommand(p.cliPath, argsJSON...)

	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run perccli2 command")
	}

	return output, nil
}
