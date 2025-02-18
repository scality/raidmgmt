//nolint:cyclop // This package contains parser functions, which are inherently complex.
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
		DevicePath string
		Size       uint64
		Rotational string
		Type       string
		Tran       string
		MountPoint string
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
	blockDevices, err := r.listBlockDevices()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list block devices")
	}

	physicalDrive := &physicaldrive.PhysicalDrive{}

	for _, device := range blockDevices {
		if device.DevicePath == metadata.DevicePath {
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

			if device.MountPoint != "" {
				physicalDrive.Status = physicaldrive.PDStatusUsed
			} else {
				status, err := r.physicalDriveStatus(device.DevicePath)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get physical drive status")
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

			break
		}
	}

	return physicalDrive, nil
}

func (r *RHEL8) physicalDriveStatus(devicePath string) (physicaldrive.PDStatus, error) {
	output, err := r.SmartCTL.Run([]string{
		"-a",
		devicePath,
	})
	if err != nil {
		return physicaldrive.PDStatusUnknown, errors.Wrap(err, "failed to run smartctl command")
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
			reallocatedCount, _ = strconv.Atoi(parts[len(parts)-1])
		}

		re = regexp.MustCompile(`Current Pending Sector:\s+\d+`)
		if re.MatchString(line) {
			parts := strings.Fields(line)
			pendingCount, _ = strconv.Atoi(parts[len(parts)-1])
		}

		re = regexp.MustCompile(`Offline Uncorrectable:\s+\d+`)
		if re.MatchString(line) {
			parts := strings.Fields(line)
			uncorrectableCount, _ = strconv.Atoi(parts[len(parts)-1])
		}
	}

	if reallocatedCount > 0 || pendingCount > 0 || uncorrectableCount > 0 {
		return physicaldrive.PDStatusFailed, nil
	}

	return physicaldrive.PDStatusUnassignedGood, nil
}

func (r *RHEL8) listBlockDevices() ([]BlockDevice, error) {
	output, err := r.LSBLK.Run([]string{
		"--list",
		"--paths",
		"--bytes",
		"--nodeps",
		"--output",
		"name,rota,size,type,tran,mountpoint",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to run lsblk command")
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
				}
			}
		}

		// Skip RAID devices, they will be listed later
		// Also ignore partitions, only interested in physical drives
		if device.Type == "part" || strings.Contains(device.DevicePath, "md") {
			continue
		}

		devices = append(devices, device)
	}

	return devices, nil
}
