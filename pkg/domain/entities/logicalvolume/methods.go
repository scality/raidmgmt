package logicalvolume

import (
	"github.com/pkg/errors"
)

// Validate checks if the CacheOptions instance is valid.
func (co *CacheOptions) Validate() error {
	if co.ReadPolicy == ReadPolicyUnknown {
		return errors.New("read policy is unknown")
	}

	if co.WritePolicy == WritePolicyUnknown {
		return errors.New("write policy is unknown")
	}

	if co.IOPolicy == IOPolicyUnknown {
		return errors.New("io policy is unknown")
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
func (r *Request) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}

	if err := r.CtrlMetadata.Validate(); err != nil {
		return errors.Wrap(err, "controller metadata is invalid")
	}

	if r.RAIDLevel == RAIDLevelUnknown {
		return errors.New("raid level is unknown")
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

	if err := r.CacheOptions.Validate(); err != nil {
		return errors.Wrap(err, "cache options are invalid")
	}

	return nil
}

// checkRAIDRequirement checks if the RAID level requirements are met.
// Only regarding the number of physical drives.
// The identical disk size requirement cannot be checked here.
func (r *Request) checkRAIDRequirement() error {
	if r.RAIDLevel == RAIDLevel1 {
		if len(r.PDrivesMetadata) < RAID1DiskRequirement {
			return errors.New("not enough physical drives for RAID 1")
		}
	}

	if r.RAIDLevel == RAIDLevel10 {
		if len(r.PDrivesMetadata) < RAID10DiskRequirement {
			return errors.New("not enough physical drives for RAID 10")
		}

		if len(r.PDrivesMetadata)%2 != 0 {
			return errors.New("odd number of physical drives for RAID 10")
		}
	}

	return nil
}

// ToMetadata returns the Metadata instance of the LogicalVolume.
func (lv *LogicalVolume) ToMetadata() *Metadata {
	if lv == nil {
		return nil
	}

	return &Metadata{
		CtrlMetadata: lv.CtrlMetadata,
		ID:           lv.ID,
	}
}
