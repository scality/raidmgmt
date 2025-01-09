package logicalvolume

import (
	"fmt"
)

// Validate checks if the CacheOptions instance is valid.
func (co *CacheOptions) Validate() error {

	if co.ReadPolicy == ReadPolicyUnknown {
		return ErrUnknownReadPolicy
	}

	if co.WritePolicy == WritePolicyUnknown {
		return ErrUnknownWritePolicy
	}

	if co.IOPolicy == IOPolicyUnknown {
		return ErrUnknownIOPolicy
	}

	return nil
}

// Validate checks if the LogicalVolumeMeta instance is valid.
func (m *Metadata) Validate() error {
	if m == nil {
		return ErrMetadataNil
	}

	if m.CtrlMetadata == nil {
		return ErrControllerMetaNil
	}

	if err := m.CtrlMetadata.Validate(); err != nil {
		return fmt.Errorf("%s: %w", prefixMetadataErr, err)
	}

	return nil
}

// Validate checks if the Request instance is valid.
func (r *Request) Validate() error {

	if r == nil {
		return ErrRequestNil
	}

	if err := r.CtrlMetadata.Validate(); err != nil {
		return fmt.Errorf("%s: %w", prefixRequestErr, err)
	}

	if r.RAIDLevel == RAIDLevelUnknown {
		return ErrUnknownRAIDLevel
	}

	if len(r.PDrivesMetadata) == 0 {
		return ErrEmptyPhysicalDrives
	}

	if err := r.checkRAIDRequirement(); err != nil {
		return fmt.Errorf("%s: %w", prefixRequestErr, err)
	}

	for _, pdm := range r.PDrivesMetadata {
		if err := pdm.Validate(); err != nil {
			return fmt.Errorf("%s: %w", prefixRequestErr, err)
		}
	}

	if err := r.CacheOptions.Validate(); err != nil {
		return fmt.Errorf("%s: %w", prefixRequestErr, err)
	}

	return nil
}

// checkRAIDRequirement checks if the RAID level requirements are met.
// Only regarding the number of physical drives.
// The identical disk size requirement cannot be checked here.
func (r *Request) checkRAIDRequirement() error {
	if r.RAIDLevel == RAIDLevel1 {
		if len(r.PDrivesMetadata) < RAID1DiskRequirement {
			return ErrNotEnoughPhysicalDrives
		}
	}

	if r.RAIDLevel == RAIDLevel10 {
		if len(r.PDrivesMetadata) < RAID10DiskRequirement {
			return ErrNotEnoughPhysicalDrives
		}

		if len(r.PDrivesMetadata)%2 != 0 {
			return ErrOddNumberOfPhysicalDrives
		}
	}

	return nil
}

func (lv *LogicalVolume) ToMetadata() *Metadata {
	if lv == nil {
		return nil
	}

	return &Metadata{
		CtrlMetadata: lv.CtrlMetadata,
		ID:           lv.ID,
	}
}
