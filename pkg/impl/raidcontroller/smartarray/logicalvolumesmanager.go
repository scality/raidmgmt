package smartarray

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
)

const (
	arrayIDPattern = `Array\s+(\w+)`
)

var arrayIDRegexp = regexp.MustCompile(arrayIDPattern)

// formatDrives formats the physical drives to a string.
// It returns a string with the physical drives formatted as "slot1,slot2,slot3".
func formatDrives(metadatas []*physicaldrive.Metadata) string {
	var formattedDrives string

	if len(metadatas) == 0 {
		return formattedDrives
	}

	if len(metadatas) == 1 {
		return formatSlot(metadatas[0].Slot)
	}

	for _, drive := range metadatas {
		formattedDrives += formatSlot(drive.Slot) + ","
	}

	return formattedDrives
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

// getLogicalDriveID finds the logical drive ID that contains one of the physical drives.
// It returns the logical drive ID and an error if any.
// nolint: gocognit // This function is not too complex.
func getLogicalDriveID(
	request *logicalvolume.Request,
	output []byte,
) (string, error) {
	blocks := splitOutput(associatedPhysicalDriveRegexp, output)

	var logicalDriveID string

	for _, block := range blocks {
		for _, line := range strings.Split(string(block), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "logicaldrive") {
				// Extract the logical drive ID
				parts := strings.Fields(line)
				if len(parts) > 1 {
					logicalDriveID = parts[1]
				}
			} else if strings.Contains(line, formatSlot(request.PDrivesMetadata[0].Slot)) {
				// Found the physical drive in the logical drive, return the logical drive ID
				return logicalDriveID, nil
			}
		}
	}

	return "", errors.Errorf(
		"physical drive %s not found in any logical drive",
		formatSlot(request.PDrivesMetadata[0].Slot),
	)
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

	output, err := s.Run(args)
	if err != nil {
		return "", errors.Wrapf(err, "failed to show details for logical drive %s", metadata.ID)
	}

	matches := arrayIDRegexp.FindStringSubmatch(string(output))
	if len(matches) < minMatches {
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
	pdMetadatas []*physicaldrive.Metadata,
	action string,
) error {
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(lvMetadata.CtrlMetadata.ID),
		"array",
		arrayID,
		action,
		"drives=" + formatDrives(pdMetadatas),
		"forced", // To bypass the warning
	}

	_, err := s.Run(args)
	if err != nil {
		return errors.Wrapf(err, "failed to %s drives to array %s", action, arrayID)
	}

	return nil
}
