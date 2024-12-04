package logicalvolume

import (
	"fmt"
)

// Validate checks if the CacheOptions instance is valid.
func (co *CacheOptions) Validate() error {
	if co == nil {
		return ErrCacheOptionsNil
	}

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

	if r.CtrlMetadata == nil {
		return ErrControllerMetaNil
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

	for _, pdm := range r.PDrivesMetadata {
		if err := pdm.Validate(); err != nil {
			return fmt.Errorf("%s: %w", prefixRequestErr, err)
		}
	}

	if r.CacheOptions == nil {
		return ErrCacheOptionsNil
	}

	if err := r.CacheOptions.Validate(); err != nil {
		return fmt.Errorf("%s: %w", prefixRequestErr, err)
	}

	return nil
}
