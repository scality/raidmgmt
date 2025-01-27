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
	details, err := rhel8.ParseMDADMExportOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse mdadm export output")
	}

	logicalVolumes := make([]*logicalvolume.LogicalVolume, 0, len(details))

	for _, detail := range details {
		// Fill the information about the logical volume
		logicalVolume := &logicalvolume.LogicalVolume{
			Metadata: &logicalvolume.Metadata{
				CtrlMetadata: metadata,
				ID:           detail.Name,
			},
			DevicePath:      detail.DeviceName,
			RAIDLevel:       detail.RaidLevel,
			PDrivesMetadata: make([]*physicaldrive.Metadata, detail.DevicesCount),
		}

		if metadata != nil {
			logicalVolume.CtrlMetadata = metadata
		}

		logicalVolumes = append(logicalVolumes, logicalVolume)
	}

	return logicalVolumes, nil
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
	deviceNamePrefix := "/dev/"

	if !strings.HasPrefix(metadata.ID, "md") {
		deviceNamePrefix = "/dev/md"
	}

	deviceName := deviceNamePrefix + metadata.ID

	// Get the details of the logical volume
	output, err := m.Run([]string{
		"--detail",
		deviceName,
		"--export", // Export to get a key=value format output
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get details of logical volume %s", deviceName)
	}

	// Parse the key=value output
	details, err := rhel8.ParseMDADMExportOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse mdadm export output")
	}

	logicalVolume := &logicalvolume.LogicalVolume{
		Metadata: &logicalvolume.Metadata{
			ID:           details[0].Name,
			CtrlMetadata: metadata.CtrlMetadata,
		},
		DevicePath:      details[0].DeviceName,
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
