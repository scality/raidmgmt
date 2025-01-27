//nolint:cyclop // Come on
package rhel8

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
)

const (
	mdadmMatchDeviceRegexpPattern  = `MD_DEVICE_(.*)_(ROLE|DEV)=(.*)`
	mdadmMatchDeviceRegexpPattern2 = `MD_LEVEL`
)

var (
	mdadmMatchDeviceRegexp  = regexp.MustCompile(mdadmMatchDeviceRegexpPattern)
	mdadmMatchDeviceRegexp2 = regexp.MustCompile(mdadmMatchDeviceRegexpPattern2)
)

type (
	MDADMExportDetails struct {
		RaidLevel    logicalvolume.RAIDLevel // MD_LEVEL
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

func splitOutputOnMDLevel(output []byte) [][]byte {
	pouet := mdadmMatchDeviceRegexp2.FindAllIndex(output, -1)

	block := make([][]byte, 0)

	index := 0

	for i, matchIndex := range pouet {
		if i == 0 {
			continue
		}

		currentIndex := matchIndex[0]

		currentBlock := output[index:currentIndex]

		block = append(block, currentBlock)

		index = matchIndex[0]
	}

	return append(block, output[index:])
}

//nolint:gocognit,funlen // This function is complex by nature
func ParseMDADMExportOutput(output []byte) ([]*MDADMExportDetails, error) {
	if len(output) == 0 || output == nil {
		return []*MDADMExportDetails{}, nil
	}

	blocks := splitOutputOnMDLevel(output)

	details := make([]*MDADMExportDetails, 0, len(blocks))

	for _, block := range blocks {
		currentDetails := &MDADMExportDetails{}

		for _, line := range strings.Split(string(block), "\n") {
			switch {
			case strings.HasPrefix(line, "MD_LEVEL="):
				raidLevel := strings.TrimPrefix(line, "MD_LEVEL=")

				currentDetails.RaidLevel = logicalvolume.RAIDLevelMap(raidLevel)
			case strings.HasPrefix(line, "MD_DEVICES="):
				_, err := fmt.Sscanf(line, "MD_DEVICES=%d", &currentDetails.DevicesCount)
				if err != nil {
					return nil, errors.Wrap(err, "failed to parse MD_DEVICES")
				}
			case strings.HasPrefix(line, "MD_METADATA="):
				currentDetails.Metadata = strings.TrimPrefix(line, "MD_METADATA=")
			case strings.HasPrefix(line, "MD_UUID="):
				currentDetails.UUID = strings.TrimPrefix(line, "MD_UUID=")
			case strings.HasPrefix(line, "MD_NAME="):
				currentDetails.Name = strings.TrimPrefix(line, "MD_NAME=")
			case strings.HasPrefix(line, "MD_DEVICE_"):
				if currentDetails.Devices == nil {
					currentDetails.Devices = make(map[string]MDADMDevices)
				}

				matches := mdadmMatchDeviceRegexp.FindStringSubmatch(line)
				if len(matches) != 4 { //nolint:mnd // Expected length
					return nil, errors.Errorf("invalid MD_DEVICE line: %s", line)
				}

				deviceName := matches[1]

				var device MDADMDevices

				if _, ok := currentDetails.Devices[deviceName]; !ok {
					currentDetails.Devices[deviceName] = MDADMDevices{}
				} else {
					device = currentDetails.Devices[deviceName]
				}

				if matches[2] == "ROLE" {
					device.Role = matches[3]
				} else if matches[2] == "DEV" {
					device.Path = matches[3]
				}

				currentDetails.Devices[deviceName] = device
			}
		}

		details = append(details, currentDetails)
	}

	return details, nil
}
