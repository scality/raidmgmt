// Package scenario replays hardware scenarios, such as a drive failing and
// being replaced, from real RAID controller output, so that what raidmgmt
// reports can be tested without the hardware.
//
// Scenarios are Gherkin features under features/, tagged with the controller
// family playing them. Each scenario starts from a capture of a real
// controller, plays events on its drives (a drive fails, is replaced, gets a
// new volume) and gives, as a table, the disks expected. A backend per
// controller family (see megaraidsim) replays the capture with the events
// applied, as that controller shows them; Run plays the features through it.
//
// An expected table can be followed by a known bug step giving the table
// reported today because of it: the step passes while that table is
// reported, and fails both when the expected table is reported (the bug is
// fixed, remove the known bug step) and when neither is (the behaviour
// changed).
package scenario

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type (
	// Expectation is the table of disks expected after some events.
	Expectation struct {
		Expect Table
		// KnownBug documents a bug that makes what is reported differ from
		// Expect.
		KnownBug *KnownBug
	}

	// KnownBug is a bug that makes an expectation report something else than
	// its expected table.
	KnownBug struct {
		Ticket string
		// Reported is what is reported today because of the bug.
		Reported Table
	}

	// Event is something happening to a drive. A backend applies it as its
	// controller shows it, in the controller's own vocabulary.
	Event struct {
		Kind EventKind
		// Slot is the drive, as the controller names it.
		Slot string
		// Serial and WWN are the ones of the new drive, for DriveReplaced.
		Serial, WWN string
		// CarriesData is set, for DriveReplaced, when the new drive already
		// carries data, such as a filesystem.
		CarriesData bool
		// Device is the block device of the new drive when the controller
		// exposes it to the OS, for DriveReplaced, or the block device of the
		// new volume, for VolumeCreated. VolumeWWN is the identifier behind
		// the /dev/disk/by-id/wwn-* link of the new volume.
		Device, VolumeWWN string
	}

	// EventKind is what happens to a drive.
	EventKind int

	// Disk is what is reported for one physical drive.
	Disk struct {
		Slot          string
		Status        string
		Serial        string
		DevicePath    string
		PermanentPath string
	}

	// Table is a list of disks, written in a feature as a data table with
	// the columns of tableColumns. An empty cell is an empty value.
	Table []Disk
)

const (
	// DriveFails: the drive fails and the volume holding it goes offline,
	// no longer a block device of the host.
	DriveFails EventKind = iota + 1
	// DriveReplaced: the volume of the drive is deleted and a new drive,
	// in no volume, is inserted in its slot. A controller that exposes free
	// drives to the OS shows it as Device.
	DriveReplaced
	// VolumeCreated: the volume the drive had in the capture is created
	// again on it.
	VolumeCreated
)

// tableColumns are the columns of a Table, in order.
var tableColumns = []string{ //nolint:gochecknoglobals // constant list
	"slot", "status", "serial", "devicePath", "permanentPath",
}

// ParseTable reads a table of disks from the rows of a data table, the first
// one being the header. A table without disk is rejected: it would accept an
// adapter reporting no drive at all.
func ParseTable(rows [][]string) (Table, error) {
	if len(rows) == 0 || strings.Join(rows[0], " ") != strings.Join(tableColumns, " ") {
		return nil, errors.Errorf("table header must be | %s |", strings.Join(tableColumns, " | "))
	}

	if len(rows) == 1 {
		return nil, errors.New("table has no disk")
	}

	table := make(Table, 0, len(rows)-1)
	seen := map[string]bool{}

	for _, row := range rows[1:] {
		disk := Disk{
			Slot: row[0], Status: row[1], Serial: row[2],
			DevicePath: row[3], PermanentPath: row[4],
		}

		if disk.Slot == "" {
			return nil, errors.New("row without slot")
		}

		if seen[disk.Slot] {
			return nil, errors.Errorf("slot %s already in the table", disk.Slot)
		}

		seen[disk.Slot] = true

		table = append(table, disk)
	}

	return table, nil
}

// Diff lists the differences between the disks of a table and the disks
// reported, one line per field, ordered by slot.
func (t Table) Diff(got []Disk) []string {
	want := indexBySlot(t)
	reported := indexBySlot(got)

	var diffs []string

	// indexBySlot keeps one disk per slot: a drive reported twice must show.
	counts := map[string]int{}
	for _, d := range got {
		counts[d.Slot]++
	}

	for _, slot := range sortedSlots(want, reported) {
		if counts[slot] > 1 {
			diffs = append(diffs, fmt.Sprintf("%s: reported %d times", slot, counts[slot]))
		}

		w, inWant := want[slot]
		g, inGot := reported[slot]

		switch {
		case !inGot:
			diffs = append(diffs, slot+": not reported")
		case !inWant:
			diffs = append(diffs, slot+": reported but not in the table")
		default:
			diffs = append(diffs, diffDisk(slot, w, g)...)
		}
	}

	return diffs
}

// Check compares the disks reported with the tables of an expectation and
// returns an error when the outcome is not the expected one. Without known
// bug, the expected table must be reported. With a known bug, the table of
// the bug must be reported; the differences with the expected table are then
// returned as notes.
func (e *Expectation) Check(got []Disk) (notes []string, err error) {
	diffs := e.Expect.Diff(got)

	if e.KnownBug == nil {
		if len(diffs) > 0 {
			return nil, errors.Errorf("the disks reported differ from the expected table:\n%s\n"+
				"reported:\n%s", strings.Join(diffs, "\n"), FormatTable(got))
		}

		return nil, nil
	}

	if len(diffs) == 0 {
		return nil, errors.Errorf("the expected table is reported, not the one of %s: "+
			"the bug looks fixed, remove its known bug step", e.KnownBug.Ticket)
	}

	if bugDiffs := e.KnownBug.Reported.Diff(got); len(bugDiffs) > 0 {
		return nil, errors.Errorf("the disks reported match neither the expected table "+
			"nor the one of %s:\ndifferences with the table of %s:\n%s\nreported:\n%s",
			e.KnownBug.Ticket, e.KnownBug.Ticket, strings.Join(bugDiffs, "\n"), FormatTable(got))
	}

	return diffs, nil
}

// FormatTable writes disks as a data table, ready to paste in a feature.
func FormatTable(disks []Disk) string {
	rows := make([][]string, 0, len(disks)+1)
	rows = append(rows, tableColumns)

	for _, d := range indexedSorted(disks) {
		rows = append(rows, []string{d.Slot, d.Status, d.Serial, d.DevicePath, d.PermanentPath})
	}

	widths := make([]int, len(tableColumns))

	for _, row := range rows {
		for i, cell := range row {
			widths[i] = max(widths[i], len(cell))
		}
	}

	lines := make([]string, 0, len(rows))

	for _, row := range rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = fmt.Sprintf("%-*s", widths[i], cell)
		}

		lines = append(lines, "| "+strings.Join(cells, " | ")+" |")
	}

	return strings.Join(lines, "\n") + "\n"
}

func diffDisk(slot string, want, got Disk) []string {
	var diffs []string

	for _, f := range []struct{ name, want, got string }{
		{"status", want.Status, got.Status},
		{"serial", want.Serial, got.Serial},
		{"devicePath", want.DevicePath, got.DevicePath},
		{"permanentPath", want.PermanentPath, got.PermanentPath},
	} {
		if f.want != f.got {
			diffs = append(diffs, fmt.Sprintf("%s %s: got %s, want %s",
				slot, f.name, quote(f.got), quote(f.want)))
		}
	}

	return diffs
}

func indexBySlot(disks []Disk) map[string]Disk {
	bySlot := make(map[string]Disk, len(disks))
	for _, d := range disks {
		bySlot[d.Slot] = d
	}

	return bySlot
}

func sortedSlots(maps ...map[string]Disk) []string {
	seen := map[string]bool{}

	var slots []string

	for _, m := range maps {
		for slot := range m {
			if !seen[slot] {
				seen[slot] = true

				slots = append(slots, slot)
			}
		}
	}

	sort.Slice(slots, func(i, j int) bool { return slotLess(slots[i], slots[j]) })

	return slots
}

func indexedSorted(disks []Disk) []Disk {
	sorted := append([]Disk(nil), disks...)
	sort.Slice(sorted, func(i, j int) bool { return slotLess(sorted[i].Slot, sorted[j].Slot) })

	return sorted
}

// slotLess orders slots such as "251:2" and "251:10" numerically per part.
func slotLess(a, b string) bool {
	pa, pb := strings.Split(a, ":"), strings.Split(b, ":")

	for i := 0; i < len(pa) && i < len(pb); i++ {
		if pa[i] != pb[i] {
			if len(pa[i]) != len(pb[i]) {
				return len(pa[i]) < len(pb[i])
			}

			return pa[i] < pb[i]
		}
	}

	return len(pa) < len(pb)
}

// quote shows an empty value as "" in a difference.
func quote(s string) string {
	if s == "" {
		return `""`
	}

	return s
}
