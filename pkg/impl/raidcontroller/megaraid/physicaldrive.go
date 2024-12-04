package megaraid

import (
	"sort"
	"strconv"
	"strings"

	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

const (
	patternEnclosure   string = "/c%s/e%d/s%d"
	patternNoEnclosure string = "/c%s/s%d"
)

var diskTypeMap = map[string]physicaldrive.DiskType{
	"HDD":  physicaldrive.DiskTypeHDD,
	"SSD":  physicaldrive.DiskTypeSSD,
	"NVME": physicaldrive.DiskTypeNVMe,
}

var PDStatusMap = map[string]physicaldrive.PDStatus{
	"Onln": physicaldrive.PDStatusUsed,
	// TODO : check the real values
	"UGood":  physicaldrive.PDStatusUnassignedGood,
	"UBad":   physicaldrive.PDStatusUnassignedBad,
	"Offln":  physicaldrive.PDStatusOffline,
	"Failed": physicaldrive.PDStatusFailed,
}

// physicaldrives returns all physical drives for a given controller.
func (m *Adapter) physicaldrives(metadata *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error) {
	// Get the physical drives for the controller
	pds, err := m.ShowAllPhysicalDrives(metadata.ID)
	if err != nil {
		return nil, err
	}

	// Prepare the slice of physical drives to return
	physicalDrives := make([]*physicaldrive.PhysicalDrive, 0)

	// Get the controller
	ctrl, err := m.ControllerByID(metadata.ID)
	if err != nil {
		return nil, err
	}

	// Fill the slice of physical drives
	for _, pd := range pds {
		enclosure, slot := pd.EnclosureSlot()

		size, err := convertSizeBytes(pd.Size)
		if err != nil {
			return nil, err
		}

		ddAttributes, err := m.ShowDeviceAttributes(metadata.ID, enclosure, slot)
		if err != nil {
			return nil, err
		}

		physicalDrive := &physicaldrive.PhysicalDrive{
			Controller: ctrl,
			ID:         strconv.Itoa(pd.DeviceID),
			Model:      strings.Trim(pd.Model, " "),
			Slot: &physicaldrive.Slot{
				Enclosure: enclosure,
				// TODO Port is not available in the output of storcli
				Bay: slot,
			},
			Type:   pd.DiskType(),
			Status: pd.PDStatus(),
			Size:   size,
			Vendor: strings.Trim(ddAttributes.ManufacturerID, " "),
			Serial: strings.Trim(ddAttributes.SerialNumber, " "),
			// TODO JBOD is not available in the output of storcli
			// Get JBOD depeding on the controller capabilities
			JBOD: false,
		}

		physicalDrives = append(physicalDrives, physicalDrive)
	}

	sort.Slice(physicalDrives, func(i, j int) bool {
		a, _ := strconv.Atoi(physicalDrives[i].ID)
		b, _ := strconv.Atoi(physicalDrives[j].ID)

		return a < b
	})

	return physicalDrives, nil
}

// EnclosureSlot returns the enclosure and slot of a physical drive.
func (pd *PD) EnclosureSlot() (int, int) {
	eidSlotSplit := strings.Split(pd.EIDSlot, ":")

	// If the enclosureSlot is not in the format "enclosure:slot"
	// then the slot is the value of EIDSlot
	if len(eidSlotSplit) != 2 {
		slot, err := strconv.Atoi(pd.EIDSlot)
		if err != nil {
			slot = -1
		}

		return -1, slot
	}

	enclosure, err := strconv.Atoi(eidSlotSplit[0])
	if err != nil {
		enclosure = -1
	}

	slot, _ := strconv.Atoi(eidSlotSplit[1])

	return enclosure, slot
}

// DiskType returns the disk type of a physical drive.
func (pd *PD) DiskType() physicaldrive.DiskType {
	if dt, ok := diskTypeMap[strings.ToUpper(pd.MediaType)]; ok {
		return dt
	}

	return physicaldrive.DiskTypeUnknown
}

// convertPVStatus converts a string to a PVStatus.
func (pd *PD) PDStatus() physicaldrive.PDStatus {
	if status, ok := PDStatusMap[pd.State]; ok {
		return status
	}

	return physicaldrive.PDStatusUnknown
}
