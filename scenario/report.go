package scenario

import (
	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
)

// Inventory is what a RAID controller adapter reports.
type Inventory interface {
	Controllers() ([]*raidcontroller.RAIDController, error)
	PhysicalDrives(metadata *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error)
	LogicalVolumes(metadata *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error)
}

// Report returns what an adapter reports for a controller, as scenario disks.
// The controller must be listed first, as disk-management-agent discovers
// controllers before their drives. Paths are merged as disk-management-agent
// does: a drive keeps its own paths, and only the empty ones are filled from
// the first volume holding it. When the volumes cannot be listed, the drives
// are reported without volume paths, as disk-management-agent does, and the
// error is returned as volumesErr.
func Report(inv Inventory, ctrl *raidcontroller.Metadata) (disks []Disk, volumesErr, err error) {
	ctrls, err := inv.Controllers()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to list controllers")
	}

	if !hasController(ctrls, ctrl.ID) {
		return nil, nil, errors.Errorf("controller %d not listed", ctrl.ID)
	}

	pds, err := inv.PhysicalDrives(ctrl)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to list physical drives")
	}

	lvs, err := inv.LogicalVolumes(ctrl)
	if err != nil {
		volumesErr = errors.Wrap(err, "failed to list logical volumes")
	}

	volumes := firstVolumeOfDrives(lvs)
	disks = make([]Disk, 0, len(pds))

	for _, pd := range pds {
		devicePath, permanentPath := drivePaths(pd, volumes[pd.ID])

		disks = append(disks, Disk{
			Slot:          pd.ID,
			Status:        pd.Status.String(),
			Serial:        pd.Serial,
			DevicePath:    devicePath,
			PermanentPath: permanentPath,
		})
	}

	return disks, volumesErr, nil
}

// firstVolumeOfDrives returns, by drive, the first volume holding it.
func firstVolumeOfDrives(
	lvs []*logicalvolume.LogicalVolume,
) map[string]*logicalvolume.LogicalVolume {
	volumes := map[string]*logicalvolume.LogicalVolume{}

	for _, lv := range lvs {
		for _, pd := range lv.PDrivesMetadata {
			if _, ok := volumes[pd.ID]; !ok {
				volumes[pd.ID] = lv
			}
		}
	}

	return volumes
}

// drivePaths returns the paths of a drive: its own, with the empty ones taken
// from the volume holding it, if any.
func drivePaths(pd *physicaldrive.PhysicalDrive, lv *logicalvolume.LogicalVolume) (
	devicePath, permanentPath string,
) {
	devicePath, permanentPath = pd.DevicePath, pd.PermanentPath
	if lv == nil {
		return devicePath, permanentPath
	}

	if devicePath == "" {
		devicePath = lv.DevicePath
	}

	if permanentPath == "" {
		permanentPath = lv.PermanentPath
	}

	return devicePath, permanentPath
}

func hasController(ctrls []*raidcontroller.RAIDController, id int) bool {
	for _, c := range ctrls {
		if c.Metadata != nil && c.ID == id {
			return true
		}
	}

	return false
}
