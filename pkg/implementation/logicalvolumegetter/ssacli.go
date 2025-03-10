//nolint:cyclop // Right above the accepted ratio
package logicalvolumegetter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/utils"
)

const (
	ssacliLogicalVolumeRegexpPattern         = `\s*Logical Drive:\s+\d+`
	ssacliLogicalVolumeIDStatusRegexpPattern = `logicaldrive\s+(\d+)`
	ssacliRAIDLevelRegexpPattern             = `RAID\s+(\d+)`
	ssacliArrayOrUnassignedRegexpPattern     = `(Array\s+[A-Z]+\s+\(.*\)|Unassigned)`

	ssacliPhysicalDriveConfigRegexpPattern = `physicaldrive\s+.+?\s+\(port\s+(.+?):box\s+(.+?):bay\s+(.+?),.*\)` // nolint: lll // This is a regexp

	ssacliMinStringMatches = 2
)

type SSACLI struct {
	SSACLI commandrunner.CommandRunner
	LSBLK  commandrunner.CommandRunner
}

var (
	_ ports.LogicalVolumesGetter = &SSACLI{}

	ssacliLogicalVolumeRegexp         = regexp.MustCompile(ssacliLogicalVolumeRegexpPattern)
	ssacliLogicalVolumeIDStatusRegexp = regexp.MustCompile(ssacliLogicalVolumeIDStatusRegexpPattern)
	ssacliRAIDLevelRegexp             = regexp.MustCompile(ssacliRAIDLevelRegexpPattern)
	ssacliArrayOrUnassignedRegexp     = regexp.MustCompile(ssacliArrayOrUnassignedRegexpPattern)
	ssacliPhysicalDriveConfigRegexp   = regexp.MustCompile(ssacliPhysicalDriveConfigRegexpPattern)
)

func NewSSACLI(
	ssacli *commandrunner.SSACLI,
	lsblk *commandrunner.LSBLK,
) *SSACLI {
	return &SSACLI{
		SSACLI: ssacli,
		LSBLK:  lsblk,
	}
}

// LogicalVolumes returns all logical volumes for a given controller.
func (s *SSACLI) LogicalVolumes(metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume,
	error,
) {
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.ID),
		"logicaldrive",
		"all",
		"show",
		"detail",
	}

	output, err := s.SSACLI.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to show all logical drives details")
	}

	logicalVolumes, err := parseLogicalVolumes(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse logical drives details")
	}

	// Get the controller config to get the physical drives metadata and RAID level
	args = []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.ID),
		"show",
		"config",
	}

	output, err = s.SSACLI.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to show controller config")
	}

	for _, lv := range logicalVolumes {
		// Set the controller metadata
		lv.CtrlMetadata = metadata

		// Extract the RAID level and physical drives metadata
		raidLevel, pdsMetadata := extractInfoFromConfig(lv, output)
		lv.RAIDLevel = raidLevel
		lv.PDrivesMetadata = pdsMetadata
	}

	return logicalVolumes, nil
}

// LogicalVolume returns a logical volume for a given metadata.
func (s *SSACLI) LogicalVolume(metadata *logicalvolume.Metadata) (
	*logicalvolume.LogicalVolume,
	error,
) {
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.CtrlMetadata.ID),
		"logicaldrive",
		metadata.ID,
		"show",
		"detail",
	}

	output, err := s.SSACLI.Run(args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show details for logical drive %s", metadata.ID)
	}

	logicalVolume, err := parseLogicalVolume(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse logical drive")
	}

	logicalVolume.Metadata = metadata

	// Get the controller config to get the physical drives metadata and RAID level
	args = []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.CtrlMetadata.ID),
		"show",
		"config",
	}

	output, err = s.SSACLI.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to show controller config")
	}

	// Extract the RAID level and physical drives metadata
	raidLevel, pdsMetadata := extractInfoFromConfig(logicalVolume, output)
	logicalVolume.RAIDLevel = raidLevel
	logicalVolume.PDrivesMetadata = pdsMetadata

	return logicalVolume, nil
}

func parseLogicalVolumes(output []byte) (
	[]*logicalvolume.LogicalVolume,
	error,
) {
	blocks := utils.SplitOutput(ssacliLogicalVolumeRegexp, output)

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

	for line := range strings.SplitSeq(string(block), "\n") {
		if err := parseLVLine(logicalVolume, line); err != nil {
			return nil, errors.Wrap(err, "failed to parse line")
		}
	}

	return logicalVolume, nil
}

// parseLVLine parses a line of the logical volume output and updates the logical volume entity.
func parseLVLine(logicalVolume *logicalvolume.LogicalVolume, line string) error {
	key, value := utils.ParseLineDetail(line)

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
	blocks := utils.SplitOutput(ssacliArrayOrUnassignedRegexp, output)

	// Get the physical drives metadata
	pDrivesMetadata := make([]*physicaldrive.Metadata, 0, len(blocks))
	raidLevel := logicalVolume.RAIDLevel

	for _, block := range blocks {
		// Parse the block only if the id of logical volume match
		idMatch := ssacliLogicalVolumeIDStatusRegexp.FindStringSubmatch(string(block))
		if len(idMatch) < ssacliMinStringMatches || idMatch[1] != logicalVolume.ID {
			continue
		}

		for line := range strings.SplitSeq(string(block), "\n") {
			// Get the RAID level
			if raidLevel.String() == "Unknown" {
				raidLevel = extractRAIDLevel(line)
			}

			matches := ssacliPhysicalDriveConfigRegexp.FindStringSubmatch(line)
			//nolint:mnd // The matches required here are 4
			if len(matches) < 4 {
				continue
			}

			// Create the PhysicalDrive metadata
			pDriveMetadata := &physicaldrive.Metadata{
				CtrlMetadata: logicalVolume.CtrlMetadata,
				Slot: &physicaldrive.Slot{
					Port:      matches[1],
					Enclosure: matches[2],
					Bay:       matches[3],
				},
			}

			pDrivesMetadata = append(pDrivesMetadata, pDriveMetadata)
		}
	}

	return raidLevel, pDrivesMetadata
}

func extractRAIDLevel(line string) logicalvolume.RAIDLevel {
	var raidLevel logicalvolume.RAIDLevel

	raidMatch := ssacliRAIDLevelRegexp.FindStringSubmatch(line)

	if len(raidMatch) > 1 {
		raidLevel = logicalvolume.RAIDLevelMap(raidMatch[1])
	}

	return raidLevel
}
