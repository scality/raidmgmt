//nolint:cyclop,funlen // This package contains parser functions, which are inherently complex.
package physicaldrivegetter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

type (
	RHEL8 struct {
		UDevADM  commandrunner.CommandRunner
		LSBLK    commandrunner.CommandRunner
		SmartCTL commandrunner.CommandRunner
	}

	BlockDevice struct {
		DevicePath     string
		Size           uint64
		Rotational     string
		Type           string
		Tran           string
		MountPoint     string
		PartitionType  string
		FilesystemType string
	}
)

var _ ports.PhysicalDrivesGetter = &RHEL8{}

func NewRHEL8(
	uDevADMCommandRunner *commandrunner.UDevADM,
	lsblkCommandRunner *commandrunner.LSBLK,
	smartCTLCommandRunner *commandrunner.SmartCTL,
) *RHEL8 {
	return &RHEL8{
		UDevADM:  uDevADMCommandRunner,
		LSBLK:    lsblkCommandRunner,
		SmartCTL: smartCTLCommandRunner,
	}
}

func (r *RHEL8) PhysicalDrives(
	_ *raidcontroller.Metadata,
) ([]*physicaldrive.PhysicalDrive, error) {
	blockDevices, err := r.listBlockDevices()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list block devices")
	}

	physicalDrives := make([]*physicaldrive.PhysicalDrive, 0, len(blockDevices))

	for _, device := range blockDevices {
		physicalDrive, err := r.PhysicalDrive(&physicaldrive.Metadata{
			DevicePath: device.DevicePath,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive: %s", device.DevicePath)
		}

		physicalDrives = append(physicalDrives, physicalDrive)
	}

	return physicalDrives, nil
}

//nolint:gocognit // This function is complicated by essence.
func (r *RHEL8) PhysicalDrive(
	metadata *physicaldrive.Metadata,
) (*physicaldrive.PhysicalDrive, error) {
	device, err := r.getBlockDevice(metadata.DevicePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block device: %s", metadata.DevicePath)
	}

	physicalDrive := &physicaldrive.PhysicalDrive{}

	output, err := r.UDevADM.Run([]string{
		"info",
		"--query=all",
		"--name=" + metadata.DevicePath,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to run udevadm physical drive info command")
	}

	physicalDrive, err = ParseUDevADMOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse udevadm physical drive info command output")
	}

	physicalDrive.Metadata = metadata
	physicalDrive.Size = device.Size

	if device.MountPoint != "" || device.FilesystemType != "" || device.PartitionType != "" {
		physicalDrive.Status = physicaldrive.PDStatusUsed
	} else {
		status, err := r.physicalDriveStatus(device.DevicePath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive status: %s", device.DevicePath)
		}

		physicalDrive.Status = status
	}

	switch device.Rotational {
	default:
		physicalDrive.Type = physicaldrive.DiskTypeUnknown
	case "0": // Not a rotative disk, it's an SSD or NVMe
		switch device.Tran {
		case "sata":
			physicalDrive.Type = physicaldrive.DiskTypeSSD
		case "nvme":
			physicalDrive.Type = physicaldrive.DiskTypeNVMe
		}
	case "1":
		physicalDrive.Type = physicaldrive.DiskTypeHDD
	}

	return physicalDrive, nil
}

//nolint:gocognit //
func (r *RHEL8) physicalDriveStatus(devicePath string) (physicaldrive.PDStatus, error) {
	output, err := r.SmartCTL.Run([]string{
		"-a",
		devicePath,
	})
	if err != nil {
		return physicaldrive.PDStatusUnknown, errors.Wrap(
			err,
			"failed to get physical drive status with smartctl",
		)
	}

	smartCTLLines := strings.Split(string(output), "\n")

	var healthStatus string

	for _, line := range smartCTLLines {
		if strings.Contains(line, "overall-health") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				healthStatus = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	if healthStatus != "PASSED" {
		return physicaldrive.PDStatusFailed, nil
	}

	reallocatedCount := 0
	pendingCount := 0
	uncorrectableCount := 0

	for _, line := range smartCTLLines {
		re := regexp.MustCompile(`Reallocated Sector Count:\s+\d+`)
		if re.MatchString(line) {
			parts := strings.Fields(line)

			reallocatedCount, err = strconv.Atoi(parts[len(parts)-1])
			if err != nil {
				return physicaldrive.PDStatusUnknown, errors.Wrap(
					err,
					"failed to parse reallocated sector count",
				)
			}
		}

		re = regexp.MustCompile(`Current Pending Sector:\s+\d+`)
		if re.MatchString(line) {
			parts := strings.Fields(line)

			pendingCount, err = strconv.Atoi(parts[len(parts)-1])
			if err != nil {
				return physicaldrive.PDStatusUnknown, errors.Wrap(
					err,
					"failed to parse current pending sector count",
				)
			}
		}

		re = regexp.MustCompile(`Offline Uncorrectable:\s+\d+`)
		if re.MatchString(line) {
			parts := strings.Fields(line)

			uncorrectableCount, err = strconv.Atoi(parts[len(parts)-1])
			if err != nil {
				return physicaldrive.PDStatusUnknown, errors.Wrap(
					err,
					"failed to parse offline uncorrectable sector count",
				)
			}
		}
	}

	if reallocatedCount > 0 || pendingCount > 0 || uncorrectableCount > 0 {
		return physicaldrive.PDStatusFailed, nil
	}

	return physicaldrive.PDStatusUnassignedGood, nil
}

func (r *RHEL8) getBlockDevice(devicePath string) (*BlockDevice, error) {
	output, err := r.LSBLK.Run([]string{
		devicePath,
		"--paths",
		"--bytes",
		"--nodeps",
		"--output",
		"name,rota,size,type,tran,mountpoint,fstype,parttype",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block device using lsblk")
	}

	blockDevices, err := ParseLSBLKOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse lsblk command output")
	}

	if len(blockDevices) <= 0 {
		return nil, errors.Errorf("block device not found: %s", devicePath)
	}

	return &blockDevices[0], errors.Errorf("block device not found: %s", devicePath)
}

func (r *RHEL8) listBlockDevices() ([]BlockDevice, error) {
	output, err := r.LSBLK.Run([]string{
		"--list",
		"--paths",
		"--bytes",
		"--nodeps",
		"--output",
		"name,rota,size,type,tran,mountpoint,fstype,parttype",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to run list block devices command")
	}

	blockDevices, err := ParseLSBLKOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse lsblk command output")
	}

	return blockDevices, nil
}

//nolint:gocognit,nestif // Parser functions are complicated by essence.
func ParseUDevADMOutput(output []byte) (*physicaldrive.PhysicalDrive, error) {
	lines := strings.Split(string(output), "\n")
	physicalDrive := &physicaldrive.PhysicalDrive{}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "E: ID_MODEL="):
			physicalDrive.Model = strings.TrimPrefix(line, "E: ID_MODEL=")
		case strings.HasPrefix(line, "E: ID_SERIAL_SHORT="):
			physicalDrive.Serial = strings.TrimPrefix(line, "E: ID_SERIAL_SHORT=")
		case strings.HasPrefix(line, "E: ID_WWN="):
			physicalDrive.ID = strings.TrimPrefix(line, "E: ID_WWN=")
		case strings.HasPrefix(line, "E: DEVNAME="):
			if physicalDrive.Metadata == nil {
				physicalDrive.Metadata = &physicaldrive.Metadata{}
			}

			physicalDrive.DevicePath = strings.TrimPrefix(line, "E: DEVNAME=")
		case strings.HasPrefix(line, "E: DEVLINKS="):
			devlinks := strings.Split(strings.TrimPrefix(line, "E: DEVLINKS="), " ")
			for _, devlink := range devlinks {
				if len(devlink) > len(physicalDrive.PermanentPath) && strings.Contains(devlink, "by-id") {
					physicalDrive.PermanentPath = devlink
				}
			}
		}
	}

	return physicalDrive, nil
}

//nolint:gocognit,cyclop,funlen // Parser functions are complicated by essence.
func ParseLSBLKOutput(output []byte) ([]BlockDevice, error) {
	lines := strings.Split(string(output), "\n")
	//nolint:mnd // No need for a constant here.
	if len(lines) < 2 { // Check if there's at least a header and one device
		return nil, nil
	}

	header := strings.Fields(lines[0]) // Split the header line
	devices := []BlockDevice{}

	for _, line := range lines[1:] { // Skip the header line
		line = strings.TrimSpace(line) // remove leading/trailing spaces
		if line == "" {                // skip empty lines
			continue
		}

		fields := strings.Fields(line)

		device := BlockDevice{}

		for i, field := range fields {
			if header[i] != "" && len(header) > i {
				switch header[i] {
				case "NAME":
					device.DevicePath = field
				case "SIZE":
					size, err := strconv.ParseUint(field, 10, 64)
					if err != nil {
						return nil, errors.Wrap(err, "failed to parse size")
					}

					device.Size = size
				case "ROTA":
					device.Rotational = field
				case "TYPE":
					device.Type = field
				case "TRAN":
					device.Tran = field
				case "MOUNTPOINT":
					device.MountPoint = field
				case "FSTYPE":
					device.FilesystemType = field
				case "PARTTYPE":
					device.PartitionType = field
				}
			}
		}

		// Skip RAID devices, they will be listed later
		if strings.Contains(device.DevicePath, "md") {
			continue
		}

		devices = append(devices, device)
	}

	return devices, nil
}
