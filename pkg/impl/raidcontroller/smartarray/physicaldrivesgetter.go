package smartarray

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/utils"
)

const (
	slotRegexpPattern                = `Slot (\d+)`
	physicaldriveRegexpPattern       = `physicaldrive\s+(.+)`
	physicaldriveConfigRegexpPattern = `physicaldrive\s+.+?\s+\(port\s+(.+?):box\s+(.+?):bay\s+(.+?),.*\)` //nolint: lll // This is a regexp

	keyValueParts = 2
)

var (
	slotRegexp                = regexp.MustCompile(slotRegexpPattern)
	physicaldriveRegexp       = regexp.MustCompile(physicaldriveRegexpPattern)
	physicaldriveConfigRegexp = regexp.MustCompile(physicaldriveConfigRegexpPattern)
)

// parsePhysicalDrives parses the output of the physicaldrive command and
// returns a list of PhysicalDrive entities.
func parsePhysicalDrives(output []byte) ([]*physicaldrive.PhysicalDrive, error) {
	blocks := splitOutput(physicaldriveRegexp, output)

	physicalDrives := make([]*physicaldrive.PhysicalDrive, 0, len(blocks))

	controllerID, err := parseControllerID(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse controller ID")
	}

	for _, block := range blocks {
		physicalDrive, err := parsePhysicalDrive(block)
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
	match := slotRegexp.FindSubmatch(output)

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
func parsePhysicalDrive(block []byte) (*physicaldrive.PhysicalDrive, error) {
	// Create the PhysicalDrive entity
	physicalDrive := &physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{
			CtrlMetadata: &raidcontroller.Metadata{},
			Slot:         &physicaldrive.Slot{},
		},
	}

	// Split the block into lines and parse each line
	for line := range strings.SplitSeq(string(block), "\n") {
		if err := parsePDLine(physicalDrive, line); err != nil {
			return nil, errors.Wrapf(err, "failed to parse line of physical drive: %s",
				strings.TrimSpace(line),
			)
		}
	}

	return physicalDrive, nil
}

// parseLineDetail parses a line of the show detail command and returns the key and value.
func parseLineDetail(line string) (key, value string) {
	if line == "" {
		return "", ""
	}

	splitParts := strings.Split(line, ":")

	if len(splitParts) != keyValueParts {
		return "", ""
	}

	key = strings.TrimSpace(splitParts[0])
	value = strings.TrimSpace(splitParts[1])

	return key, value
}

// parsePDLine parses a line of the physicaldrive command output
// and updates the PhysicalDrive entity.
// nolint: cyclop // The switch statement is necessary
// to parse the different key-value pairs.
func parsePDLine(physicalDrive *physicaldrive.PhysicalDrive, line string) error {
	key, value := parseLineDetail(line)

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
		// TODO check DriveType : unassigned or assigned in string
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

	case "Drive Type":
		if strings.Contains(value, "Unassigned") {
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

	case "Drive Unique ID":
		physicalDrive.ID = value

	case "Disk Name":
		physicalDrive.DevicePath = value
		// TODO miss permanent path
	}

	return nil
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
