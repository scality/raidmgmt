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
	commandrunner.CommandRunner
}

var (
	_ ports.LogicalVolumesManager = &SSACLI{}

	ssacliArrayOrUnassignedRegexp = regexp.MustCompile(ssacliArrayOrUnassignedRegexpPattern)
	ssacliArrayIDRegexp           = regexp.MustCompile(ssacliArrayIDRegexpPattern)
)

func NewSSACLI(
	commandRunner commandrunner.CommandRunner,
	physicalDrivesGetter ports.PhysicalDrivesGetter,
	logicalVolumesGetter ports.LogicalVolumesGetter,
) *SSACLI {
	return &SSACLI{
		CommandRunner:        commandRunner,
		PhysicalDrivesGetter: physicalDrivesGetter,
		LogicalVolumesGetter: logicalVolumesGetter,
	}
}

// CreateLV creates a logical volume from a request.
func (s *SSACLI) CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	physicalDrivesToUse := make([]*physicaldrive.PhysicalDrive, 0, len(request.PDrivesMetadata))

	for _, pdMetadata := range request.PDrivesMetadata {
		pd, err := s.PhysicalDrive(pdMetadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive %s",
				utils.FormatSlot(pdMetadata.Slot))
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

	// Convert the RAID level to SSA CLI format
	raidLevel := string(request.RAIDLevel)
	if request.RAIDLevel == logicalvolume.RAIDLevel10 {
		raidLevel = "1+0"
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

	_, err = s.CommandRunner.Run(args)
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

	output, err := s.CommandRunner.Run(args)
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

	_, err := s.CommandRunner.Run(args)
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

	output, err := s.CommandRunner.Run(args)
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

	_, err := s.CommandRunner.Run(args)
	if err != nil {
		return errors.Wrapf(err, "failed to %s drives to array %s", action, arrayID)
	}

	return nil
}

// formatDrives formats the physical drives to a string.
// It returns a string with the physical drives formatted as "slot1,slot2,slot3".
func formatDrives(pdsMetadata []*physicaldrive.Metadata) string {
	var formattedDrives string

	if len(pdsMetadata) == 0 {
		return ""
	}

	formattedDrives = utils.FormatSlot(pdsMetadata[0].Slot)

	for _, drive := range pdsMetadata[1:] {
		formattedDrives += "," + utils.FormatSlot(drive.Slot)
	}

	return formattedDrives
}

// getLogicalDriveID finds the logical drive ID that contains one of the physical drives.
// It returns the logical drive ID and an error if any.
// nolint: gocognit // This function is not too complex.
func getLogicalDriveID(
	request *logicalvolume.Request,
	output []byte,
) (string, error) {
	blocks := utils.SplitOutput(ssacliArrayOrUnassignedRegexp, output)

	var logicalDriveID string

	for _, block := range blocks {
		logicalDriveID = ""

		for line := range strings.SplitSeq(string(block), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "logicaldrive") {
				// Extract the logical drive ID
				parts := strings.Fields(line)
				if len(parts) > 1 {
					logicalDriveID = parts[1]
				}
			} else if strings.Contains(line, utils.FormatSlot(request.PDrivesMetadata[0].Slot)) {
				// Check if line contains the physical drive slot
				// If the logical drive ID is empty, return it
				// If the logical drive ID is not empty, return an error
				if logicalDriveID == "" {
					// Found the physical drive in the logical drive, return the logical drive ID
					return logicalDriveID, nil
				}

				return "", errors.Errorf(
					"physical drive %s found in multiple logical drives",
					utils.FormatSlot(request.PDrivesMetadata[0].Slot),
				)
			}
		}
	}

	return "", errors.Errorf(
		"physical drive %s not found in any logical drive",
		utils.FormatSlot(request.PDrivesMetadata[0].Slot),
	)
}
