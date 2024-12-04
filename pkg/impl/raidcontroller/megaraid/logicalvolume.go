package megaraid

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

const (
	RAID0DiskRequirement  = 1
	RAID1DiskRequirement  = 2
	RAID10DiskRequirement = 4

	patternLV string = "/c%s/v%s"
)

var RAIDLevelMap = map[string]logicalvolume.RAIDLevel{
	"RAID0":  logicalvolume.RAIDLevel0,
	"RAID1":  logicalvolume.RAIDLevel1,
	"RAID10": logicalvolume.RAIDLevel10,
}

var LVStatusMap = map[string]logicalvolume.LVStatus{
	"Optl": logicalvolume.LVStatusOptimal,
	// TODO : check the real values
	"Dgrd": logicalvolume.LVStatusDegraded,
	"OfLn": logicalvolume.LVStatusOffline,
	"Pdgd": logicalvolume.LVStatusPartiallyDegraded,
	"Fail": logicalvolume.LVStatusFailed,
}

// logicalvolumes returns all logical volumes for a given controller.
func (m *Adapter) logicalvolumes(metadata *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error) {
	vds, err := m.ShowAllVirtualDrives(metadata.ID)
	if err != nil {
		return nil, err
	}

	// Get the physical drives for the controller
	physicalDrives, err := m.PhysicalDrives(metadata)
	if err != nil {
		return nil, err
	}

	ctrl, err := m.ControllerByID(metadata.ID)
	if err != nil {
		return nil, err
	}

	// Skip the error as the ID is already validated
	ctrlMetaIDInt, _ := metadata.IDInt()

	logicalVolumes := make([]*logicalvolume.LogicalVolume, 0)

	for _, vd := range vds {
		_, virtualDriveID := vd.DeviceGroupVirtualDrive()

		output, err := m.cmd.Run([]string{
			fmt.Sprintf(patternLV, metadata.ID, virtualDriveID),
			"show", "all",
		})
		if err != nil {
			return nil, err
		}

		responseData, err := output.GetResponseDataByCtrlID(ctrlMetaIDInt)
		if err != nil {
			return nil, err
		}

		key := fmt.Sprintf("VD%s Properties", virtualDriveID)

		vdProperties, err := unmarshalToPointer[VDProperties](responseData, key)
		if err != nil {
			return nil, err
		}

		key = fmt.Sprintf("PDs for VD %s", virtualDriveID)

		pDrives, err := unmarshalToSlice[PD](responseData, key)
		if err != nil {
			return nil, err
		}

		associatedPDrives, err := matchPhysicalDrives(physicalDrives, pDrives)
		if err != nil {
			return nil, err
		}

		logicalVolume := &logicalvolume.LogicalVolume{
			Controller:     ctrl,
			ID:             virtualDriveID,
			DevicePath:     vdProperties.OSDriveName,
			RAIDLevel:      vd.RAIDLevel(),
			PhysicalDrives: associatedPDrives,
			CacheOptions:   vd.CacheOptions(),
			Status:         vd.LVStatus(),
		}

		logicalVolumes = append(logicalVolumes, logicalVolume)
	}

	sort.Slice(logicalVolumes, func(i, j int) bool {
		a, _ := strconv.Atoi(logicalVolumes[i].ID)
		b, _ := strconv.Atoi(logicalVolumes[j].ID)

		return a < b
	})

	return logicalVolumes, nil
}

// DeviceGroupVirtualDrive returns the device group and virtual drive number.
func (vd *VD) DeviceGroupVirtualDrive() (deviceGroup, virtualDrive string) {
	deviceGroupVirtualDrive := strings.Split(vd.DGVD, "/")

	return deviceGroupVirtualDrive[0], deviceGroupVirtualDrive[1]
}

// RAIDLevel returns the RAID level of a logical volume.
func (vd *VD) RAIDLevel() logicalvolume.RAIDLevel {
	if raidLevel, ok := RAIDLevelMap[vd.Type]; ok {
		return raidLevel
	}

	return logicalvolume.RAIDLevelUnknown
}

// CacheOptions returns the cache options for a logical volume.
func (vd *VD) CacheOptions() *logicalvolume.CacheOptions {
	return &logicalvolume.CacheOptions{
		ReadPolicy:  parseReadPolicy(vd.Cache),
		WritePolicy: parseWritePolicy(vd.Cache),
		IOPolicy:    parseIOPolicy(vd.Cache),
	}
}

// parseReadPolicy parses the read policy from the cache string.
func parseReadPolicy(cache string) logicalvolume.ReadPolicy {
	switch {
	case strings.Contains(cache, "R"):
		return logicalvolume.ReadPolicyReadAhead
	case strings.Contains(cache, "NR"):
		return logicalvolume.ReadPolicyNoReadAhead
	default:
		return logicalvolume.ReadPolicyUnknown
	}
}

// parseWritePolicy parses the write policy from the cache string.
func parseWritePolicy(cache string) logicalvolume.WritePolicy {
	switch {
	case strings.Contains(cache, "WB"):
		return logicalvolume.WritePolicyWriteBack
	case strings.Contains(cache, "AWB"):
		return logicalvolume.WritePolicyAlwaysWriteBack
	case strings.Contains(cache, "WT"):
		return logicalvolume.WritePolicyWriteThrough
	default:
		return logicalvolume.WritePolicyUnknown
	}
}

// parseIOPolicy parses the IO policy from the cache string.
func parseIOPolicy(cache string) logicalvolume.IOPolicy {
	switch {
	case strings.Contains(cache, "C"):
		return logicalvolume.IOPolicyCached
	case strings.Contains(cache, "D"):
		return logicalvolume.IOPolicyDirect
	default:
		return logicalvolume.IOPolicyUnknown
	}
}

// LVStatus returns the logical volume status.
func (vd *VD) LVStatus() logicalvolume.LVStatus {
	if status, ok := LVStatusMap[vd.State]; ok {
		return status
	}

	return logicalvolume.LVStatusUnknown
}

// matchPhysicalDrives matches physical drives with physical drive
// information from the controller.
func matchPhysicalDrives(allPDrives []*physicaldrive.PhysicalDrive,
	pdList []PD,
) ([]*physicaldrive.PhysicalDrive, error) {
	// Create a map from physicalDrives with ID as the key.
	driveMap := make(map[int]*physicaldrive.PhysicalDrive)

	for i := range allPDrives {
		idInt, err := strconv.Atoi(allPDrives[i].ID)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrMatchPhysicalDrives, err)
		}

		driveMap[idInt] = allPDrives[i]
	}

	// Prepare a slice to store pointers to matched PhysicalDrive structs.
	var matches []*physicaldrive.PhysicalDrive

	// Iterate over pdList to find matches in the driveMap.
	for _, pd := range pdList {
		if physicalDrive, found := driveMap[pd.DeviceID]; found {
			// Store the pointer to the matched PhysicalDrive.
			matches = append(matches, physicalDrive)
		}
	}

	return matches, nil
}

// selectorLV returns the selector for a logical volume.
func selectorLV(m *logicalvolume.Metadata) string {
	return fmt.Sprintf(patternLV, m.CtrlMetadata.ID, m.ID)
}

// logicalVolume returns a logical volume for a given logical volume metadata.
func (m *Adapter) logicalVolume(
	metadata *logicalvolume.Metadata) (
	*logicalvolume.LogicalVolume, error,
) {
	lvs, err := m.logicalvolumes(metadata.CtrlMetadata)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLogicalVolumes, err)
	}

	for _, lv := range lvs {
		if lv.ID == metadata.ID {
			return lv, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrLogicalVolumeNotFound, selectorLV(metadata))
}

// deleteLV deletes a logical volume.
func (m *Adapter) deleteLV(metadata *logicalvolume.Metadata) error {
	selector := selectorLV(metadata)
	_, err := m.cmd.Run([]string{selector, "delete"})

	return err
}

// createLV creates a logical volume.
func (m *Adapter) createLV(
	request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume, error,
) {
	// Get the logical volumes before creating the new one
	lvsBefore, err := m.logicalvolumes(request.CtrlMetadata)
	if err != nil {
		return nil, err
	}

	selector := selectorCtrl(request.CtrlMetadata)

	raidLevel := fmt.Sprintf("type=raid%d", request.RAIDLevel)

	// Get the physical drives for the controller
	err = checkRAIDRequirement(request.RAIDLevel, request.PDrivesMetadata)
	if err != nil {
		return nil, err
	}

	// Check if the physical drives are available
	err = checkAvailabilityPDMetadata(m, request.PDrivesMetadata)
	if err != nil {
		return nil, err
	}

	// Prepare the string of drives
	enclosure, slots, err := slotsEnclosure(request.PDrivesMetadata)
	if err != nil {
		return nil, err
	}

	drives := fmt.Sprintf("%d:%s", enclosure, strings.Join(slots, ","))

	read, write, io := cacheOptions(request)

	_, err = m.cmd.Run([]string{selector, "add", "vd", raidLevel, "drives=" + drives, read, write, io})
	if err != nil {
		return nil, err
	}

	// Get the newly created logical volume

	lvsAfter, err := m.logicalvolumes(request.CtrlMetadata)
	if err != nil {
		return nil, err
	}

	newLV, err := findNewLogicalVolume(lvsBefore, lvsAfter)
	if err != nil {
		return nil, err
	}

	return newLV, nil
}

func slotsEnclosure(pdsMetadata []*physicaldrive.Metadata) (int, []string, error) {
	enclosures := make(map[int]struct{})
	slots := make([]string, len(pdsMetadata))

	for i, pd := range pdsMetadata {
		enclosure, bay := pd.Slot.Enclosure, pd.Slot.Bay

		if enclosure < 0 {
			return -1, nil, fmt.Errorf("%w: %d", ErrInvalidEnclosureID, enclosure)
		}

		if bay < 0 {
			return -1, nil, fmt.Errorf("%w: %d", ErrInvalidSlotID, bay)
		}

		// Add the enclosure to the map
		// to check if there are multiple enclosures
		// in the same logical volume
		enclosures[enclosure] = struct{}{}

		slots[i] = fmt.Sprintf("%d", bay)
	}

	// Check if there are multiple enclosures
	if len(enclosures) > 1 {
		return -1, nil, ErrMultipleEnclosuresNotSupported
	}

	// Get the enclosure number
	enclosure := 0
	for key := range enclosures {
		enclosure = key
	}

	return enclosure, slots, nil
}

// cacheOptions returns the cache options for a logical volume.
func cacheOptions(lv *logicalvolume.Request) (read, write, io string) {
	read = lv.CacheOptions.ReadPolicy.String()
	write = lv.CacheOptions.WritePolicy.String()
	io = lv.CacheOptions.IOPolicy.String()

	return read, write, io
}

// checkRAIDRequirement validates the number of physical drives
// for a given RAID level.
func checkRAIDRequirement(
	raidLevel logicalvolume.RAIDLevel,
	pds []*physicaldrive.Metadata,
) error {
	// Check the RAID level and the number of physical drives
	switch raidLevel {
	case logicalvolume.RAIDLevel0:
		if len(pds) < RAID0DiskRequirement {
			return ErrRaid0RequiresAtLeast1Drive
		}
	case logicalvolume.RAIDLevel1:
		if len(pds) != RAID1DiskRequirement {
			return ErrRaid1Requires2Drives
		}
	case logicalvolume.RAIDLevel10:
		if len(pds) < RAID10DiskRequirement {
			return ErrRaid10RequiresAtLeast4
		}
	default:
		return ErrInvalidRAIDLevel
	}

	return nil
}

// setLVCacheOptions sets the cache options for a logical volume.
func (m *Adapter) setLVCacheOptions(
	metadata *logicalvolume.Metadata,
	cacheOpts *logicalvolume.CacheOptions,
) error {
	selector := selectorLV(metadata)

	lv, err := m.logicalVolume(metadata)
	if err != nil {
		return err
	}

	// Dynamically build the options slice
	var options []string

	if cacheOpts.ReadPolicy != lv.CacheOptions.ReadPolicy {
		options = append(options, "rdcache="+cacheOpts.ReadPolicy.String())
	}

	if cacheOpts.WritePolicy != lv.CacheOptions.WritePolicy {
		options = append(options, "wrcache="+cacheOpts.WritePolicy.String())
	}

	if cacheOpts.IOPolicy != lv.CacheOptions.IOPolicy {
		options = append(options, "iopolicy="+cacheOpts.IOPolicy.String())
	}

	// If no options need to be updated, return the appropriate error
	if len(options) == 0 {
		return ErrNoCacheOptionsToUpdate
	}

	// Pass only non-empty options to the command
	args := []string{selector, "set"}
	_, err = m.cmd.Run(append(args, options...))
	// _, err = m.cmd.Run([]string{selector, "set", options...})

	return err
}

func findNewLogicalVolume(
	before, after []*logicalvolume.LogicalVolume) (
	*logicalvolume.LogicalVolume, error,
) {
	beforeMap := make(map[string]*logicalvolume.LogicalVolume)
	for _, b := range before {
		beforeMap[b.ID] = b
	}

	newVolumes := []*logicalvolume.LogicalVolume{}

	for _, a := range after {
		if _, exists := beforeMap[a.ID]; !exists {
			newVolumes = append(newVolumes, a)
		}
	}

	if len(newVolumes) == 0 {
		return nil, ErrNewLogicalVolumeNotFound
	}

	// Check if there are multiple new logical volumes
	if len(newVolumes) > 1 {
		ids := make([]string, len(newVolumes))
		for i, nv := range newVolumes {
			ids[i] = nv.ID
		}

		return nil, fmt.Errorf("%w: %s", ErrMultipleNewLogicalVolumes, strings.Join(ids, " "))
	}

	return newVolumes[0], nil
}
