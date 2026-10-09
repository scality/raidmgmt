package megaraidsim_test

import (
	"strings"
	"testing"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/raidcontroller/megaraid"
	"github.com/scality/raidmgmt/scenario"
	"github.com/scality/raidmgmt/scenario/megaraidsim"
)

// TestScenarios plays every MegaRAID scenario through the raidmgmt megaraid
// adapter of this repository and checks the drives it reports, with the
// device and permanent paths of the volume holding each drive.
func TestScenarios(t *testing.T) {
	names, err := scenario.List("megaraid")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range names {
		s, err := scenario.Load(name)
		if err != nil {
			t.Fatal(err)
		}

		for i, phase := range s.Phases {
			t.Run(name+"/"+phase.Name, func(t *testing.T) {
				ctrl, err := megaraidsim.New(s, i)
				if err != nil {
					t.Fatal(err)
				}

				t.Cleanup(ctrl.UseHost())

				disks, err := reportedDisks(t, megaraid.New(ctrl))
				if err != nil {
					t.Fatal(err)
				}

				notes, err := phase.Check(disks)
				if err != nil {
					t.Fatal(err)
				}

				if len(notes) > 0 {
					t.Logf("known bug %s reproduced, differences with the expected table:\n%s",
						phase.KnownBug.Ticket, strings.Join(notes, "\n"))
				}
			})
		}
	}
}

// reportedDisks returns what the adapter reports for controller 0: each drive
// with the paths of the volume holding it. When the volumes cannot be listed,
// the drives are reported without paths, as the agent does, and the error is
// logged.
func reportedDisks(t *testing.T, adapter *megaraid.Adapter) ([]scenario.Disk, error) {
	t.Helper()

	ctrl := &raidcontroller.Metadata{ID: 0}

	pds, err := adapter.PhysicalDrives(ctrl)
	if err != nil {
		return nil, err
	}

	lvs, err := adapter.LogicalVolumes(ctrl)
	if err != nil {
		t.Logf("LogicalVolumes: %v", err)
	}

	paths := map[string][2]string{}

	for _, lv := range lvs {
		for _, pd := range lv.PDrivesMetadata {
			paths[pd.ID] = [2]string{lv.DevicePath, lv.PermanentPath}
		}
	}

	disks := make([]scenario.Disk, 0, len(pds))

	for _, pd := range pds {
		disks = append(disks, scenario.Disk{
			Slot:          pd.ID,
			Status:        pd.Status.String(),
			Serial:        pd.Serial,
			DevicePath:    paths[pd.ID][0],
			PermanentPath: paths[pd.ID][1],
		})
	}

	return disks, nil
}

func TestNewRejectsUnknownPhase(t *testing.T) {
	s, err := scenario.Load("megaraid-disk-failure")
	if err != nil {
		t.Fatal(err)
	}

	for _, phase := range []int{-1, len(s.Phases)} {
		if _, err := megaraidsim.New(s, phase); err == nil {
			t.Errorf("phase %d: want an error, got none", phase)
		}
	}
}
