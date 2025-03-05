package smartarray

import (
	"bytes"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"github.com/scality/raidmgmt/commandrunner"
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/domain/ports"
)

type SSACLI struct {
	commandrunner.CommandRunner
}

type SSACLIRaidController interface {
	ports.ControllersGetter
	ports.PhysicalDrivesGetter
	ports.LogicalVolumesGetter
	ports.LogicalVolumesManager
}

var (
	_ SSACLIRaidController = &SSACLI{}

	// formatSlot formats a physical drive slot for SSA CLI.
	// The format is "port:enclosure:bay".
	//
	//nolint:gochecknoglobals // This is necessary since SSA CLI requires this format.
	formatSlot = func(slot *physicaldrive.Slot) string {
		if slot.Port == "" {
			return slot.Enclosure + ":" + slot.Bay
		}

		return slot.Port + ":" + slot.Enclosure + ":" + slot.Bay
	}
)

func NewSSACLI(
	runner commandrunner.CommandRunner,
) *SSACLI {
	return &SSACLI{
		CommandRunner: runner,
	}
}

// Controllers returns a list of RAID controllers.
func (s *SSACLI) Controllers() ([]*raidcontroller.RAIDController, error) {
	output, err := s.Run([]string{
		"controller",
		"all",
		"show",
		"detail",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to show all controllers details")
	}

	controllers, err := parseControllers(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse controllers details")
	}

	return controllers, nil
}

// Controller returns a RAID controller for a given metadata.
func (s *SSACLI) Controller(metadata *raidcontroller.Metadata) (
	*raidcontroller.RAIDController,
	error,
) {
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.ID),
		"show",
		"detail",
	}

	output, err := s.Run(args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show details for controller %d", metadata.ID)
	}

	controller, err := parseController(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse controller")
	}

	return controller, nil
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

	output, err := s.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to show all physical drives details")
	}

	physicalDrives, err := parsePhysicalDrives(output)
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
	slot := formatSlot(metadata.Slot)

	args := []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.CtrlMetadata.ID),
		"physicaldrive",
		slot,
		"show",
		"detail",
	}

	output, err := s.Run(args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show details for physical drive %s", slot)
	}

	controllerID, err := parseControllerID(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse controller ID")
	}

	physicalDrive, err := parsePhysicalDrive(output)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse physical drive %s", slot)
	}

	physicalDrive.CtrlMetadata.ID = controllerID

	return physicalDrive, nil
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

	output, err := s.Run(args)
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

	output, err = s.Run(args)
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

	output, err := s.Run(args)
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

	output, err = s.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to show controller config")
	}

	// Extract the RAID level and physical drives metadata
	raidLevel, pdsMetadata := extractInfoFromConfig(logicalVolume, output)
	logicalVolume.RAIDLevel = raidLevel
	logicalVolume.PDrivesMetadata = pdsMetadata

	return logicalVolume, nil
}

// CreateLV creates a logical volume from a request.
func (s *SSACLI) CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	physicalDrivesToUse := make([]*physicaldrive.PhysicalDrive, 0, len(request.PDrivesMetadata))

	for _, pdMetadata := range request.PDrivesMetadata {
		pd, err := s.PhysicalDrive(pdMetadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive %s",
				formatSlot(pdMetadata.Slot))
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

	_, err = s.Run(args)
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

	output, err := s.Run(args)
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

	_, err := s.Run(args)
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

// StartBlink starts blinking a physical drive.
func (s *SSACLI) StartBlink(metadata *physicaldrive.Metadata) error {
	err := s.blink(metadata, "on")
	if err != nil {
		return errors.Wrap(err, "failed to start blinking physical drive")
	}

	return nil
}

// StopBlink stops blinking a physical drive.
func (s *SSACLI) StopBlink(metadata *physicaldrive.Metadata) error {
	err := s.blink(metadata, "off")
	if err != nil {
		return errors.Wrap(err, "failed to stop blinking physical drive")
	}

	return nil
}
