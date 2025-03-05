package smartarray

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

const (
	logicalVolumeRegexpPattern           = `\s*Logical Drive:\s+\d+`
	logicalVolumeIDStatusRegexpPattern   = `logicaldrive\s+(\d+)`
	raidLevelRegexpPattern               = `RAID\s+(\d+)`
	associatedPhysicalDriveRegexpPattern = `logicaldrive\s+\d+\s+\(.*?\)\n(\s*physicaldrive.*?\n)*`

	minMatches = 2
)

var (
	logicalVolumeRegexp           = regexp.MustCompile(logicalVolumeRegexpPattern)
	logicalVolumeIDStatusRegexp   = regexp.MustCompile(logicalVolumeIDStatusRegexpPattern)
	raidLevelRegexp               = regexp.MustCompile(raidLevelRegexpPattern)
	associatedPhysicalDriveRegexp = regexp.MustCompile(associatedPhysicalDriveRegexpPattern)
)

func parseLogicalVolumes(output []byte) (
	[]*logicalvolume.LogicalVolume,
	error,
) {
	blocks := splitOutput(logicalVolumeRegexp, output)

	logicalVolumes := make([]*logicalvolume.LogicalVolume, 0, len(blocks))

	for _, block := range blocks {
		logicalVolume, err := parseLogicalVolume(block)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse logical volume: %s", block)
		}

		logicalVolumes = append(logicalVolumes, logicalVolume)
	}

	return logicalVolumes, nil
}

func parseLogicalVolume(block []byte) (
	*logicalvolume.LogicalVolume,
	error,
) {
	// Create the LogicalVolume entity
	logicalVolume := &logicalvolume.LogicalVolume{
		Metadata: &logicalvolume.Metadata{
			CtrlMetadata: &raidcontroller.Metadata{},
			ID:           "",
		},
	}

	for _, line := range strings.Split(string(block), "\n") {
		if err := parseLVLine(logicalVolume, line); err != nil {
			return nil, errors.Wrap(err, "failed to parse line")
		}
	}

	return logicalVolume, nil
}

// parseLVLine parses a line of the logical volume output and updates the logical volume entity.
func parseLVLine(logicalVolume *logicalvolume.LogicalVolume, line string) error {
	key, value := parseLineDetail(line)

	switch key {
	case "Logical Drive":
		logicalVolume.ID = value
	case "Status":
		mapStatus := map[string]logicalvolume.LVStatus{
			"OK":     logicalvolume.LVStatusOptimal,
			"Failed": logicalvolume.LVStatusFailed,
			// TODO check real values
		}

		status, ok := mapStatus[value]
		if !ok {
			return errors.Errorf("invalid status: %s", value)
		}

		logicalVolume.Status = status
	case "Disk Name":
		logicalVolume.DevicePath = value
		// TODO miss permanent path
	}

	return nil
}

// extractInfoFromConfig extracts the RAID level and the physical drives metadata
// from the config show output
// it is necessary to keep it as is.
//
//nolint:gocognit // This function may seem complex but due to the "continue" statements
func extractInfoFromConfig(
	logicalVolume *logicalvolume.LogicalVolume,
	output []byte,
) (logicalvolume.RAIDLevel, []*physicaldrive.Metadata) {
	blocks := splitOutput(associatedPhysicalDriveRegexp, output)

	// Get the physical drives metadata
	pDrivesMetadata := make([]*physicaldrive.Metadata, 0, len(blocks))
	raidLevel := logicalVolume.RAIDLevel

	for _, block := range blocks {
		// Parse the block only if the id of logical volume match
		idMatch := logicalVolumeIDStatusRegexp.FindStringSubmatch(string(block))
		if len(idMatch) < minMatches || idMatch[1] != logicalVolume.ID {
			continue
		}

		for _, line := range strings.Split(string(block), "\n") {
			// Get the RAID level
			if raidLevel == "" {
				raidLevel = extractRAIDLevel(line)
			}

			matches := physicaldriveConfigRegexp.FindStringSubmatch(line)
			if len(matches) < minMatches {
				continue
			}

			// Create the PhysicalDrive metadata
			pDriveMetadata := &physicaldrive.Metadata{
				CtrlMetadata: logicalVolume.CtrlMetadata,
				Slot: &physicaldrive.Slot{
					Port:      matches[2],
					Enclosure: matches[3],
					Bay:       matches[4],
				},
			}

			pDrivesMetadata = append(pDrivesMetadata, pDriveMetadata)
		}
	}

	return raidLevel, pDrivesMetadata
}

func extractRAIDLevel(line string) logicalvolume.RAIDLevel {
	var raidLevel logicalvolume.RAIDLevel

	raidMatch := raidLevelRegexp.FindStringSubmatch(line)

	if len(raidMatch) > 1 {
		raidLevel = logicalvolume.RAIDLevelMap(raidMatch[1])
	}

	return raidLevel
}
