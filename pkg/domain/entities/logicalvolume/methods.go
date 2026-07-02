package logicalvolume

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
)

// Validate checks if the CacheOptions instance is valid. It rejects a nil
// receiver (rather than panicking) so callers that require cache options, such
// as SetLVCacheOptions, surface a clean error. The read and write policies must
// be known settable values. The IO policy is optional -- not every controller
// supports it (storcli2 dropped it entirely) -- so it is only rejected when the
// caller set it to an unrecognized value; an unset or Unknown IO policy is
// accepted and left to the controller.
func (co *CacheOptions) Validate() error {
	if co == nil {
		return errors.New("cache options are nil")
	}

	if !co.ReadPolicy.IsValid() {
		return errors.Errorf("invalid read policy: %q", co.ReadPolicy)
	}

	if !co.WritePolicy.IsValid() {
		return errors.Errorf("invalid write policy: %q", co.WritePolicy)
	}

	// storcli2 dropped the IO policy, so an unset or Unknown value is accepted and
	// left to the controller; only an unrecognized value is rejected.
	switch co.IOPolicy {
	case "", IOPolicyUnknown, IOPolicyDirect, IOPolicyCached:
	default:
		return errors.Errorf("invalid io policy: %q", co.IOPolicy)
	}

	return nil
}

// Validate checks if the LogicalVolumeMeta instance is valid.
func (m *Metadata) Validate() error {
	if m == nil {
		return errors.New("metadata is nil")
	}

	if m.ID == "" {
		return errors.New("id is empty")
	}

	if err := m.CtrlMetadata.Validate(); err != nil {
		return errors.Wrap(err, "controller metadata is invalid")
	}

	return nil
}

// Validate checks if the Request instance is valid.
//
//nolint:funlen // this is a validation method
func (r *Request) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if err := r.CtrlMetadata.Validate(); err != nil {
		return errors.Wrap(err, "controller metadata is invalid")
	}

	if !r.RAIDLevel.IsValid() {
		return errors.Errorf("invalid raid level: %s", r.RAIDLevel)
	}

	if len(r.PDrivesMetadata) == 0 {
		return errors.New("physical drives is empty")
	}

	if err := r.checkRAIDRequirement(); err != nil {
		return errors.Wrap(err, "raid level requirement is not met")
	}

	for _, pdm := range r.PDrivesMetadata {
		if err := pdm.Validate(); err != nil {
			return errors.Wrap(err, "physical drive metadata is invalid")
		}
	}

	if r.CacheOptions == nil {
		return nil
	}

	if err := r.CacheOptions.Validate(); err != nil {
		return errors.Wrap(err, "cache options are invalid")
	}

	return nil
}

// checkRAIDRequirement checks if the RAID level requirements are met.
// Only regarding the number of physical drives.
// The identical disk size requirement cannot be checked here
// because we only have the metadata and not the actual size.
func (r *Request) checkRAIDRequirement() error {
	switch r.RAIDLevel {
	case RAIDLevel0:
		if len(r.PDrivesMetadata) < RAID0DiskRequirement {
			return errors.New("not enough physical drives for RAID 0")
		}
	case RAIDLevel1:
		if len(r.PDrivesMetadata) < RAID1DiskRequirement {
			return errors.New("not enough physical drives for RAID 1")
		}
	case RAIDLevel10:
		if len(r.PDrivesMetadata) < RAID10DiskRequirement {
			return errors.New("not enough physical drives for RAID 10")
		}

		if len(r.PDrivesMetadata)%2 != 0 {
			return errors.New("odd number of physical drives for RAID 10")
		}
	case RAIDLevelUnknown:
		return errors.New("unknown RAID level")
	default:
		return errors.Errorf("unsupported RAID level: %s", r.RAIDLevel)
	}

	return nil
}

// ValidateRAIDCreation validates the creation of a RAID logical volume
// by checking the availability of the physical drives and their sizes.
// We need to get the full physical drive instances to check the sizes.
func ValidateRAIDCreation(
	pds []*physicaldrive.PhysicalDrive,
	raidLevel RAIDLevel,
) error {
	if len(pds) == 0 {
		return errors.New("no physical drives")
	}

	if !raidLevel.IsValid() {
		return errors.Errorf("invalid RAID level: %s", raidLevel)
	}

	// Check if there are unavailable drives
	unavailableDrives := unavailableDrives(pds)

	// If there are unavailable drives, return an error
	if len(unavailableDrives) > 0 {
		return errors.Errorf(ErrUnavailableDrives, strings.Join(unavailableDrives, ", "))
	}

	// Don't check size for RAID 0
	if raidLevel == RAIDLevel0 {
		return nil
	}

	// Find the most frequent size among the physical drives
	mode := findMostFrequentSize(pds)

	// Collect IDs of drives that don't fit within the tolerance of the mode size
	outsideToleranceIDs := outsideToleranceIDs(pds, mode)

	// If there are mismatches, return an error
	if len(outsideToleranceIDs) > 0 {
		return errors.Errorf("mismatched sizes for drives with IDs: %v", outsideToleranceIDs)
	}

	return nil
}

// unavailableDrives returns the IDs of the unavailable physical drives.
func unavailableDrives(pds []*physicaldrive.PhysicalDrive) []string {
	var unavailableDrives []string

	for _, pd := range pds {
		// Check if the physical drive is available
		if !pd.IsAvailable() {
			unavailableDrives = append(unavailableDrives, pd.Slot.String())
		}
	}

	return unavailableDrives
}

func outsideToleranceIDs(pds []*physicaldrive.PhysicalDrive, mostFrequentSize uint64) []string {
	var outsideToleranceIDs []string

	// Check if the size of each drive is within the tolerance of the mode size
	lowerLimit := mostFrequentSize - (mostFrequentSize * sizeTolerancePercent / percent)
	upperLimit := mostFrequentSize + (mostFrequentSize * sizeTolerancePercent / percent)

	for _, pd := range pds {
		if pd.Size < lowerLimit || pd.Size > upperLimit {
			outsideToleranceIDs = append(outsideToleranceIDs, pd.ID)
		}
	}

	return outsideToleranceIDs
}

// findMostFrequentSize finds the most frequent size among the physical drives.
func findMostFrequentSize(pds []*physicaldrive.PhysicalDrive) uint64 {
	// Count occurrences of each size
	sizeCounts := make(map[uint64]int)

	for _, drive := range pds {
		sizeCounts[drive.Size]++
	}

	// Find the most frequent size (mode)
	var mostFrequentSize uint64

	maxCount := 0
	for size, count := range sizeCounts {
		if count > maxCount {
			mostFrequentSize = size
			maxCount = count
		}
	}

	return mostFrequentSize
}

// RAIDLevelMap maps the RAID level string to the RAID level type.
func RAIDLevelMap(str string) RAIDLevel {
	// Remove the "RAID" prefix from the string if it exists
	raidLevelString := strings.TrimPrefix(str, "RAID")

	// raidLevelMap maps the RAID level string to the RAID level type.
	raidLevelMap := map[string]RAIDLevel{
		"0":  RAIDLevel0,
		"1":  RAIDLevel1,
		"10": RAIDLevel10,
	}

	if raidLevel, ok := raidLevelMap[raidLevelString]; ok {
		return raidLevel
	}

	return RAIDLevelUnknown
}
