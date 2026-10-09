//nolint:cyclop,funlen,gocognit,lll // This package contains parser functions, which are inherently complex.
package physicaldrivegetter

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

// lsblk TYPE values the getter cares about. Whole disks and partitions are both
// returned as physical drives; the IsPartition flag lets callers tell them
// apart. Every other type (raid*, lvm, loop...) is dropped in ParseLSBLKOutput.
const (
	diskDeviceType      = "disk"
	partitionDeviceType = "part"
)

type (
	RHEL8 struct {
		UDevADM  commandrunner.CommandRunner
		LSBLK    commandrunner.CommandRunner
		SmartCTL commandrunner.CommandRunner
	}

	BlockDevice struct {
		DevicePath       string
		Size             uint64
		Rotational       string
		Type             string
		Tran             string
		MountPoint       string
		PartitionType    string
		FileSystemType   string
		ParentKernelName string
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

	pkNames := make(map[string]struct{})

	physicalDrives := make([]*physicaldrive.PhysicalDrive, 0, len(blockDevices))

	for _, device := range blockDevices {
		// We need to find the parent kernel names of the block devices
		// to set the Used status correctly.
		if device.ParentKernelName != "" {
			pkNames[device.ParentKernelName] = struct{}{}
		}

		physicalDrive, err := r.PhysicalDrive(&physicaldrive.Metadata{
			ID: device.DevicePath,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive: %s", device.DevicePath)
		}

		physicalDrives = append(physicalDrives, physicalDrive)
	}

	// Set the Used status if the physical drive is a disk
	// and its parent kernel name is in the list of block devices
	// of the partitions.
	for pkName := range pkNames {
		for _, physicalDrive := range physicalDrives {
			if pkName == physicalDrive.ID {
				physicalDrive.Status = physicaldrive.PDStatusUsed
			}
		}
	}

	return physicalDrives, nil
}

//nolint:gocognit // This function is complicated by essence.
func (r *RHEL8) PhysicalDrive(
	metadata *physicaldrive.Metadata,
) (*physicaldrive.PhysicalDrive, error) {
	device, err := r.getBlockDevice(metadata.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block device: %s", metadata.ID)
	}

	physicalDrive := &physicaldrive.PhysicalDrive{}

	output, err := r.UDevADM.Run([]string{
		"info",
		"--query=all",
		"--name=" + device.DevicePath,
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
	physicalDrive.IsPartition = device.Type == partitionDeviceType

	status, reason, err := r.physicalDriveStatus(device)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get physical drive status: %s", device.DevicePath)
	}

	physicalDrive.Reason = reason
	physicalDrive.Status = status

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

func (r *RHEL8) physicalDriveStatus(device *BlockDevice) (physicaldrive.PDStatus, string, error) {
	var reason string

	// FIXME Ignore errors for now
	output, err := r.SmartCTL.Run([]string{
		"-a",
		device.DevicePath,
	})
	if err != nil {
		reason = "smartctl command failed to get physical drive status"
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

	// If err is not nil, it means smartctl failed to run
	// so we ignore this case for now.
	// Sometimes health status can be empty, because some older drives
	// doesn't support the associated SMART instruction
	if healthStatus != "PASSED" && healthStatus != "" && err == nil {
		return physicaldrive.PDStatusFailed, "", nil
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
				return physicaldrive.PDStatusUnknown, "", errors.Wrap(
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
				return physicaldrive.PDStatusUnknown, "", errors.Wrap(
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
				return physicaldrive.PDStatusUnknown, "", errors.Wrap(
					err,
					"failed to parse offline uncorrectable sector count",
				)
			}
		}
	}

	if reallocatedCount > 0 || pendingCount > 0 || uncorrectableCount > 0 {
		return physicaldrive.PDStatusFailed, "", nil
	}

	if device.MountPoint != "" || device.FileSystemType != "" || device.PartitionType != "" {
		return physicaldrive.PDStatusUsed, reason, nil
	}

	return physicaldrive.PDStatusUnassignedGood, reason, nil
}

func (r *RHEL8) getBlockDevice(devicePath string) (*BlockDevice, error) {
	output, err := r.LSBLK.Run([]string{
		devicePath,
		"--paths",
		"--bytes",
		"--nodeps",
		"--json",
		"--output",
		"name,rota,size,type,tran,mountpoint,fstype,parttype,pkname",
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

	return &blockDevices[0], nil
}

func (r *RHEL8) listBlockDevices() ([]BlockDevice, error) {
	output, err := r.LSBLK.Run([]string{
		"--list",
		"--json",
		"--paths",
		"--bytes",
		"--output",
		"name,rota,size,type,tran,mountpoint,fstype,parttype,pkname",
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
		case strings.HasPrefix(line, "E: DEVNAME="):
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

type (
	// lsblkOutput is the document printed by `lsblk --json`.
	lsblkOutput struct {
		BlockDevices []lsblkDevice `json:"blockdevices"`
	}

	// lsblkDevice is one entry of `lsblk --json --list` or `lsblk --json --nodeps`.
	lsblkDevice struct {
		Name       lsblkValue `json:"name"`
		Rota       lsblkValue `json:"rota"`
		Size       lsblkValue `json:"size"`
		Type       lsblkValue `json:"type"`
		Tran       lsblkValue `json:"tran"`
		MountPoint lsblkValue `json:"mountpoint"`
		FSType     lsblkValue `json:"fstype"`
		PartType   lsblkValue `json:"parttype"`
		PKName     lsblkValue `json:"pkname"`
	}

	// lsblkValue is a JSON scalar read as a string, whatever its JSON type.
	//
	// The JSON value types depend on the util-linux version: 2.32 (RHEL 8) prints
	// every value as a string ("rota": "1", "size": "1000"), while 2.37 (RHEL 9)
	// prints booleans and numbers ("rota": true, "size": 1000). Both print null
	// for an empty cell.
	lsblkValue string
)

// UnmarshalJSON reads a JSON string as is, null as "", and any other scalar
// (number, boolean) as its literal text.
func (v *lsblkValue) UnmarshalJSON(data []byte) error {
	switch {
	case bytes.Equal(data, []byte("null")):
		*v = ""
	case len(data) > 0 && data[0] == '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return errors.Wrap(err, "failed to unmarshal lsblk string value")
		}

		*v = lsblkValue(s)
	default:
		*v = lsblkValue(data)
	}

	return nil
}

// rotational returns the ROTA column as "0" or "1", whether lsblk printed it as
// a string or as a boolean.
func (v lsblkValue) rotational() string {
	switch v {
	case "true":
		return "1"
	case "false":
		return "0"
	default:
		return string(v)
	}
}

// ParseLSBLKOutput parses the output of `lsblk --json` run with the
// name,rota,size,type,tran,mountpoint,fstype,parttype,pkname columns.
// Devices that are neither disks nor partitions are dropped.
func ParseLSBLKOutput(output []byte) ([]BlockDevice, error) {
	if len(bytes.TrimSpace(output)) == 0 {
		return nil, nil
	}

	var parsed lsblkOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal lsblk json output")
	}

	devices := []BlockDevice{}

	for _, d := range parsed.BlockDevices {
		// Skip non-disk and non-part devices
		if d.Type != diskDeviceType && d.Type != partitionDeviceType {
			continue
		}

		size, err := strconv.ParseUint(string(d.Size), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse size of %s", d.Name)
		}

		devices = append(devices, BlockDevice{
			DevicePath:       string(d.Name),
			Size:             size,
			Rotational:       d.Rota.rotational(),
			Type:             string(d.Type),
			Tran:             string(d.Tran),
			MountPoint:       string(d.MountPoint),
			PartitionType:    string(d.PartType),
			FileSystemType:   string(d.FSType),
			ParentKernelName: string(d.PKName),
		})
	}

	return devices, nil
}
