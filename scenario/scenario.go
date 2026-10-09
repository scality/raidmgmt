// Package scenario replays hardware scenarios, such as a drive failing and
// being replaced, from real RAID controller output, so that what raidmgmt and
// its consumers report can be tested without the hardware.
//
// A scenario is a YAML file under scenarios/: a capture of a real controller,
// then phases that change the state of its drives and volumes and give, as a
// table, the disks expected in that phase. A backend per controller family
// (see megaraidsim) replays the capture with the changes applied.
//
// A phase can document a known bug with the table reported today because of
// it: the phase passes while that table is reported, and fails both when its
// expected table is reported (the bug is fixed, remove the knownBug block) and
// when neither is (the behaviour changed).
package scenario

import (
	"bytes"
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//go:embed scenarios/*.yaml
var scenarios embed.FS //nolint:gochecknoglobals // embedded scenario files

type (
	// Scenario is a sequence of phases played on one captured controller.
	Scenario struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		// Controller is the backend replaying the capture, e.g. "megaraid".
		Controller string `yaml:"controller"`
		// Capture names the real controller output the scenario starts from.
		Capture string  `yaml:"capture"`
		Phases  []Phase `yaml:"phases"`
	}

	// Phase is one state of the controller and the disks expected in it.
	Phase struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		// KnownBug documents a bug that makes this phase differ from Expect.
		KnownBug *KnownBug `yaml:"knownBug"`
		// Changes are applied on top of the state of the previous phase.
		Changes Changes `yaml:"changes"`
		Expect  Table   `yaml:"expect"`
	}

	// KnownBug is a bug that makes a phase report something else than its
	// expected table.
	KnownBug struct {
		Ticket string `yaml:"ticket"`
		// Reported is what is reported today because of the bug.
		Reported Table `yaml:"reported"`
	}

	// Changes to the state of the controller, by drive and by volume ID as
	// the controller names them. Values use the controller's own vocabulary.
	Changes struct {
		Drives  map[string]DriveChange  `yaml:"drives"`
		Volumes map[string]VolumeChange `yaml:"volumes"`
	}

	// DriveChange changes a drive; unset fields keep their value.
	DriveChange struct {
		State *string `yaml:"state"`
		// Status is the drive status, for controllers reporting it apart from
		// the state (storcli2: state Conf, status Online or Failed).
		Status *string `yaml:"status"`
		Serial *string `yaml:"serial"`
		WWN    *string `yaml:"wwn"`
		// DeviceID is the ID the controller gives the drive, which changes
		// when a drive is replaced.
		DeviceID *int `yaml:"deviceID"`
		// Group is the drive group of the drive, or "-" when it has none.
		Group *string `yaml:"group"`
	}

	// VolumeChange changes a volume; unset fields keep their value.
	VolumeChange struct {
		State *string `yaml:"state"`
		// Exposed tells whether the volume is a block device of the host.
		Exposed *bool `yaml:"exposed"`
		Deleted *bool `yaml:"deleted"`
		// Device is the block device of the volume, e.g. /dev/sdm.
		Device *string `yaml:"device"`
		// WWN is the volume identifier behind its /dev/disk/by-id/wwn-* link.
		WWN *string `yaml:"wwn"`
	}

	// Disk is what is reported for one physical drive.
	Disk struct {
		Slot          string
		Status        string
		Serial        string
		DevicePath    string
		PermanentPath string
	}

	// Table is the list of disks expected in a phase, written in the scenario
	// as a whitespace-separated table with a header line and "-" for empty
	// values.
	Table []Disk
)

// tableColumns are the columns of a Table, in order.
var tableColumns = []string{ //nolint:gochecknoglobals // constant list
	"slot", "status", "serial", "devicePath", "permanentPath",
}

// Load reads the scenario of the given name from scenarios/<name>.yaml.
func Load(name string) (*Scenario, error) {
	data, err := scenarios.ReadFile(path.Join("scenarios", name+".yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read scenario %s", name)
	}

	// Unknown keys are errors, so that a typo does not silently change the
	// state a phase plays.
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var s Scenario
	if err := dec.Decode(&s); err != nil {
		return nil, errors.Wrapf(err, "failed to parse scenario %s", name)
	}

	if err := s.validate(); err != nil {
		return nil, errors.Wrapf(err, "invalid scenario %s", name)
	}

	return &s, nil
}

// validate rejects phases without table: an empty table would accept an
// adapter reporting no drive at all.
func (s *Scenario) validate() error {
	if len(s.Phases) == 0 {
		return errors.New("no phase")
	}

	for _, p := range s.Phases {
		if len(p.Expect) == 0 {
			return errors.Errorf("phase %s has no expected table", p.Name)
		}

		if p.KnownBug != nil && len(p.KnownBug.Reported) == 0 {
			return errors.Errorf("phase %s: known bug %s has no reported table", p.Name, p.KnownBug.Ticket)
		}
	}

	return nil
}

// NoGroup is the group of a drive that belongs to no drive group.
const NoGroup = "-"

// NumericGroup returns a drive group as storcli prints it: a number, or the
// string "-" for a drive in no group.
func NumericGroup(group string) (any, error) {
	if group == NoGroup {
		return group, nil
	}

	n, err := strconv.Atoi(group)
	if err != nil {
		return nil, errors.Errorf("group %q is neither a number nor %q", group, NoGroup)
	}

	return n, nil
}

// List returns the names of the scenarios played on the given controller.
func List(controller string) ([]string, error) {
	files, err := scenarios.ReadDir("scenarios")
	if err != nil {
		return nil, errors.Wrap(err, "failed to list scenarios")
	}

	var names []string

	for _, f := range files {
		name := strings.TrimSuffix(f.Name(), ".yaml")

		s, err := Load(name)
		if err != nil {
			return nil, err
		}

		if s.Controller == controller {
			names = append(names, name)
		}
	}

	return names, nil
}

// ChangesUntil returns the changes of the phases up to and including phase i,
// in order, which is the state of the controller in phase i.
func (s *Scenario) ChangesUntil(i int) []Changes {
	changes := make([]Changes, 0, i+1)

	for _, p := range s.Phases[:i+1] {
		changes = append(changes, p.Changes)
	}

	return changes
}

// UnmarshalYAML reads a table written as text.
func (t *Table) UnmarshalYAML(node *yaml.Node) error {
	lines := strings.Split(strings.TrimSpace(node.Value), "\n")

	header := strings.Fields(lines[0])
	if strings.Join(header, " ") != strings.Join(tableColumns, " ") {
		return errors.Errorf("line %d: table header must be %q", node.Line+1,
			strings.Join(tableColumns, " "))
	}

	seen := map[string]bool{}

	// node.Line is the line of the key: the header is on the next line and
	// row i on the one after it.
	for i, line := range lines[1:] {
		lineNo := node.Line + i + 2 //nolint:mnd // key and header lines

		disk, err := parseRow(line)
		if err != nil {
			return errors.Wrapf(err, "line %d", lineNo)
		}

		if seen[disk.Slot] {
			return errors.Errorf("line %d: slot %s already in the table", lineNo, disk.Slot)
		}

		seen[disk.Slot] = true

		*t = append(*t, disk)
	}

	return nil
}

// parseRow reads one row of a table, "-" standing for an empty value.
func parseRow(line string) (Disk, error) {
	fields := strings.Fields(line)
	if len(fields) != len(tableColumns) {
		return Disk{}, errors.Errorf("want %d columns, got %d", len(tableColumns), len(fields))
	}

	for i := range fields {
		if fields[i] == "-" {
			fields[i] = ""
		}
	}

	return Disk{
		Slot: fields[0], Status: fields[1], Serial: fields[2],
		DevicePath: fields[3], PermanentPath: fields[4],
	}, nil
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

// Check compares the disks reported in a phase with its tables and returns an
// error when the outcome is not the expected one. Without known bug, the
// expected table must be reported. With a known bug, the table of the bug must
// be reported; the differences with the expected table are then returned as
// notes.
func (p *Phase) Check(got []Disk) (notes []string, err error) {
	diffs := p.Expect.Diff(got)

	if p.KnownBug == nil {
		if len(diffs) > 0 {
			return nil, errors.Errorf("phase %s differs from its expected table:\n%s\nreported:\n%s",
				p.Name, strings.Join(diffs, "\n"), FormatTable(got))
		}

		return nil, nil
	}

	if len(diffs) == 0 {
		return nil, errors.Errorf("phase %s reports its expected table, not the one of %s: "+
			"the bug looks fixed, remove its knownBug block", p.Name, p.KnownBug.Ticket)
	}

	if bugDiffs := p.KnownBug.Reported.Diff(got); len(bugDiffs) > 0 {
		return nil, errors.Errorf("phase %s reports neither its expected table nor the one of %s:\n"+
			"differences with the table of %s:\n%s\nreported:\n%s",
			p.Name, p.KnownBug.Ticket, p.KnownBug.Ticket, strings.Join(bugDiffs, "\n"), FormatTable(got))
	}

	return diffs, nil
}

// FormatTable writes disks as a table, in the format of the scenarios.
func FormatTable(disks []Disk) string {
	rows := make([][]string, 0, len(disks)+1)
	rows = append(rows, tableColumns)

	for _, d := range indexedSorted(disks) {
		rows = append(rows, []string{
			d.Slot, dash(d.Status), dash(d.Serial), dash(d.DevicePath), dash(d.PermanentPath),
		})
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

		lines = append(lines, strings.TrimRight(strings.Join(cells, "  "), " "))
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
				slot, f.name, dash(f.got), dash(f.want)))
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

func dash(s string) string {
	if s == "" {
		return "-"
	}

	return s
}
