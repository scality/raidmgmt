package logicalvolumegetter

import (
	"commandrunner"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/rhel8"
)

type (
	MDADM struct {
		commandrunner.CommandRunner
	}

	LogicalVolumesGetter interface {
		// LogicalVolumes returns a list of logical volumes for a given RAID controller
		LogicalVolumes(metadata *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error)

		// LogicalVolume returns a logical volume for a given metadata
		LogicalVolume(metadata *logicalvolume.Metadata) (*logicalvolume.LogicalVolume, error)
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

var _ LogicalVolumesGetter = &MDADM{}

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

		logicalVolume.RAIDLevel = logicalvolume.RAIDLevelMap[raidLevel]

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
	logicalVolume, err := m.logicalVolume(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volume")
	}

	return logicalVolume, nil
}

//nolint:revive // This is the wrapped LogicalVolume method, name is fine in this context.
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

	raidLevel, ok := logicalvolume.RAIDLevelMap[details[0].RaidLevel]
	if !ok {
		return nil, errors.Errorf("unknown RAID level: %s", details[0].RaidLevel)
	}

	// FIXME I think we can get more fields from the output
	logicalVolume := &logicalvolume.LogicalVolume{
		ID:           details[0].UUID,
		CtrlMetadata: metadata.CtrlMetadata,
		DevicePath:   details[0].DeviceName,
		RAIDLevel:    raidLevel,
	}

	return logicalVolume, nil
}
