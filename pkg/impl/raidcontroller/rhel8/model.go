package rhel8

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

const (
	matchDeviceRegexpPattern = `MD_DEVICE_(.*)_(ROLE|DEV)=(.*)`
)

var matchDeviceRegexp = regexp.MustCompile(matchDeviceRegexpPattern)

type (
	MDADMExportDetails struct {
		RaidLevel    string                  // MD_LEVEL
		DevicesCount int                     // MD_DEVICES
		Metadata     string                  // MD_METADATA
		UUID         string                  // MD_UUID
		Name         string                  // MD_NAME
		ArraySize    string                  // MD_ARRAY_SIZE
		DeviceName   string                  // MD_DEVNAME
		Devices      map[string]MDADMDevices // MD_DEV_0, MD_DEV_1, ...
	}

	MDADMDevices struct {
		Role  string
		State string
		Path  string
	}
)

func ParseMDADMExportOutput(output []byte) ([]*MDADMExportDetails, error) {
	details := &MDADMExportDetails{}

	for _, line := range strings.Split(string(output), "\n") {
		switch {
		case strings.HasPrefix(line, "MD_LEVEL="):
			details.RaidLevel = strings.TrimPrefix(line, "MD_LEVEL=")
		case strings.HasPrefix(line, "MD_DEVICES="):
			_, err := fmt.Sscanf(line, "MD_DEVICES=%d", &details.DevicesCount)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse MD_DEVICES")
			}
		case strings.HasPrefix(line, "MD_METADATA="):
			details.Metadata = strings.TrimPrefix(line, "MD_METADATA=")
		case strings.HasPrefix(line, "MD_UUID="):
			details.UUID = strings.TrimPrefix(line, "MD_UUID=")
		case strings.HasPrefix(line, "MD_NAME="):
			details.Name = strings.TrimPrefix(line, "MD_NAME=")
		case strings.HasPrefix(line, "MD_DEVICE_"):
			if details.Devices == nil {
				details.Devices = make(map[string]MDADMDevices)
			}

			matches := matchDeviceRegexp.FindStringSubmatch(line)
			if len(matches) != 4 {
				return nil, errors.Errorf("invalid MD_DEVICE line: %s", line)
			}

			deviceName := matches[1]

			var device MDADMDevices

			if _, ok := details.Devices[deviceName]; !ok {
				details.Devices[deviceName] = MDADMDevices{}
			} else {
				device = details.Devices[deviceName]
			}

			if matches[2] == "ROLE" {
				device.Role = matches[3]
			} else if matches[2] == "DEV" {
				device.Path = matches[3]
			}

			details.Devices[deviceName] = device
		}
	}

	return []*MDADMExportDetails{details}, nil
}
