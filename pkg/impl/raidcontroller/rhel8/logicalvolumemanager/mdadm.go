package logicalvolumemanager

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/commandrunner"
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/ports"
)

type MDADM struct {
	commandrunner.CommandRunner
	ports.LogicalVolumesGetter
}

var _ ports.LogicalVolumesManager = &MDADM{}

func NewMDADM(
	runner commandrunner.CommandRunner,
	getter ports.LogicalVolumesGetter,
) *MDADM {
	return &MDADM{
		CommandRunner:        runner,
		LogicalVolumesGetter: getter,
	}
}

func (m *MDADM) CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	// Validate input
	err := request.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate create logical volume request")
	}

	physicalDrivesName := make([]string, 0, len(request.PDrivesMetadata))

	for _, drive := range request.PDrivesMetadata {
		physicalDrivesName = append(physicalDrivesName, drive.DevicePath)
	}

	// Prepare the mdadm create command
	createCmdArgs := []string{
		"--create", fmt.Sprintf("/dev/%s", request.Name),
		"--level", strings.ToLower(logicalvolume.RAIDLevelMapToString[request.RAIDLevel]),
		"--raid-devices", fmt.Sprintf("%d", len(physicalDrivesName)),
	}

	// Add the physical drive names
	createCmdArgs = append(createCmdArgs, physicalDrivesName...)

	// Ignore the output
	// If there's an error, it doesn't matter, otherwise we already had it as parameter
	_, err = m.Run(createCmdArgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logical volume")
	}

	// Get the newly created logical volume metadata
	logicalVolume, err := m.LogicalVolume(&logicalvolume.Metadata{
		ID: request.ID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volume details")
	}

	return logicalVolume, nil
}

func (m *MDADM) DeleteLV(metadata *logicalvolume.Metadata) error {
	if metadata == nil {
		return errors.New("metadata is nil")
	}

	// Get the logical volume about to be deleted
	logicalVolume, err := m.LogicalVolume(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volume")
	}

	// Delete the logical volume
	err = m.deleteLV(logicalVolume)
	if err != nil {
		return errors.Wrap(err, "failed to delete logical volume")
	}

	return nil
}

func (m *MDADM) AddPDsToLV(
	lvMetadata *logicalvolume.Metadata,
	pvsMetadata ...*physicaldrive.Metadata,
) error {
	if lvMetadata == nil {
		return errors.New("logical volume metadata is nil")
	} else if pvsMetadata == nil {
		return errors.New("physical drive metadata is nil")
	}

	for _, pv := range pvsMetadata {
		if pv == nil {
			return errors.New("physical drive metadata is nil")
		}
	}

	// Get the logical volume
	logicalVolume, err := m.LogicalVolume(lvMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volume")
	}

	if logicalVolume.RAIDLevel == logicalvolume.RAIDLevel10 && len(pvsMetadata)%2 != 0 {
		return errors.New("cannot add an odd number of physical drives to a RAID10")
	}

	devicesPaths := make([]string, 0, len(pvsMetadata))
	for _, pvMetadata := range pvsMetadata {
		devicesPaths = append(devicesPaths, pvMetadata.DevicePath)
	}

	arrayLength := len(logicalVolume.PDrivesMetadata) + len(pvsMetadata)

	addCmd := []string{
		"--grow", logicalVolume.DevicePath,
		"--level", logicalvolume.RAIDLevelMapToString[logicalVolume.RAIDLevel],
		"--raid-devices", fmt.Sprintf("%d", arrayLength),
		"--add",
	}

	addCmd = append(addCmd, devicesPaths...)

	// Grow the array to include the new physical drives as a functioning part of the array
	_, err = m.Run(addCmd)
	if err != nil {
		return errors.Wrap(err, "failed to grow logical volume")
	}

	return nil
}

func (m *MDADM) DeletePDsFromLV(
	lvMetadata *logicalvolume.Metadata,
	pvsMetadata ...*physicaldrive.Metadata,
) error {
	if lvMetadata == nil {
		return errors.New("logical volume metadata is nil")
	} else if pvsMetadata == nil {
		return errors.New("physical drive metadata is nil")
	}

	for _, pv := range pvsMetadata {
		if pv == nil {
			return errors.New("physical drive metadata is nil")
		}
	}

	// Get the logical volume
	logicalVolume, err := m.LogicalVolume(lvMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volume")
	}

	if logicalVolume.RAIDLevel == logicalvolume.RAIDLevel0 {
		return errors.New("cannot remove physical drives from a RAID0")
	} else if logicalVolume.RAIDLevel == logicalvolume.RAIDLevel10 && len(pvsMetadata) > 1 {
		return errors.New("cannot remove more than one physical drive from a RAID10")
	}

	pvsDevicePaths := make([]string, 0, len(pvsMetadata))
	for _, pvMetadata := range pvsMetadata {
		pvsDevicePaths = append(pvsDevicePaths, pvMetadata.DevicePath)
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
		return errors.Wrap(err, "failed to change device state to failed in logical volume")
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
		return errors.Wrap(err, "failed to remove device from logical volume")
	}

	// Shrink the array to exclude the physical drive
	_, err = m.Run([]string{
		"--grow", logicalVolume.DevicePath,
		"--raid-devices", fmt.Sprintf("%d", len(logicalVolume.PDrivesMetadata)-len(pvsMetadata)),
	})
	if err != nil {
		return errors.Wrap(err, "failed to shrink logical volume")
	}

	return nil
}

func (m *MDADM) deleteLV(volume *logicalvolume.LogicalVolume) error {
	// Stop the array
	_, err := m.Run([]string{
		"--stop", volume.DevicePath,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to stop logical volume: %s", volume.DevicePath)
	}

	for _, device := range volume.PDrivesMetadata {
		// Remove the superblock of the device
		_, err = m.Run([]string{
			"--zero-superblock", device.DevicePath,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to zero superblock of physical drive: %s", device.DevicePath)
		}
	}

	return nil
}
