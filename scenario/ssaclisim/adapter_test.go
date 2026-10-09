package ssaclisim_test

import (
	"strings"
	"testing"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/controllergetter"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
	"github.com/scality/raidmgmt/pkg/implementation/physicaldrivegetter"
	"github.com/scality/raidmgmt/scenario"
	"github.com/scality/raidmgmt/scenario/ssaclisim"
)

// inventory composes the ssacli getters as disk-management-agent does.
type inventory struct {
	controllers    *controllergetter.SSACLI
	physicalDrives *physicaldrivegetter.SSACLI
	logicalVolumes *logicalvolumegetter.SSACLI
}

func newInventory(ctrl *ssaclisim.Controller) inventory {
	return inventory{
		controllers:    &controllergetter.SSACLI{SSACLI: ctrl.SSACLI()},
		physicalDrives: &physicaldrivegetter.SSACLI{SSACLI: ctrl.SSACLI(), LSBLK: ctrl.LSBLK()},
		logicalVolumes: &logicalvolumegetter.SSACLI{SSACLI: ctrl.SSACLI(), LSBLK: ctrl.LSBLK()},
	}
}

func (i inventory) Controllers() ([]*raidcontroller.RAIDController, error) {
	return i.controllers.Controllers()
}

func (i inventory) PhysicalDrives(m *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error) {
	return i.physicalDrives.PhysicalDrives(m)
}

func (i inventory) LogicalVolumes(m *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error) {
	return i.logicalVolumes.LogicalVolumes(m)
}

// TestScenarios plays every ssacli scenario through the raidmgmt ssacli
// getters of this repository and checks the drives they report, with the
// device and permanent paths of the volume holding each drive.
func TestScenarios(t *testing.T) {
	names, err := scenario.List("ssacli")
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
				ctrl, err := ssaclisim.New(s, i)
				if err != nil {
					t.Fatal(err)
				}

				disks, volumesErr, err := scenario.Report(newInventory(ctrl), &raidcontroller.Metadata{ID: 0})
				if err != nil {
					t.Fatal(err)
				}

				if volumesErr != nil {
					t.Logf("LogicalVolumes: %v", volumesErr)
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

func TestNewRejectsUnknownPhase(t *testing.T) {
	s, err := scenario.Load("ssacli-disk-failure")
	if err != nil {
		t.Fatal(err)
	}

	for _, phase := range []int{-1, len(s.Phases)} {
		if _, err := ssaclisim.New(s, phase); err == nil {
			t.Errorf("phase %d: want an error, got none", phase)
		}
	}
}

// TestNewRejectsUnreplayedChange checks that a change this backend does not
// replay fails instead of being silently ignored.
func TestNewRejectsUnreplayedChange(t *testing.T) {
	deviceID, otherArray := 12, "D"

	tests := map[string]scenario.DriveChange{
		"device ID":     {DeviceID: &deviceID},
		"another array": {Group: &otherArray},
	}

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := scenario.Load("ssacli-disk-failure")
			if err != nil {
				t.Fatal(err)
			}

			s.Phases[0].Changes.Drives = map[string]scenario.DriveChange{"1I:1:4": change}

			if _, err := ssaclisim.New(s, 0); err == nil || !strings.Contains(err.Error(), "not replayed") {
				t.Fatalf("want a not replayed error, got %v", err)
			}
		})
	}
}

// TestUnassignedDriveListedOnce checks that a drive leaving its array while
// its logical drive still exists is listed only as unassigned in show config.
func TestUnassignedDriveListedOnce(t *testing.T) {
	s, err := scenario.Load("ssacli-disk-failure")
	if err != nil {
		t.Fatal(err)
	}

	unassigned := "-"
	s.Phases[0].Changes.Drives = map[string]scenario.DriveChange{"1I:1:4": {Group: &unassigned}}

	ctrl, err := ssaclisim.New(s, 0)
	if err != nil {
		t.Fatal(err)
	}

	out, err := ctrl.SSACLI().Run([]string{"controller", "slot=0", "show", "config"})
	if err != nil {
		t.Fatal(err)
	}

	if n := strings.Count(string(out), "physicaldrive 1I:1:4 "); n != 1 {
		t.Fatalf("want 1I:1:4 listed once, got %d times:\n%s", n, out)
	}
}
