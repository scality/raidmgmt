//nolint:cyclop // Juuuuust above complexity limit
package logicalvolumemanager

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/utils"
)

const (
	ssacliArrayOrUnassignedRegexpPattern = `(Array\s+[A-Z]+\s+\(.*\)|Unassigned)`
	ssacliArrayIDRegexpPattern           = `Array\s+(\w+)`

	ssacliMinMatches = 2
)

type SSACLI struct {
	ports.PhysicalDrivesGetter
	ports.LogicalVolumesGetter
	SSACLI commandrunner.CommandRunner
}

var (
	_ ports.LogicalVolumesManager = &SSACLI{}

	ssacliArrayOrUnassignedRegexp = regexp.MustCompile(ssacliArrayOrUnassignedRegexpPattern)
	ssacliArrayIDRegexp           = regexp.MustCompile(ssacliArrayIDRegexpPattern)
)

func NewSSACLI(
	ssacli *commandrunner.SSACLI,
	physicalDrivesGetter ports.PhysicalDrivesGetter,
	logicalVolumesGetter ports.LogicalVolumesGetter,
) *SSACLI {
	return &SSACLI{
		SSACLI:               ssacli,
		PhysicalDrivesGetter: physicalDrivesGetter,
		LogicalVolumesGetter: logicalVolumesGetter,
	}
}

// CreateLV creates a logical volume from a request.
//
//nolint:funlen // This function is long.
func (s *SSACLI) CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	physicalDrivesToUse := make([]*physicaldrive.PhysicalDrive, 0, len(request.PDrivesMetadata))

	for _, pdMetadata := range request.PDrivesMetadata {
		pd, err := s.PhysicalDrive(pdMetadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive %s",
				pdMetadata.ID)
		}

		physicalDrivesToUse = append(physicalDrivesToUse, pd)
	}

	// Validate the RAID creation
	err := logicalvolume.ValidateRAIDCreation(physicalDrivesToUse, request.RAIDLevel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate RAID creation")
	}

	// Format the physical drives
	drives := formatDrives(request.PDrivesMetadata)

	raidLevel, ok := ssacliRAIDLevelToken(request.RAIDLevel)
	if !ok {
		return nil, errors.Errorf("ssacli cannot express RAID level %s", request.RAIDLevel)
	}

	// Create the logical volume
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(request.CtrlMetadata.ID),
		"create",
		"type=ld",
		"drives=" + drives,
		"raid=" + raidLevel,
		"forced", // To bypass the warning and confirmation prompt
	}

	_, err = s.SSACLI.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run create logical drive command")
	}

	// Find the new logical drive using the controller config
	// Get the controller config to get the physical drives metadata and RAID level
	args = []string{
		"controller",
		"slot=" + strconv.Itoa(request.CtrlMetadata.ID),
		"show",
		"config",
	}

	output, err := s.SSACLI.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to show controller config")
	}

	newLogicalDrive, err := s.findNewLogicalDrive(request, output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find the new logical drive")
	}

	return newLogicalDrive, nil
}

// DeleteLV deletes a logical volume.
func (s *SSACLI) DeleteLV(metadata *logicalvolume.Metadata) error {
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.CtrlMetadata.ID),
		"logicaldrive",
		metadata.ID,
		"delete",
		"forced", // To bypass the warning message
	}

	_, err := s.SSACLI.Run(args)
	if err != nil {
		return errors.Wrapf(err, "failed to delete logical drive %s", metadata.ID)
	}

	return nil
}

// AddPDsToLV adds a physical drive to a logical volume.
func (s *SSACLI) AddPDsToLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	arrayID, err := s.getArrayID(lvMetadata)
	if err != nil {
		return errors.Wrapf(err, "failed to get array ID for logical drive %s", lvMetadata.ID)
	}

	err = s.migrateArray(arrayID, lvMetadata, pdsMetadata, "add")
	if err != nil {
		return errors.Wrapf(err, "failed to expand array %s (logical drive %s) with physical drives",
			arrayID, lvMetadata.ID)
	}

	return nil
}

// DeletePDsFromLV deletes a physical drive from a logical volume.
func (s *SSACLI) DeletePDsFromLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	arrayID, err := s.getArrayID(lvMetadata)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to get array ID for logical drive %s",
			lvMetadata.ID,
		)
	}

	err = s.migrateArray(arrayID, lvMetadata, pdsMetadata, "remove")
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to shrink array %s (logical drive %s) with physical drives",
			arrayID, lvMetadata.ID,
		)
	}

	return nil
}

// findNewLogicalDrive finds the new logical drive created by the controller.
// It returns the new logical drive and an error if any.
func (s *SSACLI) findNewLogicalDrive(
	request *logicalvolume.Request,
	output []byte,
) (
	*logicalvolume.LogicalVolume, error,
) {
	id, err := getLogicalDriveID(request, output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find logical drive ID")
	}

	// Get the logical drive details
	metadata := &logicalvolume.Metadata{
		CtrlMetadata: request.CtrlMetadata,
		ID:           id,
	}

	newLV, err := s.LogicalVolume(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get new logical drive %s", id)
	}

	return newLV, nil
}

// getArrayID gets the array ID of the logical volume.
func (s *SSACLI) getArrayID(metadata *logicalvolume.Metadata) (string, error) {
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
		return "", errors.Wrapf(err, "failed to show details for logical drive %s", metadata.ID)
	}

	matches := ssacliArrayIDRegexp.FindStringSubmatch(string(output))
	if len(matches) < ssacliMinMatches {
		return "", errors.New("failed to parse array ID")
	}

	return matches[1], nil
}

// migrateArray migrates the physical drives to the logical volume.
//
// action can be "add" or "remove".
func (s *SSACLI) migrateArray(
	arrayID string,
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata []*physicaldrive.Metadata,
	action string,
) error {
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(lvMetadata.CtrlMetadata.ID),
		"array",
		arrayID,
		action,
		"drives=" + formatDrives(pdsMetadata),
		"forced", // To bypass the warning
	}

	_, err := s.SSACLI.Run(args)
	if err != nil {
		return errors.Wrapf(err, "failed to %s drives to array %s", action, arrayID)
	}

	return nil
}

// ssacliRAIDLevelToken maps a RAID level to its ssacli "raid=" token, which
// spells RAID 10 as 1+0. A level ssacli cannot express yields ok=false so the
// caller fails closed rather than passing a token ssacli cannot parse.
func ssacliRAIDLevelToken(level logicalvolume.RAIDLevel) (string, bool) {
	switch level { //nolint:exhaustive // unmappable levels handled by the default
	case logicalvolume.RAIDLevel0:
		return "0", true
	case logicalvolume.RAIDLevel1:
		return "1", true
	case logicalvolume.RAIDLevel10:
		return "1+0", true
	default:
		return "", false
	}
}

// formatDrives formats the physical drives to a string.
// It returns a string with the physical drives formatted as "slot1,slot2,slot3".
func formatDrives(pdsMetadata []*physicaldrive.Metadata) string {
	var formattedDrives string

	if len(pdsMetadata) == 0 {
		return ""
	}

	formattedDrives = pdsMetadata[0].ID

	for _, drive := range pdsMetadata[1:] {
		formattedDrives += "," + drive.ID
	}

	return formattedDrives
}

// getLogicalDriveID finds the logical drive holding the first physical drive of
// the request, from a "controller show config" output. That output lists an
// array as a block opening with its logical drives and closing with its member
// drives, so the drive belongs to the logical drive declared in its own block.
// A drive still sitting in the Unassigned block belongs to none, and an array
// declaring several logical drives cannot be told apart by membership alone.
func getLogicalDriveID(
	request *logicalvolume.Request,
	output []byte,
) (string, error) {
	driveID := request.PDrivesMetadata[0].ID

	var holders []string

	for _, block := range utils.SplitOutput(ssacliArrayOrUnassignedRegexp, output) {
		logicalDriveIDs, holdsDrive := parseArrayBlock(block, driveID)
		if holdsDrive {
			holders = append(holders, logicalDriveIDs...)
		}
	}

	switch len(holders) {
	case 0:
		return "", errors.Errorf("physical drive %s not found in any logical drive", driveID)
	case 1:
		return holders[0], nil
	default:
		return "", errors.Errorf("physical drive %s found in multiple logical drives: %s",
			driveID, strings.Join(holders, ", "))
	}
}

// parseArrayBlock reads one block of a "controller show config" output and
// returns the logical drives it declares, along with whether the given physical
// drive is one of its members.
func parseArrayBlock(block []byte, driveID string) ([]string, bool) {
	var (
		logicalDriveIDs []string
		holdsDrive      bool
	)

	for line := range strings.SplitSeq(string(block), "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "logicaldrive"):
			if parts := strings.Fields(trimmed); len(parts) > 1 {
				logicalDriveIDs = append(logicalDriveIDs, parts[1])
			}
		case strings.HasPrefix(trimmed, "physicaldrive"):
			// The address is the second field, compared whole: a substring
			// match would read bay 11 as bay 1.
			if parts := strings.Fields(trimmed); len(parts) > 1 && parts[1] == driveID {
				holdsDrive = true
			}
		}
	}

	return logicalDriveIDs, holdsDrive
}
