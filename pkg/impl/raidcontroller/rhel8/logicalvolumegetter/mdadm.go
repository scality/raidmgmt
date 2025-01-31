package logicalvolumegetter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/impl/commandrunner"
)

type (
	MDADM struct {
		commandrunner.CommandRunner
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

var _ ports.LogicalVolumesGetter = &MDADM{}

func NewMDADM(
	runner commandrunner.CommandRunner,
) *MDADM {
	return &MDADM{
		CommandRunner: runner,
	}
}

// LogicalVolumes returns all the logical volumes on the system.
func (m *MDADM) LogicalVolumes(
	metadata *raidcontroller.Metadata,
) ([]*logicalvolume.LogicalVolume, error) {
	// List existing logical volumes
	output, err := m.Run([]string{
		"--detail",
		"--scan",
		"--export", // Export to get a key=value format output
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan logical volumes")
	}

	// Parse the key=value output
	details, err := ParseMDADMExportOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse mdadm export output")
	}

	logicalVolumes := make([]*logicalvolume.LogicalVolume, 0, len(details))

	for _, detail := range details {
		devicePath := deviceNameToDevicePath(detail.Name)

		// Fill the information about the logical volume
		logicalVolume := &logicalvolume.LogicalVolume{
			Metadata: &logicalvolume.Metadata{
				CtrlMetadata: metadata,
				ID:           detail.Name,
			},
			DevicePath:      devicePath,
			RAIDLevel:       detail.RaidLevel,
			PDrivesMetadata: make([]*physicaldrive.Metadata, 0, detail.DevicesCount),
		}

		for _, device := range detail.Devices {
			logicalVolume.PDrivesMetadata = append(logicalVolume.PDrivesMetadata, &physicaldrive.Metadata{
				DevicePath:   device.Path,
				CtrlMetadata: metadata,
			})
		}

		if metadata != nil {
			logicalVolume.CtrlMetadata = metadata
		}

		logicalVolumes = append(logicalVolumes, logicalVolume)
	}

	return logicalVolumes, nil
}

const deviceNameRegexPattern = "^._.$"

var deviceNameRegex = regexp.MustCompile(deviceNameRegexPattern)

func deviceNameToDevicePath(deviceName string) string {
	if deviceNameRegex.MatchString(deviceName) {
		return fmt.Sprintf("/dev/md/%s", deviceName)
	}

	if strings.HasPrefix(deviceName, "/dev/") {
		return deviceName
	}

	if !strings.HasPrefix(deviceName, "md") {
		return fmt.Sprintf("/dev/md%s", deviceName)
	}

	return fmt.Sprintf("/dev/%s", deviceName)
}

// LogicalVolume returns a logical volume by its metadata.
func (m *MDADM) LogicalVolume(
	metadata *logicalvolume.Metadata,
) (*logicalvolume.LogicalVolume, error) {
	if metadata == nil {
		return nil, errors.New("metadata is nil")
	}

	logicalVolume, err := m.logicalVolume(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volume")
	}

	return logicalVolume, nil
}

func (m *MDADM) logicalVolume(
	metadata *logicalvolume.Metadata,
) (*logicalvolume.LogicalVolume, error) {
	// It is assumed that the ID is the suffix of the device name
	// 	md0, md1, md/0_0 should also be supported
	// deviceNamePrefix := "/dev/"
	//
	// if !strings.HasPrefix(metadata.ID, "md") {
	// 	deviceNamePrefix = "/dev/md"
	// }
	//
	// deviceName := deviceNamePrefix + metadata.ID
	devicePath := deviceNameToDevicePath(metadata.ID)

	// Get the details of the logical volume
	output, err := m.Run([]string{
		"--detail",
		devicePath,
		"--export", // Export to get a key=value format output
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get details of logical volume %s", devicePath)
	}

	// Parse the key=value output
	details, err := ParseMDADMExportOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse mdadm export output")
	}

	logicalVolume := &logicalvolume.LogicalVolume{
		Metadata: &logicalvolume.Metadata{
			ID:           details[0].Name,
			CtrlMetadata: metadata.CtrlMetadata,
		},
		DevicePath:      devicePath,
		RAIDLevel:       details[0].RaidLevel,
		PDrivesMetadata: make([]*physicaldrive.Metadata, 0, details[0].DevicesCount),
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
