package megaraid

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

var (
	// STORCLI is the path to the storcli command.
	STORCLI = "/opt/hpe/storcli/storcli64"

	// PERCCLI is the path to the perccli command.
	PERCCLI = "/opt/MegaRAID/perccli/perccli64"
)

// Runner is an interface that defines the Run method.
//
//go:generate mockery --name=Runner --output=mocks --outpkg=mocks
type Runner interface {
	Run(args []string) (*CmdOutput, error)
}

// MegaRAIDRunner is a struct that implements the CmdRunner interface.
// It is used to run commands for the MegaRAID controller.
// Both storcli and perccli commands can be used.
type MegaRAIDRunner struct {
	cli string
}

// NewMegaRAIDRunner returns a new MegaRAIDRunner.
//
// If the path is "storcli" or "perccli", the default path will be used.
//
// If the path is a custom path, it will be used.
func NewMegaRAIDRunner(path string) *MegaRAIDRunner {
	if path == "storcli" {
		path = STORCLI
	}

	if path == "perccli" {
		path = PERCCLI
	}

	return &MegaRAIDRunner{
		cli: path,
	}
}

// Run runs a command with the given arguments.
func (mrr *MegaRAIDRunner) Run(args []string) (*CmdOutput, error) {
	// Add JSON output format
	args = append(args, "J")

	cmd := exec.Command(mrr.cli, args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	parsed, err := parse(output)
	if err != nil {
		return nil, err
	}

	// Check if there are any controllers
	if len(parsed.Controllers) == 0 {
		return nil, ErrNoControllersFound
	}

	// Check if the command was successful
	for _, controller := range parsed.Controllers {
		commandStatus := controller.CommandStatus

		if commandStatus.Status != "Success" {
			return nil, ParseError(commandStatus)
		}
	}

	return parsed, nil
}

// parse parses the output of the command and returns the parsed output.
func parse(data []byte) (*CmdOutput, error) {
	var out CmdOutput

	err := json.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// ParseError parses the error from the command status.
func ParseError(commandStatus CommandStatus) error {
	if len(commandStatus.DetailedStatus) > 0 {
		for _, ds := range commandStatus.DetailedStatus {
			if ds.Status != "Success" {
				if ds.Description != nil {
					return fmt.Errorf("%w: %s: %s", ErrCommandFailed, ds.ErrMsg, *ds.Description)
				}

				return fmt.Errorf("%w %s", ErrCommandFailed, ds.ErrMsg)
			}
		}
	}

	return fmt.Errorf("%w: %s", ErrCommandFailed, commandStatus.Description)
}
