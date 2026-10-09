package storcli2sim_test

import (
	"strings"
	"testing"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	raidcontrollers "github.com/scality/raidmgmt/pkg/implementation/raidcontroller"
	"github.com/scality/raidmgmt/scenario"
	"github.com/scality/raidmgmt/scenario/storcli2sim"
)

// TestScenarios plays every storcli2 scenario through the raidmgmt storcli2
// adapter of this repository and checks the drives it reports, with the
// device and permanent paths of the volume holding each drive.
func TestScenarios(t *testing.T) {
	names, err := scenario.List("storcli2")
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
				ctrl, err := storcli2sim.New(s, i)
				if err != nil {
					t.Fatal(err)
				}

				disks, volumesErr, err := scenario.Report(raidcontrollers.NewStorCLI2(ctrl), &raidcontroller.Metadata{ID: 0})
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
	s, err := scenario.Load("storcli2-disk-failure")
	if err != nil {
		t.Fatal(err)
	}

	for _, phase := range []int{-1, len(s.Phases)} {
		if _, err := storcli2sim.New(s, phase); err == nil {
			t.Errorf("phase %d: want an error, got none", phase)
		}
	}
}

// TestNewRejectsUnreplayedChange checks that a change this backend does not
// replay fails instead of being silently ignored.
func TestNewRejectsUnreplayedChange(t *testing.T) {
	s, err := scenario.Load("storcli2-disk-failure")
	if err != nil {
		t.Fatal(err)
	}

	value := int(12)
	s.Phases[0].Changes.Drives = map[string]scenario.DriveChange{"306:2": {DeviceID: &value}}

	if _, err := storcli2sim.New(s, 0); err == nil || !strings.Contains(err.Error(), "not replayed") {
		t.Fatalf("want a not replayed error, got %v", err)
	}
}
