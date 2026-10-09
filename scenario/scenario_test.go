package scenario_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/scality/raidmgmt/scenario"
)

const (
	phaseYAML = `
name: failed
expect: |
  slot   status  serial  devicePath  permanentPath
  251:1  Used    A       /dev/sda    /dev/disk/by-id/wwn-0x1
  251:2  Failed  B       -           -
`
	knownBugYAML = `
knownBug:
  ticket: BUG-1
  reported: |
    slot   status  serial  devicePath  permanentPath
    251:1  Used    A       -           -
    251:2  Failed  B       -           -
`
)

func loadPhase(t *testing.T, withBug bool) *scenario.Phase {
	t.Helper()

	doc := phaseYAML
	if withBug {
		doc += knownBugYAML
	}

	var p scenario.Phase
	if err := yaml.Unmarshal([]byte(doc), &p); err != nil {
		t.Fatal(err)
	}

	return &p
}

func TestPhaseCheck(t *testing.T) {
	expected := []scenario.Disk{
		{Slot: "251:1", Status: "Used", Serial: "A", DevicePath: "/dev/sda", PermanentPath: "/dev/disk/by-id/wwn-0x1"},
		{Slot: "251:2", Status: "Failed", Serial: "B"},
	}
	duplicated := []scenario.Disk{expected[0], expected[1], expected[1]}
	bug := []scenario.Disk{
		{Slot: "251:1", Status: "Used", Serial: "A"},
		{Slot: "251:2", Status: "Failed", Serial: "B"},
	}
	other := []scenario.Disk{
		{Slot: "251:1", Status: "Used", Serial: "A"},
		{Slot: "251:2", Status: "Used", Serial: "B"},
	}

	tests := []struct {
		name      string
		withBug   bool
		got       []scenario.Disk
		wantErr   string
		wantNotes int
	}{
		{name: "expected table", got: expected},
		{name: "differs", got: bug, wantErr: "251:1 devicePath: got -, want /dev/sda"},
		{name: "missing drive", got: expected[:1], wantErr: "251:2: not reported"},
		{name: "duplicate drive", got: duplicated, wantErr: "251:2: reported 2 times"},
		{name: "known bug reproduces", withBug: true, got: bug, wantNotes: 2},
		{name: "known bug fixed", withBug: true, got: expected, wantErr: "the bug looks fixed"},
		{name: "known bug changed", withBug: true, got: other, wantErr: "251:2 status: got Used, want Failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes, err := loadPhase(t, tt.withBug).Check(tt.got)

			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			case len(notes) != tt.wantNotes:
				t.Fatalf("want %d notes, got %v", tt.wantNotes, notes)
			}
		})
	}
}

func TestTableRejectsDuplicateSlots(t *testing.T) {
	var p scenario.Phase

	err := yaml.Unmarshal([]byte("expect: |\n"+
		"  slot status serial devicePath permanentPath\n"+
		"  251:1 Used A - -\n"+
		"  251:1 Used B - -\n"), &p)
	if err == nil || !strings.Contains(err.Error(), "slot 251:1 already in the table") {
		t.Fatalf("want a duplicate slot error, got %v", err)
	}
}

func TestTableErrorsNameTheirLine(t *testing.T) {
	var p scenario.Phase

	// Line 3 is the header, line 4 the bad row.
	err := yaml.Unmarshal([]byte("name: x\n"+
		"expect: |\n"+
		"  slot status serial devicePath permanentPath\n"+
		"  251:1 Used A -\n"), &p)
	if err == nil || !strings.Contains(err.Error(), "line 4:") {
		t.Fatalf("want an error on line 4, got %v", err)
	}
}

func TestTableRejectsWrongColumns(t *testing.T) {
	var p scenario.Phase

	err := yaml.Unmarshal([]byte("expect: |\n  slot status\n  251:1 Used\n"), &p)
	if err == nil || !strings.Contains(err.Error(), "table header must be") {
		t.Fatalf("want a header error, got %v", err)
	}
}
