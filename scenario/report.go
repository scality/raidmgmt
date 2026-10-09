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
// controllers before their drives. Each drive comes with the paths of the
// volume holding it, or its own paths when it is in no volume (JBOD). When the
// volumes cannot be listed, the drives are reported without volume paths, as
// disk-management-agent does, and the error is returned as volumesErr.
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

	paths := map[string][2]string{}

	for _, lv := range lvs {
		for _, pd := range lv.PDrivesMetadata {
			paths[pd.ID] = [2]string{lv.DevicePath, lv.PermanentPath}
		}
	}

	disks = make([]Disk, 0, len(pds))

	for _, pd := range pds {
		devicePath, permanentPath := pd.DevicePath, pd.PermanentPath
		if p, ok := paths[pd.ID]; ok {
			devicePath, permanentPath = p[0], p[1]
		}

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

func hasController(ctrls []*raidcontroller.RAIDController, id int) bool {
	for _, c := range ctrls {
		if c.Metadata != nil && c.ID == id {
			return true
		}
	}

	return false
}
