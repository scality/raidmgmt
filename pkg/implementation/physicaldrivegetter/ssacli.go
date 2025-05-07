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
	"github.com/scality/raidmgmt/pkg/utils"
)

const (
	ssacliSlotRegexpPattern          = `Slot (\d+)`
	ssacliPhysicalDriveRegexpPattern = `physicaldrive\s+(.+)`
)

type SSACLI struct {
	SSACLI commandrunner.CommandRunner
	LSBLK  commandrunner.CommandRunner
}

var (
	_ ports.PhysicalDrivesGetter = &SSACLI{}

	ssacliSlotRegexp          = regexp.MustCompile(ssacliSlotRegexpPattern)
	ssacliPhysicalDriveRegexp = regexp.MustCompile(ssacliPhysicalDriveRegexpPattern)
)

// NewSSACLI creates a new SSACLI instance.
func NewSSACLI(
	ssacli *commandrunner.SSACLI,
	lsblk *commandrunner.LSBLK,
) *SSACLI {
	return &SSACLI{
		SSACLI: ssacli,
		LSBLK:  lsblk,
	}
}

// PhysicalDrives returns all physical drives for a given controller.
func (s *SSACLI) PhysicalDrives(metadata *raidcontroller.Metadata) (
	[]*physicaldrive.PhysicalDrive,
	error,
) {
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.ID),
		"physicaldrive",
		"all",
		"show",
		"detail",
	}

	output, err := s.SSACLI.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to show all physical drives details")
	}

	physicalDrives, err := s.parsePhysicalDrives(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse physical drives details")
	}

	return physicalDrives, nil
}

// PhysicalDrive returns a physical drive for a given metadata.
func (s *SSACLI) PhysicalDrive(metadata *physicaldrive.Metadata) (
	*physicaldrive.PhysicalDrive,
	error,
) {
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.CtrlMetadata.ID),
		"physicaldrive",
		metadata.ID,
		"show",
		"detail",
	}

	output, err := s.SSACLI.Run(args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show details for physical drive %s", metadata.ID)
	}

	controllerID, err := parseControllerID(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse controller ID")
	}

	physicalDrive, err := s.parsePhysicalDrive(output)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse physical drive %s", metadata.ID)
	}

	physicalDrive.CtrlMetadata.ID = controllerID

	return physicalDrive, nil
}

// parsePhysicalDrives parses the output of the physicaldrive command and
// returns a list of PhysicalDrive entities.
func (s *SSACLI) parsePhysicalDrives(output []byte) ([]*physicaldrive.PhysicalDrive, error) {
	blocks := utils.SplitOutput(ssacliPhysicalDriveRegexp, output)

	physicalDrives := make([]*physicaldrive.PhysicalDrive, 0, len(blocks))

	controllerID, err := parseControllerID(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse controller ID")
	}

	for _, block := range blocks {
		physicalDrive, err := s.parsePhysicalDrive(block)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse physical drive: %s", block)
		}

		physicalDrive.CtrlMetadata.ID = controllerID

		physicalDrives = append(physicalDrives, physicalDrive)
	}

	return physicalDrives, nil
}

// parseControllerID parses the controller ID from the output of the physicaldrive command.
func parseControllerID(output []byte) (int, error) {
	match := ssacliSlotRegexp.FindSubmatch(output)

	if match == nil {
		return 0, errors.New("controller ID not found")
	}

	controllerID, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0, errors.Wrap(err, "failed to convert controller ID to integer")
	}

	return controllerID, nil
}

// parsePhysicalDrive parses a physical drive block and returns a PhysicalDrive entity.
func (s *SSACLI) parsePhysicalDrive(block []byte) (*physicaldrive.PhysicalDrive, error) {
	// Create the PhysicalDrive entity
	physicalDrive := &physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{
			CtrlMetadata: &raidcontroller.Metadata{},
		},
		Slot: &physicaldrive.Slot{},
	}

	// Split the block into lines and parse each line
	for line := range strings.SplitSeq(string(block), "\n") {
		if err := s.parsePDLine(physicalDrive, line); err != nil {
			return nil, errors.Wrapf(err, "failed to parse line of physical drive: %s",
				strings.TrimSpace(line),
			)
		}
	}

	physicalDrive.ID = physicalDrive.Slot.Format()

	return physicalDrive, nil
}

// parsePDLine parses a line of the physicaldrive command output
// and updates the PhysicalDrive entity.
// nolint: cyclop,gocognit // The switch statement is necessary
// to parse the different key-value pairs.
func (s *SSACLI) parsePDLine( //nolint:funlen // This function is long and not compressible
	physicalDrive *physicaldrive.PhysicalDrive,
	line string,
) error {
	key, value := utils.ParseLineDetail(line)

	// Parse the key-value pair
	switch key {
	case "Port", "Box", "Bay":
		parseSlotInfo(physicalDrive, key, value)

	case "Model":
		splitLine := strings.Fields(value)
		physicalDrive.Vendor = splitLine[0]
		physicalDrive.Model = splitLine[1]

	case "Serial Number":
		physicalDrive.Serial = value

	case "Size":
		size, err := utils.ConvertSizeBytes(value)
		if err != nil {
			return errors.Wrap(err, "failed to convert size to bytes")
		}

		physicalDrive.Size = size

	case "Status":
		if physicalDrive.Status == physicaldrive.PDStatusUnknown {
			mapStatus := map[string]physicaldrive.PDStatus{
				"OK":      physicaldrive.PDStatusUsed,
				"Failed":  physicaldrive.PDStatusFailed,
				"Offline": physicaldrive.PDStatusFailed,
			}

			status, ok := mapStatus[value]
			if !ok {
				return errors.Errorf("invalid status: %s", value)
			}

			physicalDrive.Status = status
		}

	case "Drive Type":
		if physicalDrive.Status != physicaldrive.PDStatusUsed &&
			strings.Contains(value, "Unassigned") {
			physicalDrive.Status = physicaldrive.PDStatusUnassignedGood
		}

	case "Interface Type":
		mapInterfaceType := map[string]physicaldrive.DiskType{
			"SATA":            physicaldrive.DiskTypeHDD,
			"SAS":             physicaldrive.DiskTypeHDD,
			"Solid State SAS": physicaldrive.DiskTypeSSD,
		}

		interfaceType, ok := mapInterfaceType[value]
		if !ok {
			return errors.Errorf("invalid interface type: %s", value)
		}

		physicalDrive.Type = interfaceType

	case "Disk Name":
		physicalDrive.DevicePath = value

		blockDevice, err := s.getBlockDevice(value)
		if err != nil {
			return errors.Wrapf(err, "failed to get block device for %s", value)
		}

		if isBlockDeviceUsed(blockDevice) {
			physicalDrive.Status = physicaldrive.PDStatusUsed
		}
		// TODO miss permanent path
	}

	return nil
}

func (s *SSACLI) getBlockDevice(devicePath string) (*BlockDevice, error) {
	output, err := s.LSBLK.Run([]string{
		devicePath,
		"--paths",
		"--bytes",
		"--nodeps",
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

// parseSlotInfo parses the slot information and updates the PhysicalDrive entity.
func parseSlotInfo(pd *physicaldrive.PhysicalDrive, key, value string) {
	switch key {
	case "Port":
		pd.Slot.Port = value
	case "Box":
		pd.Slot.Enclosure = value
	case "Bay":
		pd.Slot.Bay = value
	}
}

// isBlockDeviceUsed checks if a block device is used.
// If the device is mounted or has a filesystem type, it is considered used.
// Otherwise, it is considered unassigned good.
func isBlockDeviceUsed(device *BlockDevice) bool {
	if device.MountPoint != "" || device.FileSystemType != "" || device.PartitionType != "" {
		return true
	}

	return false
}
