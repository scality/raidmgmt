package logicalvolumemanager

import (
	"fmt"
	"strconv"

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

func (m *MDADM) createLV(
	logicalVolumeName string,
	raidLevel int,
	physicalDriveNames []string,
) (*logicalvolume.LogicalVolume, error) {
	// Prepare the mdadm create command
	createCmdArgs := []string{
		"--create", logicalVolumeName,
		"--level", fmt.Sprintf("%d", raidLevel),
		// FIXME Think about this:
		// --force basically remove the need to type y to confirm the creation of the array
		// It is useful to automate the process but could be dangerous if used wrong.
		// Is the purpose of this lib to be safe or just work ?
		// This problem can be solved by injecting something along the lines of "yes | the command" in the Run function
		// It means that we either have to complexify the Run function or add a new one that does that, thus ignoring the interface
		"--raid-devices", fmt.Sprintf("%d", len(physicalDriveNames)),
		"--force",
	}

	// Add the physical drive names
	createCmdArgs = append(createCmdArgs, physicalDriveNames...)

	// Ignore the ouput
	// If there's an error, it doesn't matter, otherwise we already had it as parameter
	_, err := m.Run(createCmdArgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logical volume")
	}

	// Get the logical volume metadata
	logicalVolume, err := m.LogicalVolume(&logicalvolume.Metadata{
		ID: logicalVolumeName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volume details")
	}

	return logicalVolume, nil
}

func (m *MDADM) CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	// Validate input
	err := request.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate create logical volume request")
	}

	// Check if there are enough devices to create a RAID array
	physicalDriveCount := len(request.PDrivesMetadata)
	if physicalDriveCount < 2 {
		return nil, errors.Errorf(
			"at least two devices are required to create a RAID array, %d provided", physicalDriveCount,
		)
	}

	existingLogicalVolumes, err := m.LogicalVolumesGetter.LogicalVolumes(request.CtrlMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing logical volumes")
	}

	existingVolumeIDs := make([]int, 0, len(existingLogicalVolumes))

	for _, logicalVolume := range existingLogicalVolumes {
		id, err := strconv.Atoi(logicalVolume.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse logical volume ID")
		}

		existingVolumeIDs = append(existingVolumeIDs, id)
	}

	smallestPositive := func(array []int) int {
		store := make(map[int]bool)

		for _, num := range array {
			store[num] = true
		}

		selected := 0

		for {
			_, ok := store[selected]
			if !ok {
				break
			}

			selected++
		}

		return selected
	}(existingVolumeIDs)

	logicalVolumeName := fmt.Sprintf("/dev/md%d", smallestPositive)

	physicalDrivesNames := make([]string, 0, physicalDriveCount)

	for _, drive := range request.PDrivesMetadata {
		physicalDrivesNames = append(physicalDrivesNames, drive.DevicePath)
	}

	logicalVolume, err := m.createLV(logicalVolumeName, int(request.RAIDLevel), physicalDrivesNames)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logical volume")
	}

	return logicalVolume, nil
}

func (m *MDADM) DeleteLV(metadata *logicalvolume.Metadata) error {
	if metadata == nil {
		return errors.New("metadata is nil")
	}

	logicalVolume, err := m.LogicalVolume(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volume")
	}

	err = m.deleteLV(logicalVolume)
	if err != nil {
		return errors.Wrap(err, "failed to delete logical volume")
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

	// Try to remove the array
	// From my tests, it often fails because mdadm still find the superblock of the device.
	_, err = m.Run([]string{
		"--remove", volume.DevicePath,
	})
	if err == nil {
		return nil
	}

	// If it fails, try to zero the superblock the devices of the array
	_, err = m.Run([]string{
		"--zero-superblock", volume.DevicePath,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to zero superblock of logical volume: %s", volume.DevicePath)
	}

	return nil
}

func (m *MDADM) AddPDToLV(
	lvMetadata *logicalvolume.Metadata,
	pvMetadata *physicaldrive.Metadata,
) error {
	if lvMetadata == nil {
		return errors.New("logical volume metadata is nil")
	} else if pvMetadata == nil {
		return errors.New("physical drive metadata is nil")
	}

	logicalVolume, err := m.LogicalVolume(lvMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volume")
	}

	// Adding the new physical drive to the logical volume will make it a spare.
	_, err = m.Run([]string{
		"--add", logicalVolume.DevicePath, pvMetadata.DevicePath,
	})
	if err != nil {
		return errors.Wrap(err, "failed to add device to logical volume")
	}

	// Grow the array to include the new physical drive as a functioning part of the array
	_, err = m.Run([]string{
		"--grow", logicalVolume.DevicePath,
		"--raid-devices", fmt.Sprintf("%d", len(logicalVolume.PDrivesMetadata)+1),
	})
	if err != nil {
		return errors.Wrap(err, "failed to grow logical volume")
	}

	return nil
}

func (m *MDADM) DeletePDFromLV(
	lvMetadata *logicalvolume.Metadata,
	pvMetadata *physicaldrive.Metadata,
) error {
	if lvMetadata == nil {
		return errors.New("logical volume metadata is nil")
	} else if pvMetadata == nil {
		return errors.New("physical drive metadata is nil")
	}

	logicalVolume, err := m.LogicalVolume(lvMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volume")
	}

	// An active device can't be removed from an array, so it needs to be marked as failed first.
	_, err = m.Run([]string{
		logicalVolume.DevicePath,
		"--fail", pvMetadata.DevicePath,
	})
	if err != nil {
		return errors.Wrap(err, "failed to change device state to failed in logical volume")
	}

	// Remove the physical drive from the logical volume
	_, err = m.Run([]string{
		"--remove", logicalVolume.DevicePath, pvMetadata.DevicePath,
	})
	if err != nil {
		return errors.Wrap(err, "failed to remove device from logical volume")
	}

	// Shrink the array to exclude the physical drive
	_, err = m.Run([]string{
		"--grow", logicalVolume.DevicePath,
		"--raid-devices", fmt.Sprintf("%d", len(logicalVolume.PDrivesMetadata)-1),
	})
	if err != nil {
		return errors.Wrap(err, "failed to shrink logical volume")
	}

	return nil
}
