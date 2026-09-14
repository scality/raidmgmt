package commandrunner

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

const SSACLIPath = "ssacli"

// ssacliNoLogicalDrivesMessage is what ssacli writes, with exit code 1, when the
// controller has no logical drive configured: "Error: The specified device does
// not have any logical drives.". Only the logical drive queries report it, but
// the runner is shared, so every invocation is checked against it.
const ssacliNoLogicalDrivesMessage = "does not have any logical drives"

type SSACLI struct {
	cliPath string
}

var (
	_ CommandRunner = &SSACLI{}

	// ErrNoLogicalDrives reports a controller without any logical drive. ssacli
	// exits non-zero in that case, which is not a failure: a caller listing
	// logical drives reads it as an empty inventory.
	ErrNoLogicalDrives = errors.New("controller has no logical drive")

	//nolint:gochecknoglobals // Needed for mocking in tests
	SSACLIExecCommand = exec.Command
)

func NewSSACLI(path *string) *SSACLI {
	cliPath := SSACLIPath
	if path != nil && *path != "" {
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
		// The exit code alone cannot tell an empty inventory from a real
		// failure, so the message decides.
		if strings.Contains(string(output), ssacliNoLogicalDrivesMessage) {
			return output, ErrNoLogicalDrives
		}

		return nil, errors.Wrapf(err, "failed to run ssacli command: %s", err)
	}

	return output, nil
}
