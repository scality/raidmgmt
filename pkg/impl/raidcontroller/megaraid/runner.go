package megaraid

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

const (
	// STORCLI is the path to the storcli command.
	STORCLI = "/opt/MegaRAID/storcli/storcli64"

	// PERCCLI is the path to the perccli command.
	PERCCLI = "/opt/MegaRAID/perccli/perccli64"
)

// Runner is an interface that defines the Run method.
// It is used to run commands for the MegaRAID controller.
// Both storcli and perccli commands can be used.
type Runner interface {
	Run(args []string) (*CmdOutput, error)
}

// MegaRAIDRunner is a struct that implements the CmdRunner interface.
// It is used to run commands for the MegaRAID controller.
// Both storcli and perccli commands can be used.
// nolint: revive // The struct is poorly named but
// it will be resolved with Valentin's work and architecture
// TODO : Rename this struct according to Valentin's CommandRunner interface
type MegaRAIDRunner struct {
	cli string
}

// NewMegaRAIDRunner returns a new MegaRAIDRunner.
//
// If the path is "storcli" or "perccli", the default path will be used.
//
// If the path is a custom path, it will be used.
func NewMegaRAIDRunner(arg string) (*MegaRAIDRunner, error) {
	path := arg

	if arg == "storcli" {
		path = STORCLI
	}

	if arg == "perccli" {
		path = PERCCLI
	}

	// Check if the path exists
	if err := validatePath(arg); err != nil {
		return nil, errors.Wrap(err, "failed to validate path")
	}

	return &MegaRAIDRunner{
		cli: path,
	}, nil
}

func validatePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Wrapf(err, "path does not exist: %s", path)
		}

		return errors.Wrap(err, "error getting path info")
	}

	if info.IsDir() {
		return errors.Wrapf(err, "path is a directory: %s", path)
	}

	return nil
}

// Run runs a command with the given arguments.
func (mrr *MegaRAIDRunner) Run(args []string) (*CmdOutput, error) {
	// Add JSON output format
	argsJSON := append(args, "J")

	// Run the command
	//nolint:gosec // Disable gosec G204 linter as the command is not user input
	cmd := exec.Command(mrr.cli, argsJSON...)

	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run command")
	}

	// Parse the output
	parsed, err := parse(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse command output")
	}

	// Check if there are any controllers
	if len(parsed.Controllers) == 0 {
		return nil, errors.New("no controllers found")
	}

	// Check if the command was successful
	for _, controller := range parsed.Controllers {
		commandStatus := controller.CommandStatus

		if commandStatus.Status != "Success" {
			// Parse the error
			err := parseError(commandStatus)

			return nil, errors.Wrap(err, "error running command")
		}
	}

	return parsed, nil
}

// parse parses the output of the command and returns the parsed output.
func parse(data []byte) (*CmdOutput, error) {
	var out CmdOutput

	err := json.Unmarshal(data, &out)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal JSON")
	}

	return &out, nil
}

// parseError parses the error from the command status.
func parseError(commandStatus CommandStatus) error {
	if len(commandStatus.DetailedStatus) > 0 {
		for _, ds := range commandStatus.DetailedStatus {
			if ds.Status != "Success" {
				if ds.Description != nil {
					return errors.Errorf("%s: %s", ds.ErrMsg, *ds.Description)
				}

				return errors.New(ds.ErrMsg)
			}
		}
	}

	return errors.New(commandStatus.Description)
}
