package megaraidsim_test

import (
	"encoding/json"
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

				disks, volumesErr, err := scenario.Report(megaraid.New(ctrl), &raidcontroller.Metadata{ID: 0})
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

// TestNewRejectsUnreplayedChange checks that a change this backend does not
// replay fails instead of being silently ignored.
func TestNewRejectsUnreplayedChange(t *testing.T) {
	s, err := scenario.Load("megaraid-disk-failure")
	if err != nil {
		t.Fatal(err)
	}

	value := string("Failed")
	s.Phases[0].Changes.Drives = map[string]scenario.DriveChange{"251:3": {Status: &value}}

	if _, err := megaraidsim.New(s, 0); err == nil || !strings.Contains(err.Error(), "not replayed") {
		t.Fatalf("want a not replayed error, got %v", err)
	}
}

// TestDriveGroupIsNumeric checks that a drive group is written as storcli
// prints it: a number, or "-" for an unconfigured drive.
func TestDriveGroupIsNumeric(t *testing.T) {
	s, err := scenario.Load("megaraid-disk-failure")
	if err != nil {
		t.Fatal(err)
	}

	for phase, want := range map[int]any{2: "-", 3: float64(2)} {
		ctrl, err := megaraidsim.New(s, phase)
		if err != nil {
			t.Fatal(err)
		}

		out, err := ctrl.Run([]string{"/c0/e251/s3", "show", "all"})
		if err != nil {
			t.Fatal(err)
		}

		var data struct {
			Drive []map[string]any `json:"Drive /c0/e251/s3"`
		}
		if err := json.Unmarshal(out.Controllers[0].ResponseData, &data); err != nil {
			t.Fatal(err)
		}

		if got := data.Drive[0]["DG"]; got != want {
			t.Errorf("%s: DG is %#v, want %#v", s.Phases[phase].Name, got, want)
		}
	}

	group := "two"
	s.Phases[0].Changes.Drives = map[string]scenario.DriveChange{"251:3": {Group: &group}}

	if _, err := megaraidsim.New(s, 0); err == nil {
		t.Fatal("want an error for a group that is not a number")
	}
}
