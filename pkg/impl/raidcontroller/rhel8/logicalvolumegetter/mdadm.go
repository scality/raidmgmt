package logicalvolumegetter

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/commandrunner"
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/domain/ports"
	"github.com/scality/raidmgmt/rhel8"
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

var (
	_            ports.LogicalVolumesGetter = &MDADM{}
	raidLevelMap                            = map[string]logicalvolume.RAIDLevel{ //nolint:gochecknoglobals,lll // Can't do anything about it
		"RAID0":  logicalvolume.RAIDLevel0,
		"RAID1":  logicalvolume.RAIDLevel1,
		"RAID10": logicalvolume.RAIDLevel10,
	}
)

func NewMDADM(
	runner commandrunner.CommandRunner,
) *MDADM {
	return &MDADM{
		CommandRunner: runner,
	}
}

func (m *MDADM) LogicalVolumes(
	metadata *raidcontroller.Metadata,
) ([]*logicalvolume.LogicalVolume, error) {
	detailCmdArguments := []string{
		"--detail",
		"--scan",
		"--export",
	}

	output, err := m.Run(detailCmdArguments)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan logical volumes")
	}

	details, err := rhel8.ParseMDADMExportOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse mdadm export output")
	}

	logicalVolumes := make([]*logicalvolume.LogicalVolume, 0, len(details))

	for _, detail := range details {
		logicalVolume := &logicalvolume.LogicalVolume{
			ID:              detail.UUID,
			DevicePath:      detail.DeviceName,
			PDrivesMetadata: make([]*physicaldrive.Metadata, detail.DevicesCount),
		}

		raidLevel := strings.ToUpper(detail.RaidLevel)

		logicalVolume.RAIDLevel = map[string]logicalvolume.RAIDLevel{
			"RAID0":  logicalvolume.RAIDLevel0,
			"RAID1":  logicalvolume.RAIDLevel1,
			"RAID10": logicalvolume.RAIDLevel10,
		}[raidLevel]

		if metadata != nil {
			logicalVolume.CtrlMetadata = metadata
		}

		logicalVolumes = append(logicalVolumes, logicalVolume)
	}

	return logicalVolumes, nil
}

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
	deviceName := metadata.ID // Here I assume that id is something like md0

	if !strings.HasPrefix(deviceName, "/dev/") {
		deviceName = "/dev/" + deviceName
	}

	output, err := m.Run([]string{
		"--detail",
		deviceName,
		"--export", // Export to get a key=value format output
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get details of logical volume %s", deviceName)
	}

	details, err := rhel8.ParseMDADMExportOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse mdadm export output")
	}

	raidLevel, ok := raidLevelMap[details[0].RaidLevel]
	if !ok {
		return nil, errors.Errorf("unknown RAID level: %s", details[0].RaidLevel)
	}

	// FIXME I think we can get more fields from the output
	logicalVolume := &logicalvolume.LogicalVolume{
		ID:              details[0].Name,
		CtrlMetadata:    metadata.CtrlMetadata,
		DevicePath:      details[0].DeviceName,
		RAIDLevel:       raidLevel,
		PDrivesMetadata: make([]*physicaldrive.Metadata, 0, details[0].DevicesCount),
	}

	for _, device := range details[0].Devices {
		logicalVolume.PDrivesMetadata = append(logicalVolume.PDrivesMetadata, &physicaldrive.Metadata{
			DevicePath: device.Path,
		})
	}

	return logicalVolume, nil
}
