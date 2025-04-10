package megaraid

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/utils"
)

// patternLV is the pattern for the logical volume selector.
const (
	patternLV string = "/c%d/v%s"
)

// logicalvolumes returns all logical volumes for a given controller.
func (a *Adapter) logicalvolumes(metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume,
	error,
) {
	selector := selectorCtrl(metadata)

	vds, err := a.showAllVirtualDrives(selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get all virtual drives")
	}

	logicalVolumes := make([]*logicalvolume.LogicalVolume, 0)

	for _, vd := range vds {
		virtualDriveID := vd.VirtualDriveID()

		lvMetadata := &logicalvolume.Metadata{
			CtrlMetadata: metadata,
			ID:           virtualDriveID,
		}

		logicalVolume, err := a.logicalVolume(lvMetadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get logical volume %s", virtualDriveID)
		}

		logicalVolumes = append(logicalVolumes, logicalVolume)
	}

	sort.Slice(logicalVolumes, func(i, j int) bool {
		// Pass the error check because the slice is already validated
		//nolint:errcheck // no err is possible since it's already validated
		a, _ := strconv.Atoi(logicalVolumes[i].ID)
		//nolint:errcheck // same as above
		b, _ := strconv.Atoi(logicalVolumes[j].ID)

		return a < b
	})

	return logicalVolumes, nil
}

// VirtualDriveID returns the Virtual Drive ID.
// The Device Group is not used.
func (vd *VD) VirtualDriveID() string {
	deviceGroupVirtualDrive := strings.Split(vd.DGVD, "/")

	return deviceGroupVirtualDrive[1]
}

// CacheOptions returns the cache options for a logical volume.
//
//nolint:funlen // The function is not too complex
func (vd *VD) CacheOptions() (*logicalvolume.CacheOptions, error) {
	if vd.Cache == "" {
		return nil, errors.New("no cache options found")
	}

	// Since the cache options have different formats, we need to check
	// the prefix of the cache string to determine the read policy.
	// Then, we remove the policy from the cache string to parse the
	// write policy and IO policy.

	remaining := vd.Cache

	// Parsing read policy
	readPolicy := logicalvolume.ReadPolicyUnknown

	switch {
	case strings.HasPrefix(vd.Cache, "R"):
		readPolicy = logicalvolume.ReadPolicyReadAhead
		remaining = strings.TrimPrefix(vd.Cache, "R")
	case strings.HasPrefix(vd.Cache, "NR"):
		readPolicy = logicalvolume.ReadPolicyNoReadAhead
		remaining = strings.TrimPrefix(vd.Cache, "NR")
	}

	// Parsing write policy
	writePolicy := logicalvolume.WritePolicyUnknown

	switch {
	case strings.HasPrefix(remaining, "WB"):
		writePolicy = logicalvolume.WritePolicyWriteBack
		remaining = strings.TrimPrefix(remaining, "WB")
	case strings.HasPrefix(remaining, "AWB"):
		writePolicy = logicalvolume.WritePolicyAlwaysWriteBack
		remaining = strings.TrimPrefix(remaining, "AWB")
	case strings.HasPrefix(remaining, "WT"):
		writePolicy = logicalvolume.WritePolicyWriteThrough
		remaining = strings.TrimPrefix(remaining, "WT")
	}

	// Parsing IO policy
	ioPolicy := logicalvolume.IOPolicyUnknown

	switch {
	case strings.HasPrefix(remaining, "C"):
		ioPolicy = logicalvolume.IOPolicyCached
		remaining = strings.TrimPrefix(remaining, "C")
	case strings.HasPrefix(remaining, "D"):
		ioPolicy = logicalvolume.IOPolicyDirect
		remaining = strings.TrimPrefix(remaining, "D")
	}

	if remaining != "" {
		return nil, errors.Errorf(ErrUnrecognizedCacheOptions, vd.Cache)
	}

	return &logicalvolume.CacheOptions{
		ReadPolicy:  readPolicy,
		WritePolicy: writePolicy,
		IOPolicy:    ioPolicy,
	}, nil
}

// LVStatus returns the logical volume status.
func (vd *VD) LVStatus() logicalvolume.LVStatus {
	// lvStatusMap maps the logical volume status string to the logical volume status type.
	lvStatusMap := map[string]logicalvolume.LVStatus{
		"Optl": logicalvolume.LVStatusOptimal,
		// TODO : check the real values and add reason for those statuses
		"Dgrd": logicalvolume.LVStatusDegraded,
		"Pdgd": logicalvolume.LVStatusDegraded,
		"Fail": logicalvolume.LVStatusFailed,
	}

	if status, ok := lvStatusMap[vd.State]; ok {
		return status
	}

	return logicalvolume.LVStatusUnknown
}

// selectorLV returns the selector for a logical volume.
func selectorLV(m *logicalvolume.Metadata) (string, error) {
	_, err := strconv.Atoi(m.ID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to convert logical volume ID to int: %s", m.ID)
	}

	return fmt.Sprintf(patternLV, m.CtrlMetadata.ID, m.ID), nil
}

// logicalVolume returns a logical volume for a given logical volume metadata.
//
//nolint:funlen,cyclop // The function is not too complex
func (a *Adapter) logicalVolume(
	metadata *logicalvolume.Metadata) (
	*logicalvolume.LogicalVolume, error,
) {
	selector, err := selectorLV(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get selector")
	}

	responseData, err := a.showAllVirtualDrive(selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get virtual drive info")
	}

	vds, err := utils.UnmarshalToSlice[VD](responseData, selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal virtual drive")
	}

	if err := validateVDs(vds); err != nil {
		return nil, errors.Wrap(err, "failed to validate virtual drives")
	}

	vd := vds[0]

	cacheOptions, err := vd.CacheOptions()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cache options")
	}

	selector = fmt.Sprintf("PDs for VD %s", metadata.ID)

	pDrives, err := utils.UnmarshalToSlice[PD](responseData, selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal PDs")
	}

	// Get the physical drives metadata
	pdsMetadata := make([]*physicaldrive.Metadata, len(pDrives))

	for i := range pDrives {
		pdMetadata := &physicaldrive.Metadata{
			CtrlMetadata: metadata.CtrlMetadata,
			ID:           pDrives[i].EIDSlot,
		}

		pdsMetadata[i] = pdMetadata
	}

	selector = fmt.Sprintf("VD%s Properties", metadata.ID)

	// Get the VD properties for the permanent path
	vdProperties, err := utils.UnmarshalToPointer[VDProperties](responseData, selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal VD properties")
	}

	// Get the physical drives from the metadata
	// NOTE: This is needed as fallback to get the device path
	pds, err := a.fillPhysicalDrives(pdsMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fill physical drives")
	}

	devicePath, permanentPath, err := getPaths(vdProperties, pds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get paths")
	}

	logicalVolume := &logicalvolume.LogicalVolume{
		Metadata:        metadata,
		DevicePath:      devicePath,
		RAIDLevel:       logicalvolume.RAIDLevelMap(vd.Type),
		PDrivesMetadata: pdsMetadata,
		CacheOptions:    cacheOptions,
		Status:          vd.LVStatus(),
		PermanentPath:   permanentPath,
		// TODO
		// Reason:          "",
	}

	return logicalVolume, nil
}

func validateVDs(vds []VD) error {
	if len(vds) == 0 {
		return errors.New("no virtual drive found")
	}

	if len(vds) > 1 {
		return errors.New("multiple virtual drives found")
	}

	return nil
}

// deleteLV deletes a logical volume.
func (a *Adapter) deleteLV(metadata *logicalvolume.Metadata) error {
	selector, err := selectorLV(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to get selector")
	}

	_, err = a.runner.Run([]string{selector, "delete"})
	if err != nil {
		return errors.Wrap(err, ErrCommandFailed.Error())
	}

	return nil
}

// createLV creates a logical volume.
func (a *Adapter) createLV(request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume, error,
) {
	selector := selectorCtrl(request.CtrlMetadata)

	raidLevel := fmt.Sprintf("type=raid%d", request.RAIDLevel.Level())

	// Get the physical drives from the metadata
	pds, err := a.fillPhysicalDrives(request.PDrivesMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fill physical drives")
	}

	// Check if the physical drives are available
	err = logicalvolume.ValidateRAIDCreation(pds, request.RAIDLevel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate RAID creation")
	}

	drives, err := formatDrivesString(request.PDrivesMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to format drives string")
	}

	// Prepare the cache options
	read := string("rdpolicy=" + request.CacheOptions.ReadPolicy)
	write := string("wrcache=" + request.CacheOptions.WritePolicy)
	io := string("iopolicy=" + request.CacheOptions.IOPolicy)

	args := []string{selector, "add", "vd", raidLevel, drives, read, write, io}

	_, err = a.runner.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, ErrCommandFailed.Error())
	}

	// Get the newly created logical volume
	newLV, err := a.findNewLogicalVolume(request.PDrivesMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find new logical volume")
	}

	return newLV, nil
}

func formatDrivesString(pdMetas []*physicaldrive.Metadata) (string, error) {
	enclosure, slots, err := enclosureSlots(pdMetas)
	if err != nil {
		return "", errors.Wrap(err, "failed to get enclosure and slots")
	}

	slotsJoined := strings.Join(slots, ",")
	drives := fmt.Sprintf("drives=%d:%s", enclosure, slotsJoined)

	// If the enclosure is not set, reformat the drives string
	if enclosure < 0 {
		drives = fmt.Sprintf("drives=%s", slotsJoined)
	}

	return drives, nil
}

func (a *Adapter) fillPhysicalDrives(pdMetadatas []*physicaldrive.Metadata) (
	[]*physicaldrive.PhysicalDrive,
	error,
) {
	pds := make([]*physicaldrive.PhysicalDrive, len(pdMetadatas))

	for i, pdMeta := range pdMetadatas {
		pd, err := a.physicalDrive(pdMeta)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive %s",
				pdMeta.ID)
		}

		pds[i] = pd
	}

	return pds, nil
}

// enclosureSlots returns the enclosure number and the slots of the physical drives.
//
// The function is not too complex, and the complexity is due to the
// multiple checks and conversions.
//
//nolint:gocognit,funlen // The function is actually not too complex
func enclosureSlots(pdsMetadatas []*physicaldrive.Metadata) (
	enclosure int,
	slots []string,
	err error,
) {
	// Map to check if there are multiple enclosures
	enclosures := make(map[int]struct{})
	defaultEnclosure := -1

	// Slice to store the slots
	slots = make([]string, len(pdsMetadatas))

	for i, pd := range pdsMetadatas {
		slot, err := physicaldrive.ParseSlot(pd.ID)
		if err != nil {
			return defaultEnclosure, nil, errors.Wrapf(err, "failed to parse slot %s", pd.ID)
		}

		enclosure, bay := slot.Enclosure, slot.Bay

		enclosureInt, err := strconv.Atoi(enclosure)
		if err != nil {
			return defaultEnclosure, nil, errors.Wrap(err, "failed to convert enclosure to int")
		}

		if enclosureInt < 0 {
			return defaultEnclosure, nil, errors.Errorf(ErrInvalidEnclosureID, enclosure)
		}

		bayInt, err := strconv.Atoi(bay)
		if err != nil {
			return defaultEnclosure, nil, errors.Wrap(err, "failed to convert bay to int")
		}

		if bayInt < 0 {
			return defaultEnclosure, nil, errors.Errorf(ErrInvalidBayID, bay)
		}

		// Add the enclosure to the map
		// to check if there are multiple enclosures
		// in the same logical volume
		enclosures[enclosureInt] = struct{}{}

		slots[i] = bay
	}

	// Check if there are multiple enclosures
	if len(enclosures) > 1 {
		return defaultEnclosure, nil, errors.New("multiple enclosures not supported")
	}

	// Get the enclosure number
	enclosure = defaultEnclosure
	for key := range enclosures {
		enclosure = key
	}

	return enclosure, slots, nil
}

// setLVCacheOptions sets the cache options for a logical volume.
func (a *Adapter) setLVCacheOptions(
	metadata *logicalvolume.Metadata,
	cacheOpts *logicalvolume.CacheOptions,
) error {
	// Get the logical volume
	lv, err := a.logicalVolume(metadata)
	if err != nil {
		return errors.Wrapf(err, "failed to get logical volume %s", metadata.ID)
	}

	// Dynamically build the options slice
	var options []string

	if cacheOpts.ReadPolicy != lv.CacheOptions.ReadPolicy {
		options = append(options, "rdcache="+string(cacheOpts.ReadPolicy))
	}

	if cacheOpts.WritePolicy != lv.CacheOptions.WritePolicy {
		options = append(options, "wrcache="+string(cacheOpts.WritePolicy))
	}

	if cacheOpts.IOPolicy != lv.CacheOptions.IOPolicy {
		options = append(options, "iopolicy="+string(cacheOpts.IOPolicy))
	}

	// If no options need to be updated, return nil
	if len(options) == 0 {
		return nil
	}

	// Get the selector
	selector, err := selectorLV(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to get selector")
	}

	// Build the arguments with the selector and the options
	args := append([]string{selector, "set"}, options...)

	_, err = a.runner.Run(args)
	if err != nil {
		return errors.Wrap(err, ErrCommandFailed.Error())
	}

	return nil
}

// migrate deletes or adds a physical drive to a logical volume.
func (a *Adapter) migrate(
	action string,
	lvMetadata *logicalvolume.Metadata,
	pdMetadatas ...*physicaldrive.Metadata,
) error {
	// Get the logical volume
	lv, err := a.logicalVolume(lvMetadata)
	if err != nil {
		return errors.Wrapf(err, "failed to get logical volume %s", lvMetadata.ID)
	}

	actionArg := fmt.Sprintf("option=%s", action)

	raidLevel := fmt.Sprintf("type=raid%d", lv.RAIDLevel.Level())

	drives, err := formatDrivesString(pdMetadatas)
	if err != nil {
		return errors.Wrap(err, "failed to format drives string")
	}

	selector, err := selectorLV(lvMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get selector")
	}

	args := []string{selector, "start", "migrate", raidLevel, actionArg, drives}

	_, err = a.runner.Run(args)
	if err != nil {
		return errors.Wrap(err, ErrCommandFailed.Error())
	}

	return nil
}

func (a *Adapter) findNewLogicalVolume(pds []*physicaldrive.Metadata) (
	*logicalvolume.LogicalVolume,
	error,
) {
	ctrlMetadata := pds[0].CtrlMetadata

	// Get logical volumes
	lvs, err := a.logicalvolumes(ctrlMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volumes")
	}

	// Create a map of physical drive slots for efficient lookup
	pdSlots := make(map[string]struct{})
	for _, pd := range pds {
		pdSlots[pd.ID] = struct{}{}
	}

	// Find the new logical volume
	for _, lv := range lvs {
		if hasMatchingPDs(lv.PDrivesMetadata, pdSlots) {
			return lv, nil
		}
	}

	return nil, errors.New("new logical volume not found")
}

// hasMatchingPDs checks if the logical volume has the same physical drives.
func hasMatchingPDs(lvPDs []*physicaldrive.Metadata, pdSlots map[string]struct{}) bool {
	for _, lvPD := range lvPDs {
		if _, found := pdSlots[lvPD.ID]; found {
			return true
		}
	}

	return false
}

// It is used to mock the functions in tests.
// nolint: gochecknoglobals // This is a variable that is used to mock a function in tests.
var (
	CustomEvalSymlinks = filepath.EvalSymlinks
	CustomFileExists   = utils.FileExists
)

// getPaths returns the device path and a permanent paths for the logical volumes.
func getPaths(vdp *VDProperties, pdrives []*physicaldrive.PhysicalDrive) (
	devicePath, permanentPath string, err error,
) {
	devicePath = vdp.OSDriveName

	permanentPath = fmt.Sprintf("/dev/disk/by-id/wwn-0x%s", vdp.SCSINAAID)
	if !CustomFileExists(permanentPath) {
		// If the permanent path is not found and there is only one physical drive,
		// we will try to get the path from the physical drive information
		// otherwise let's error here
		if len(pdrives) != 1 {
			return devicePath, "", errors.New("failed to get permanent path")
		}

		pd := pdrives[0]

		permanentPath = computePermanentPath(pd)
		if !CustomFileExists(permanentPath) {
			return devicePath, "", errors.New("failed to get permanent path from physical drive")
		}
	}

	// If the devicePath is empty let's retrieve it from the permanent path
	if devicePath == "" {
		devicePath, err = CustomEvalSymlinks(permanentPath)
		if err != nil {
			return "", "", errors.Wrap(err, "failed to evaluate symlink")
		}
	}

	return devicePath, permanentPath, nil
}
