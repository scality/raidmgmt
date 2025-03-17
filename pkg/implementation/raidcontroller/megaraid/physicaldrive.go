package megaraid

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/utils"
)

const (
	// patternEnclosure is the pattern for the physical drive selector with enclosure.
	patternEnclosure string = "/c%d/e%s/s%s"
	// patternNoEnclosure is the pattern for the physical drive selector without enclosure.
	patternNoEnclosure string = "/c%d/s%s"
)

// physicaldrives returns all physical drives for a given controller.
func (a *Adapter) physicaldrives(metadata *raidcontroller.Metadata) (
	[]*physicaldrive.PhysicalDrive, error,
) {
	selector := selectorCtrl(metadata)

	// Get the physical drives for the controller
	pds, err := a.showAllPhysicalDrives(selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get all physical drives")
	}

	// Prepare the slice of physical drives to return
	physicalDrives := make([]*physicaldrive.PhysicalDrive, 0)

	// Fill the slice of physical drives
	for _, pd := range pds {
		pdMetadata := &physicaldrive.Metadata{
			CtrlMetadata: metadata,
			ID:           pd.EIDSlot,
		}

		physicalDrive, err := a.physicalDrive(pdMetadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive %s",
				pdMetadata.ID)
		}

		physicalDrives = append(physicalDrives, physicalDrive)
	}

	sort.Slice(physicalDrives, func(i, j int) bool {
		// Pass the error check because the slice is already validated
		//nolint:errcheck // no err is possible since it's already validated
		a, _ := strconv.Atoi(physicalDrives[i].ID)
		//nolint:errcheck // no err is possible since it's already validated
		b, _ := strconv.Atoi(physicalDrives[j].ID)

		return a < b
	})

	return physicalDrives, nil
}

// physicalDrive returns a physical drive for a given physical drive metadata.
//
//nolint:funlen // no good reason to split it for now
func (a *Adapter) physicalDrive(
	metadata *physicaldrive.Metadata) (
	*physicaldrive.PhysicalDrive, error,
) {
	selector, err := selectorPD(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get selector")
	}

	// Get the physical drive
	responseData, err := a.showAllPhysicalDrive(selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get all physical drives")
	}

	key := "Drive " + selector

	pdList, err := utils.UnmarshalToSlice[PD](responseData, key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal physical drive")
	}

	pd := pdList[0]

	key = "Drive " + selector + " Device attributes"

	// Get the device attributes
	// This is needed to get the vendor and serial number
	ddAttributes, err := utils.UnmarshalToPointer[DriveDeviceAttributes](responseData, key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal device attributes")
	}

	size, err := utils.ConvertSizeBytes(pd.Size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert size")
	}

	jbod := strings.Contains(strings.ToUpper(pd.Type), "JBOD")

	slot, err := validateID(metadata.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate slot IDs")
	}

	physicalDrive := &physicaldrive.PhysicalDrive{
		Metadata: metadata,
		Slot:     slot,
		Vendor:   strings.TrimSpace(ddAttributes.ManufacturerID),
		Model:    strings.TrimSpace(pd.Model),
		Serial:   strings.TrimSpace(ddAttributes.SerialNumber),
		Size:     size,
		Type:     pd.DiskType(),
		Status:   pd.PDStatus(),
		JBOD:     jbod,
		// TODO : fill those fields
		// PermanentPath: "",
		// DevicePath:    "",
		// Reason:   "",
	}

	return physicalDrive, nil
}

// DiskType returns the disk type of a physical drive.
func (pd *PD) DiskType() physicaldrive.DiskType {
	// diskTypeMap maps the disk type string to the physical drive disk type.
	diskTypeMap := map[string]physicaldrive.DiskType{
		"HDD":  physicaldrive.DiskTypeHDD,
		"SSD":  physicaldrive.DiskTypeSSD,
		"NVME": physicaldrive.DiskTypeNVMe,
	}

	if dt, ok := diskTypeMap[strings.ToUpper(pd.MediaType)]; ok {
		return dt
	}

	return physicaldrive.DiskTypeUnknown
}

// convertPVStatus converts a string to a PVStatus.
func (pd *PD) PDStatus() physicaldrive.PDStatus {
	// pdStatusMap maps the physical drive status string to the physical drive status.
	pdStatusMap := map[string]physicaldrive.PDStatus{
		"Onln": physicaldrive.PDStatusUsed,
		// TODO : check the real values
		"UGood": physicaldrive.PDStatusUnassignedGood,
		// TODO : add reason for those statuses
		"UBad":   physicaldrive.PDStatusUnassignedBad,
		"Failed": physicaldrive.PDStatusFailed,
	}

	if status, ok := pdStatusMap[pd.State]; ok {
		return status
	}

	return physicaldrive.PDStatusUnknown
}

// validateID validates the slot IDs of a physical drive.
func validateID(s string) (*physicaldrive.Slot, error) {
	slot, err := physicaldrive.ParseSlot(s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ID")
	}

	bayID, err := strconv.Atoi(slot.Bay)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert bay ID to int: %s", slot.Bay)
	}

	if bayID < 0 {
		return nil, errors.Wrapf(err, "invalid bay ID: %s", slot.Bay)
	}

	if slot.Enclosure != "" {
		enclosureID, err := strconv.Atoi(slot.Enclosure)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert enclosure ID to int: %s", slot.Enclosure)
		}

		if enclosureID < 0 {
			return nil, errors.Wrapf(err, "invalid enclosure ID: %s", slot.Enclosure)
		}
	}

	return slot, nil
}

// selectorPD returns the selector for a physical drive metadata.
func selectorPD(m *physicaldrive.Metadata) (string, error) {
	slot, err := validateID(m.ID)
	if err != nil {
		return "", errors.Wrap(err, "failed to validate slot IDs")
	}

	selector := fmt.Sprintf(patternNoEnclosure, m.CtrlMetadata.ID, slot.Bay)

	if slot.Enclosure != "" {
		selector = fmt.Sprintf(patternEnclosure, m.CtrlMetadata.ID, slot.Enclosure, slot.Bay)
	}

	return selector, nil
}

// blink starts or stops the blinking of the given physical drive.
// action is either "start" or "stop".
func (a *Adapter) blink(
	metadata *physicaldrive.Metadata, action string,
) error {
	selector, err := selectorPD(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to get selector")
	}

	_, err = a.runner.Run([]string{selector, action, "locate"})
	if err != nil {
		return errors.Wrap(err, ErrCommandFailed.Error())
	}

	return nil
}

// setJBOD sets or deletes JBOD for the given physical drive.
// action is either "set" or "delete".
func (a *Adapter) setJBOD(
	metadata *physicaldrive.Metadata, action string,
) error {
	selector, err := selectorPD(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to get selector")
	}

	_, err = a.runner.Run([]string{selector, action, "jbod"})
	if err != nil {
		return errors.Wrap(err, ErrCommandFailed.Error())
	}

	return nil
}
