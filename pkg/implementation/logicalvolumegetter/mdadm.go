//nolint:cyclop,gocognit // Command parser are generally complex.
package logicalvolumegetter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

const (
	mdadmDeviceNameRegexPattern = "(.*_[0-9])"

	mdadmMatchDeviceRegexpPattern  = `MD_DEVICE_(.*)_(ROLE|DEV)=(.*)`
	mdadmMatchDeviceRegexpPattern2 = `MD_LEVEL`
)

type (
	MDADM struct {
		commandrunner.CommandRunner
	}

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

	ExportDetails struct {
		RaidLevel   string   // MD_LEVEL
		Devices     int      // MD_DEVICES
		Metadata    string   // MD_METADATA
		UUID        string   // MD_UUID
		Name        string   // MD_NAME
		ArraySize   string   // MD_ARRAY_SIZE
		DeviceName  string   // MD_DEVNAME
		DevicePaths []string // MD_DEV_0, MD_DEV_1, ...
	}
)

var (
	_ ports.LogicalVolumesGetter = &MDADM{}

	mdadmDeviceNameRegex    = regexp.MustCompile(mdadmDeviceNameRegexPattern)
	mdadmMatchDeviceRegexp  = regexp.MustCompile(mdadmMatchDeviceRegexpPattern)
	mdadmMatchDeviceRegexp2 = regexp.MustCompile(mdadmMatchDeviceRegexpPattern2)
)

func NewMDADM(
	runner *commandrunner.MDADM,
) *MDADM {
	return &MDADM{
		CommandRunner: runner,
	}
}

// LogicalVolumes returns all the logical volumes on the system.
func (m *MDADM) LogicalVolumes(
	_ *raidcontroller.Metadata,
) ([]*logicalvolume.LogicalVolume, error) {
	// List existing logical volumes
	output, err := m.Run([]string{
		"--detail",
		"--scan",
		"--export", // Export to get a key=value format output
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to run mdadm detail scan export command")
	}

	// Parse the key=value output
	details, err := ParseMDADMExportOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse mdadm scan output")
	}

	logicalVolumes := make([]*logicalvolume.LogicalVolume, 0, len(details))

	for _, detail := range details {
		logicalVolume, err := m.LogicalVolume(&logicalvolume.Metadata{
			ID: detail.Name,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get logical volume")
		}

		logicalVolumes = append(logicalVolumes, logicalVolume)
	}

	return logicalVolumes, nil
}

// LogicalVolume returns a logical volume by its metadata.
func (m *MDADM) LogicalVolume(
	metadata *logicalvolume.Metadata,
) (*logicalvolume.LogicalVolume, error) {
	// It is assumed that the ID is the suffix of the device name
	// 	md0, md1, md/0_0 should also be supported
	devicePath := DeviceNameToDevicePath(metadata.ID)

	logicalVolumeStatus, logicalVolumeSize, err := m.getLogicalVolumeStatusAndSize(devicePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volume status")
	}

	// Get the details of the logical volume
	output, err := m.Run([]string{
		"--detail",
		devicePath,
		"--export", // Export to get a key=value format output
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to run mdadm detail export command")
	}

	// Parse the key=value output
	details, err := ParseMDADMExportOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse mdadm detail export output")
	}

	logicalVolume := &logicalvolume.LogicalVolume{
		Status: logicalVolumeStatus,
		Metadata: &logicalvolume.Metadata{
			ID:           details[0].Name,
			CtrlMetadata: metadata.CtrlMetadata,
		},
		DevicePath:      devicePath,
		RAIDLevel:       details[0].RaidLevel,
		PDrivesMetadata: make([]*physicaldrive.Metadata, 0, details[0].DevicesCount),
		Size:            logicalVolumeSize,
	}

	for _, device := range details[0].Devices {
		logicalVolume.PDrivesMetadata = append(logicalVolume.PDrivesMetadata, &physicaldrive.Metadata{
			DevicePath: device.Path,
			// FIXME Add a const in the controller metadata to identify the controller
			CtrlMetadata: metadata.CtrlMetadata,
		})
	}

	return logicalVolume, nil
}

//nolint:funlen // Come on, one line too long
func (m *MDADM) getLogicalVolumeStatusAndSize(devicePath string) (
	logicalvolume.LVStatus,
	uint64,
	error,
) {
	output, err := m.Run([]string{
		"--detail",
		devicePath,
	})
	if err != nil {
		return logicalvolume.LVStatusUnknown, 0, errors.Wrap(err, "failed to run mdadm detail command")
	}

	logicalVolumeStatus := logicalvolume.LVStatusUnknown

	var arraySize uint64

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "State : ") {
			switch strings.TrimPrefix(line, "State : ") {
			case "degraded":
				logicalVolumeStatus = logicalvolume.LVStatusDegraded
			case "active":
				logicalVolumeStatus = logicalvolume.LVStatusOptimal
			case "failed":
				logicalVolumeStatus = logicalvolume.LVStatusFailed
			}

			break
		}

		if strings.Contains(line, "Array Size :") {
			arraySizeLineSplit := strings.Split(
				strings.TrimSpace(
					strings.TrimPrefix(line, "Array Size :"),
				), " ",
			)

			arraySize, err = strconv.ParseUint(arraySizeLineSplit[0], 10, 64)
			if err != nil {
				return logicalVolumeStatus, 0, errors.Wrap(err, "failed to parse array size")
			}
		}
	}

	if arraySize == 0 {
		return logicalVolumeStatus, 0, errors.New("failed to get array size")
	}

	return logicalVolumeStatus, arraySize, nil
}

//nolint:gocognit,funlen,cyclop // This function is complex by nature
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
			case strings.HasPrefix(line, "MD_DEVNAME="):
				currentDetails.DeviceName = strings.TrimPrefix(line, "MD_DEVNAME=")
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

func splitOutputOnMDLevel(output []byte) [][]byte {
	devicesIndexes := mdadmMatchDeviceRegexp2.FindAllIndex(output, -1)

	block := make([][]byte, 0)

	index := 0

	for i, matchIndex := range devicesIndexes {
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

func DeviceNameToDevicePath(deviceName string) string {
	// User already provided the full path
	// Could be /dev/md0, /dev/md/0_0, /dev/md/toto_1
	if strings.HasPrefix(deviceName, "/dev/md") {
		return deviceName
	}

	// User provided the device name only, with a slash
	// Could be md/0_0, md/toto_1
	if strings.HasPrefix(deviceName, "md/") {
		return fmt.Sprintf("/dev/%s", deviceName)
	}

	// Matches md/0_0, md/toto_1
	if mdadmDeviceNameRegex.MatchString(deviceName) {
		return fmt.Sprintf("/dev/md/%s", deviceName)
	}

	return fmt.Sprintf("/dev/md/%s", deviceName)
}
