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

const baseMDPath = "/dev/md"

type MDADM struct {
	MDADM commandrunner.CommandRunner
	ports.LogicalVolumesGetter
	ports.PhysicalDrivesGetter
}

var _ ports.LogicalVolumesManager = &MDADM{}

func NewMDADM(
	runner *commandrunner.MDADM,
	getter ports.LogicalVolumesGetter,
	pdGetter ports.PhysicalDrivesGetter,
) *MDADM {
	return &MDADM{
		MDADM:                runner,
		LogicalVolumesGetter: getter,
		PhysicalDrivesGetter: pdGetter,
	}
}

func (m *MDADM) CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	physicalDrivesName := make([]string, 0, len(request.PDrivesMetadata))

	for _, drive := range request.PDrivesMetadata {
		physicalDrive, err := m.PhysicalDrive(drive)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive : %s", drive.DevicePath)
		}

		if physicalDrive.Status == physicaldrive.PDStatusFailed {
			return nil, errors.New("cannot create a logical volume with a failed physical drive")
		} else if physicalDrive.Status == physicaldrive.PDStatusUsed {
			return nil, errors.New("cannot create a logical volume with a used physical drive")
		}

		physicalDrivesName = append(physicalDrivesName, drive.DevicePath)
	}

	devicePath := fmt.Sprintf("%s/%s", baseMDPath, request.Name)

	// Prepare the mdadm create command
	createCmdArgs := []string{
		"--create", devicePath,
		"--level", fmt.Sprintf("%d", request.RAIDLevel.Level()),
		"--raid-devices", fmt.Sprintf("%d", len(physicalDrivesName)),
	}

	// FIXME This might not be necessary on new disks (not used previously)
	if request.RAIDLevel == logicalvolume.RAIDLevel1 {
		createCmdArgs = append(createCmdArgs, "--metadata=0.90")
	}

	// Add the physical drive names
	createCmdArgs = append(createCmdArgs, physicalDrivesName...)

	// Ignore the output
	_, err := m.MDADM.Run(createCmdArgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run mdadm create logical volume command")
	}

	// Get the newly created logical volume metadata
	logicalVolume, err := m.LogicalVolume(&logicalvolume.Metadata{
		ID: devicePath,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get logical volume: %s", request.Name)
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
	_, err = m.MDADM.Run([]string{
		"--stop", logicalVolume.DevicePath,
	})
	if err != nil {
		return errors.Wrap(err, "failed to run mdadm stop logical volume command")
	}

	// Apply this command to every drives present in the logical volume
	// 	to make them available for reuse
	for _, device := range logicalVolume.PDrivesMetadata {
		// Remove the superblock of the device
		_, err = m.MDADM.Run([]string{
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

	devicesPaths := make([]string, 0, len(pdsMetadata))

	for _, pdMetadata := range pdsMetadata {
		physicalDrive, err := m.PhysicalDrive(pdMetadata)
		if err != nil {
			return errors.Wrap(err, "failed to get physical drive")
		}

		if physicalDrive.Status == physicaldrive.PDStatusFailed {
			return errors.Errorf(
				"cannot add a failed physical drive to a logical volume : %s ", physicalDrive.DevicePath,
			)
		} else if physicalDrive.Status == physicaldrive.PDStatusUsed {
			return errors.Errorf(
				"cannot add a used physical drive to a logical volume : %s", physicalDrive.DevicePath,
			)
		}

		devicesPaths = append(devicesPaths, pdMetadata.DevicePath)
	}

	if logicalVolume.RAIDLevel == logicalvolume.RAIDLevel10 ||
		logicalVolume.RAIDLevel == logicalvolume.RAIDLevel1 {
		// Add the devices to the array
		_, err = m.MDADM.Run(append([]string{
			logicalVolume.DevicePath,
			"--add",
		}, devicesPaths...))
		if err != nil {
			return errors.Wrapf(
				err,
				"failed to run mdadm add command with %s",
				strings.Join(devicesPaths, ","),
			)
		}

		// Enhance the size of the array
		_, err = m.MDADM.Run([]string{
			"--grow", logicalVolume.DevicePath,
			"--array-size=max",
		})
		if err != nil {
			return errors.Wrap(err, "failed to run mdadm grow command to adapt array size")
		}

		return nil
	}

	arrayLength := len(logicalVolume.PDrivesMetadata) + len(pdsMetadata)

	// This below is valid for raid0
	addCmd := []string{
		"--grow", logicalVolume.DevicePath,
		"--level", fmt.Sprintf("%d", logicalVolume.RAIDLevel.Level()),
		"--raid-devices", fmt.Sprintf("%d", arrayLength),
		"--add",
	}

	addCmd = append(addCmd, devicesPaths...)

	// Add then grow the array
	_, err = m.MDADM.Run(addCmd)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to run mdadm add / grow command on %s",
			strings.Join(devicesPaths, ","),
		)
	}

	// FIXME Might need a loop that wait here

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
		if len(logicalVolume.PDrivesMetadata)-len(pdsMetadata) < logicalvolume.RAID1DiskRequirement {
			return errors.New("cannot remove physical drives from a RAID1 with a single physical drive")
		}
	case logicalvolume.RAIDLevel10:
	default:
	}

	pdsDevicePaths := make([]string, 0, len(pdsMetadata))
	for _, pdMetadata := range pdsMetadata {
		pdsDevicePaths = append(pdsDevicePaths, pdMetadata.DevicePath)
	}

	// Prepare the mdadm fail command
	// An active device cannot be removed from an array, so it needs to be marked as failed first.
	failCmd := []string{
		"--fail",
		logicalVolume.DevicePath,
	}

	// Append the list of physical drive to be set to failed
	failCmd = append(failCmd, pdsDevicePaths...)

	_, err = m.MDADM.Run(failCmd)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to run mdadm fail physical drive command: %s",
			strings.Join(pdsDevicePaths, ","),
		)
	}

	// Prepare the mdadm remove command
	removeCmd := []string{
		"--remove",
		logicalVolume.DevicePath,
	}

	// Append the list of physical drive to be removed
	removeCmd = append(removeCmd, pdsDevicePaths...)

	_, err = m.MDADM.Run(removeCmd)
	if err != nil {
		return errors.Wrap(err, "failed to run mdadm remove command")
	}

	zeroCmd := []string{
		"--zero-superblock",
	}

	zeroCmd = append(zeroCmd, pdsDevicePaths...)

	_, err = m.MDADM.Run(zeroCmd)
	if err != nil {
		return errors.Wrap(err, "failed to run mdadm zero superblock command")
	}

	// Cannot shrink a RAID10
	if logicalVolume.RAIDLevel == logicalvolume.RAIDLevel10 {
		return nil
	}

	// Reduce the device count of the array
	_, err = m.MDADM.Run([]string{
		"--grow", logicalVolume.DevicePath,
		"--raid-devices", fmt.Sprintf("%d", len(logicalVolume.PDrivesMetadata)-len(pdsMetadata)),
	})
	if err != nil {
		return errors.Wrap(err, "failed to run mdadm grow command")
	}

	// Reduce the size of the array
	_, err = m.MDADM.Run([]string{
		"--grow", logicalVolume.DevicePath,
		"--array-size=max",
	})
	if err != nil {
		return errors.Wrap(err, "failed to run mdadm grow command to adapt array size")
	}

	return nil
}
