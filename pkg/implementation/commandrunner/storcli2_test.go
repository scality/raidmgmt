package commandrunner_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

// megaRAID2RunnerCase describes one of the second-generation MegaRAID command
// runners (storcli2, perccli2). They share behaviour (append the JSON flag,
// return stdout) and differ only by default binary path and exec-command seam.
type megaRAID2RunnerCase struct {
	name        string
	defaultPath string
	newRunner   func(path *string) commandrunner.CommandRunner
	execCommand *func(name string, args ...string) *exec.Cmd
}

func megaRAID2RunnerCases() []megaRAID2RunnerCase {
	return []megaRAID2RunnerCase{
		{
			name:        "storcli2",
			defaultPath: commandrunner.StorCLI2Path,
			newRunner:   func(p *string) commandrunner.CommandRunner { return commandrunner.NewStorCLI2(p) },
			execCommand: &commandrunner.StorCLI2ExecCommand,
		},
		{
			name:        "perccli2",
			defaultPath: commandrunner.PercCLI2Path,
			newRunner:   func(p *string) commandrunner.CommandRunner { return commandrunner.NewPercCLI2(p) },
			execCommand: &commandrunner.PercCLI2ExecCommand,
		},
	}
}

func TestMegaRAID2RunnerAppendsJSONFlag(t *testing.T) {
	for _, tc := range megaRAID2RunnerCases() {
		t.Run(tc.name, func(t *testing.T) {
			original := *tc.execCommand
			defer func() { *tc.execCommand = original }()

			var (
				gotName string
				gotArgs []string
			)

			*tc.execCommand = func(name string, args ...string) *exec.Cmd {
				gotName = name
				gotArgs = args

				return exec.Command("printf", "%s", `{"Controllers":[]}`)
			}

			output, err := tc.newRunner(nil).Run([]string{"show", "all"})
			require.NoError(t, err)

			assert.Equal(t, `{"Controllers":[]}`, string(output))
			assert.Equal(t, tc.defaultPath, gotName)
			assert.Equal(t, []string{"show", "all", "J"}, gotArgs)
		})
	}
}

func TestMegaRAID2RunnerUsesCustomPath(t *testing.T) {
	for _, tc := range megaRAID2RunnerCases() {
		t.Run(tc.name, func(t *testing.T) {
			original := *tc.execCommand
			defer func() { *tc.execCommand = original }()

			var gotName string

			*tc.execCommand = func(name string, _ ...string) *exec.Cmd {
				gotName = name

				return exec.Command("printf", "%s", "{}")
			}

			custom := "/opt/custom/" + tc.name

			_, err := tc.newRunner(&custom).Run([]string{"show"})
			require.NoError(t, err)
			assert.Equal(t, custom, gotName)
		})
	}
}
