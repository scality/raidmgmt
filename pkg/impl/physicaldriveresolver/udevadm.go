package physicaldriveresolver

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/commandrunner"
)

type UDevADM struct {
	commandrunner.CommandRunner
}

func NewUDevADM(commandRunner commandrunner.CommandRunner) *UDevADM {
	return &UDevADM{
		CommandRunner: commandRunner,
	}
}

func (u *UDevADM) ResolvePhysicalDriveDeviceNameFromID(diskID string) (string, error) {
	queryCmd := []string{
		"info",
		"--query=name",
		"--name=" + fmt.Sprintf(devDiskByIDPathFormat, diskID),
	}

	output, err := u.Run(queryCmd)
	if err != nil {
		// Error code 2 if not found
		return "", errors.Wrap(err, "failed to query physical drive device name")
	}

	// The output of the udevadm command is the device name followed by a newline character
	return strings.Trim(string(output), "\n"), nil
}
