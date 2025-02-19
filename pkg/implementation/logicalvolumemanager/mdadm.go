//nolint:cyclop,gocognit // Difficult to make those implementations smaller / less complicated
package logicalvolumemanager

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

const (
	raid1MinimalDriveCount  = 2
	raid10MinimalDriveCount = 4
)

type MDADM struct {
	commandrunner.CommandRunner
	ports.LogicalVolumesGetter
	ports.PhysicalDrivesGetter
}

var _ ports.LogicalVolumesManager = &MDADM{}

func NewMDADM(
	runner commandrunner.CommandRunner,
	getter ports.LogicalVolumesGetter,
	pdGetter ports.PhysicalDrivesGetter,
) *MDADM {
	return &MDADM{
		CommandRunner:        runner,
		LogicalVolumesGetter: getter,
		PhysicalDrivesGetter: pdGetter,
	}
}

func (m *MDADM) CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	physicalDrivesName := make([]string, 0, len(request.PDrivesMetadata))

	for _, drive := range request.PDrivesMetadata {
		physicalDrive, err := m.PhysicalDrive(drive)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get physical drive")
		}

		if physicalDrive.Status == physicaldrive.PDStatusFailed {
			return nil, errors.New("cannot create a logical volume with a failed physical drive")
		} else if physicalDrive.Status == physicaldrive.PDStatusUsed {
			return nil, errors.New("cannot create a logical volume with a used physical drive")
		}

		physicalDrivesName = append(physicalDrivesName, drive.DevicePath)
	}

	// Prepare the mdadm create command
	createCmdArgs := []string{
		"--create", fmt.Sprintf("/dev/%s", request.Name),
		"--level", string(request.RAIDLevel),
		"--raid-devices", fmt.Sprintf("%d", len(physicalDrivesName)),
	}

	// Add the physical drive names
	createCmdArgs = append(createCmdArgs, physicalDrivesName...)

	// Ignore the output
	_, err := m.Run(createCmdArgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run mdadm create logical volume command")
	}

	// Get the newly created logical volume metadata
	logicalVolume, err := m.LogicalVolume(&logicalvolume.Metadata{
		ID: request.ID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get logical volume: %s", request.ID)
	}

	return logicalVolume, nil
}

func (m *MDADM) DeleteLV(metadata *logicalvolume.Metadata) error {
	// Get the logical volume about to be deleted
	logicalVolume, err := m.LogicalVolume(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volume")
	}

	// Stop the array
	_, err = m.Run([]string{
		"--stop", logicalVolume.DevicePath,
	})
	if err != nil {
		return errors.Wrap(err, "failed to run mdadm stop logical volume command")
	}

	// Apply this command to every drives present in the logical volume
	// 	to make them available for reuse
	for _, device := range logicalVolume.PDrivesMetadata {
		// Remove the superblock of the device
		_, err = m.Run([]string{
			"--zero-superblock", device.DevicePath,
		})
		if err != nil {
			return errors.Wrapf(
				err,
				"failed to run mdadm zero superblock command on physical drive: %s",
				device.DevicePath,
			)
		}
	}

	return nil
}

//nolint:funlen // Lots of stuff to do
func (m *MDADM) AddPDsToLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	// Get the logical volume
	logicalVolume, err := m.LogicalVolume(lvMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volume")
	}

	if logicalVolume.Status == logicalvolume.LVStatusFailed {
		return errors.New("cannot add physical drives to a failed logical volume")
	}

	// RAID level checks
	if logicalVolume.RAIDLevel == logicalvolume.RAIDLevel10 && len(pdsMetadata)%2 != 0 {
		return errors.New("cannot add an odd number of physical drives to a RAID10")
	}

	devicesPaths := make([]string, 0, len(pdsMetadata))

	for _, pdMetadata := range pdsMetadata {
		physicalDrive, err := m.PhysicalDrive(pdMetadata)
		if err != nil {
			return errors.Wrap(err, "failed to get physical drive")
		}

		if physicalDrive.Status == physicaldrive.PDStatusFailed {
			return errors.New("cannot add a failed physical drive to a logical volume")
		} else if physicalDrive.Status == physicaldrive.PDStatusUsed {
			return errors.New("cannot add a used physical drive to a logical volume")
		}

		devicesPaths = append(devicesPaths, pdMetadata.DevicePath)
	}

	arrayLength := len(logicalVolume.PDrivesMetadata) + len(pdsMetadata)

	addCmd := []string{
		"--grow", logicalVolume.DevicePath,
		"--level", string(logicalVolume.RAIDLevel),
		"--raid-devices", fmt.Sprintf("%d", arrayLength),
		"--add",
	}

	addCmd = append(addCmd, devicesPaths...)

	// Add then grow the array
	_, err = m.Run(addCmd)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to run mdadm add / grow command with %s",
			strings.Join(devicesPaths, ","),
		)
	}

	return nil
}

//nolint:funlen // This function is long because of MDADM
func (m *MDADM) DeletePDsFromLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	// Get the logical volume
	logicalVolume, err := m.LogicalVolume(lvMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volume")
	}

	if logicalVolume.Status == logicalvolume.LVStatusFailed {
		return errors.New("cannot remove physical drives from a failed logical volume")
	}

	// RAID level checks
	switch logicalVolume.RAIDLevel { //nolint:exhaustive // We only support a subset of RAID levels
	case logicalvolume.RAIDLevel0:
		return errors.New("cannot remove physical drives from a RAID0")
	case logicalvolume.RAIDLevel1:
		if len(logicalVolume.PDrivesMetadata)-len(pdsMetadata) < raid1MinimalDriveCount {
			return errors.New("cannot remove physical drives from a RAID1 with less than 2 physical drives")
		}
	case logicalvolume.RAIDLevel10:
		if len(logicalVolume.PDrivesMetadata)-len(pdsMetadata) < raid10MinimalDriveCount {
			return errors.New("cannot remove physical drives from a RAID10 with less than 4 physical drives")
		}

		if len(logicalVolume.PDrivesMetadata)-len(pdsMetadata)%2 != 0 {
			return errors.New("cannot remove an odd number of physical drives from a RAID10")
		}
	default:
	}

	// Check that the drives to be removed are in the logical volume
	actualDrivesInLogicalVolumes := make([]*physicaldrive.Metadata, 0)

	for _, pdMetadata := range logicalVolume.PDrivesMetadata {
		for _, pd := range pdsMetadata {
			if pdMetadata.DevicePath == pd.DevicePath {
				actualDrivesInLogicalVolumes = append(actualDrivesInLogicalVolumes, pdMetadata)
			}
		}
	}

	pvsDevicePaths := make([]string, 0, len(actualDrivesInLogicalVolumes))
	for _, pdMetadata := range actualDrivesInLogicalVolumes {
		pvsDevicePaths = append(pvsDevicePaths, pdMetadata.DevicePath)
	}

	// Prepare the mdadm fail command
	// An active device cannot be removed from an array, so it needs to be marked as failed first.
	failCmd := []string{
		logicalVolume.DevicePath,
		"--fail",
	}

	// Append the list of physical drive to be set to failed
	failCmd = append(failCmd, pvsDevicePaths...)

	_, err = m.Run(failCmd)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to run mdadm fail physical drive command: %s",
			strings.Join(pvsDevicePaths, ","),
		)
	}

	// Prepare the mdadm remove command
	removeCmd := []string{
		"--remove",
		logicalVolume.DevicePath,
	}

	// Append the list of physical drive to be removed
	removeCmd = append(removeCmd, pvsDevicePaths...)

	_, err = m.Run(removeCmd)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to run mdadm remove command: %s",
			strings.Join(pvsDevicePaths, ","),
		)
	}

	currentSizeOfArray := uint64(0)

	for _, pdMetadata := range logicalVolume.PDrivesMetadata {
		physicalDrive, err := m.PhysicalDrive(pdMetadata)
		if err != nil {
			return errors.Wrap(err, "failed to get physical drive")
		}

		currentSizeOfArray += physicalDrive.Size
	}

	newSizeOfArray := currentSizeOfArray

	for _, pdMetadata := range pdsMetadata {
		physicalDrive, err := m.PhysicalDrive(pdMetadata)
		if err != nil {
			return errors.Wrap(err, "failed to get physical drive")
		}

		newSizeOfArray -= physicalDrive.Size
	}

	_, err = m.Run([]string{
		"--grow", logicalVolume.DevicePath,
		"--array-size", fmt.Sprintf("%d", newSizeOfArray),
		"--raid-devices", fmt.Sprintf("%d", len(logicalVolume.PDrivesMetadata)-len(pdsMetadata)),
	})
	if err != nil {
		return errors.Wrap(err, "failed to run mdadm grow command")
	}

	return nil
}
