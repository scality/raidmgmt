package physicaldrivegetter

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/impl/commandrunner"
)

type (
	RHEL8 struct {
		uDevADM *commandrunner.UDevADM
		lsblk   *commandrunner.LSBLK
	}

	BlockDevice struct {
		DevicePath string
		MajMin     string
		RM         string
		Size       uint64
		RO         string
		Type       string
		Mountpoint string
	}
)

var _ ports.PhysicalDrivesGetter = &RHEL8{}

func NewRHEL8(
	uDevADMCommandRunner *commandrunner.UDevADM,
	lsblkCommandRunner *commandrunner.LSBLK,
) *RHEL8 {
	return &RHEL8{
		uDevADM: uDevADMCommandRunner,
		lsblk:   lsblkCommandRunner,
	}
}

func (r *RHEL8) PhysicalDrives(metadata *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error) {
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
			return nil, errors.Wrap(err, "failed to get physical drive")
		}

		physicalDrives = append(physicalDrives, physicalDrive)
	}

	return physicalDrives, nil
}

func (r *RHEL8) PhysicalDrive(metadata *physicaldrive.Metadata) (*physicaldrive.PhysicalDrive, error) {
	blockDevices, err := r.listBlockDevices()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list block devices")
	}

	physicalDrive := &physicaldrive.PhysicalDrive{}

	for _, device := range blockDevices {
		if device.DevicePath == metadata.DevicePath {
			output, err := r.uDevADM.Run([]string{
				"info",
				"--query=all",
				"--name=" + metadata.DevicePath,
			})
			if err != nil {
				return nil, errors.Wrap(err, "failed to run udevadm physical drive command")
			}

			physicalDrive, err = r.ParseUDevADMOutput(output)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse udevadm output")
			}

			physicalDrive.DevicePath = metadata.DevicePath
			physicalDrive.Size = device.Size

			break
		}
	}

	return physicalDrive, nil
}

func (r *RHEL8) listBlockDevices() ([]BlockDevice, error) {
	output, err := r.lsblk.Run([]string{"--list", "--paths", "--bytes"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list block devices")
	}

	blockDevices, err := r.ParseLSBLKOutput(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse lsblk output")
	}

	return blockDevices, nil
}

func (r *RHEL8) ParseUDevADMOutput(output []byte) (*physicaldrive.PhysicalDrive, error) {
	lines := strings.Split(string(output), "\n")
	physicalDrive := &physicaldrive.PhysicalDrive{}

	for _, line := range lines {
		if strings.HasPrefix(line, "E: ID_MODEL=") {
			physicalDrive.Model = strings.TrimPrefix(line, "E: ID_MODEL=")
		} else if strings.HasPrefix(line, "E: ID_SERIAL_SHORT=") {
			physicalDrive.Serial = strings.TrimPrefix(line, "E: ID_SERIAL_SHORT=")
		} else if strings.HasPrefix(line, "E: ID_WWN=") {
			physicalDrive.ID = strings.TrimPrefix(line, "E: ID_WWN=")
		} else if strings.HasPrefix(line, "E: DEVNAME=") {
			physicalDrive.DevicePath = strings.TrimPrefix(line, "E: DEVNAME=")
		}
	}

	return physicalDrive, nil
}

func (r *RHEL8) ParseLSBLKOutput(output []byte) ([]BlockDevice, error) {
	lines := strings.Split(string(output), "\n")
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
				case "MAJ:MIN":
					device.MajMin = field
				case "RM":
					device.RM = field
				case "SIZE":
					size, err := strconv.ParseInt(field, 10, 64)
					if err != nil {
						return nil, errors.Wrap(err, "failed to parse size")
					}

					device.Size = uint64(size)
				case "RO":
					device.RO = field
				case "TYPE":
					device.Type = field
				case "MOUNTPOINT":
					device.Mountpoint = field
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
