// Package ssaclisim is the scenario backend for HPE Smart Array controllers
// driven by ssacli: it answers the ssacli commands of the raidmgmt ssacli
// getters from a real capture with the changes of a phase applied.
//
// A capture is a directory under captures/ holding the text output of each
// ssacli command, named after it with "_" for spaces and "slot=0" written
// "slot0": "controller slot=0 show config" -> controller_slot0_show_config.txt.
//
// In the scenarios, a drive state is its ssacli "Status" (OK, Failed), its
// group the array it belongs to ("-" for an unassigned drive), and a volume
// state the "Status" of its logical drive. A volume is exposed when ssacli
// reports its "Disk Name". The ssacli getters read volume paths from the
// controller output only, so a phase needs no host file.
package ssaclisim

import (
	"embed"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/scenario"
)

//go:embed captures
var captures embed.FS //nolint:gochecknoglobals // embedded captures

const (
	unassigned = "-"

	physicalDrivesFile = "controller_slot0_physicaldrive_all_show_detail"
	logicalDrivesFile  = "controller_slot0_logicaldrive_all_show_detail"
	configFile         = "controller_slot0_show_config"
)

//nolint:gochecknoglobals // compiled once
var (
	// sectionRegexp starts an "Array X" or "Unassigned" section of an output.
	sectionRegexp = regexp.MustCompile(`(?m)^   (Array ([A-Z]+)|Unassigned)\b.*$`)
	pdBlockRegexp = regexp.MustCompile(`(?ms)^      physicaldrive (\S+)\n.*?(?:\n\n|\z)`)
	ldIDRegexp    = regexp.MustCompile(`(?m)^      Logical Drive: (\d+)$`)
	// statusRegexp is the "Status" line of a drive or logical drive block,
	// not "MultiDomain Status" or "Drive Authentication Status".
	statusRegexp     = regexp.MustCompile(`(?m)^         Status: .*$`)
	serialRegexp     = regexp.MustCompile(`(?m)^(         Serial Number: ).*$`)
	wwidRegexp       = regexp.MustCompile(`(?m)^(         WWID: ).*$`)
	driveTypeRegexp  = regexp.MustCompile(`(?m)^(         Drive Type: ).*$`)
	uniqueIDRegexp   = regexp.MustCompile(`(?m)^(         Unique Identifier: ).*$`)
	diskNameRegexp   = regexp.MustCompile(`(?m)^         Disk Name: .*\n`)
	configLDRegexp   = regexp.MustCompile(`(?m)^(      logicaldrive (\d+) \(.*, )[^,()]+\)$`)
	configPDRegexp   = regexp.MustCompile(`(?m)^(      physicaldrive (\S+) \(.*, )[^,()]+\)$`)
	trailingNewlines = regexp.MustCompile(`\n+$`)
)

type (
	// Controller replays one phase of a scenario: SSACLI answers ssacli
	// commands, LSBLK the lsblk commands of the getters.
	Controller struct {
		capture string
		drives  map[string]*drive  // by "port:box:bay"
		volumes map[string]*volume // by logical drive ID
		// configLines are the "show config" lines of the drives, used to
		// list an unassigned drive.
		configLines map[string]string
	}

	drive struct {
		status, serial, wwid string
		// array is the array of the drive, or "-" when it is unassigned.
		array string
		// captured is the array of the drive in the capture: a drive can only
		// leave it and come back.
		captured string
	}

	volume struct {
		status, device, uniqueID string
		exposed, deleted         bool
	}

	// Runner answers the commands of one CLI. It implements
	// commandrunner.CommandRunner.
	Runner func(args []string) ([]byte, error)
)

var _ commandrunner.CommandRunner = Runner(nil)

// Run answers a command.
func (r Runner) Run(args []string) ([]byte, error) { return r(args) }

// New returns the controller of the given phase of a scenario.
func New(s *scenario.Scenario, phase int) (*Controller, error) {
	if phase < 0 || phase >= len(s.Phases) {
		return nil, errors.Errorf("scenario %s has no phase %d", s.Name, phase)
	}

	c := &Controller{
		capture:     path.Join("captures", s.Capture),
		drives:      map[string]*drive{},
		volumes:     map[string]*volume{},
		configLines: map[string]string{},
	}

	if err := c.loadCapture(); err != nil {
		return nil, errors.Wrapf(err, "failed to load capture %s", s.Capture)
	}

	for _, changes := range s.ChangesUntil(phase) {
		if err := c.apply(changes); err != nil {
			return nil, errors.Wrapf(err, "failed to apply the changes of %s", s.Phases[phase].Name)
		}
	}

	return c, nil
}

// SSACLI returns the runner answering ssacli commands.
func (c *Controller) SSACLI() Runner {
	return c.runSSACLI
}

// LSBLK returns the runner answering lsblk commands. The getters only call
// lsblk for a drive exposed to the OS on its own, which no capture has.
func (*Controller) LSBLK() Runner {
	return Runner(func(args []string) ([]byte, error) {
		return nil, errors.Errorf("lsblk %s: not replayed", strings.Join(args, " "))
	})
}

func (c *Controller) runSSACLI(args []string) ([]byte, error) {
	cmd := strings.Join(args, " ")
	name := strings.ReplaceAll(strings.ReplaceAll(cmd, "slot=", "slot"), " ", "_")

	text, err := c.read(name)
	if err != nil {
		return nil, errors.Wrapf(err, "ssacli %s: not replayed", cmd)
	}

	switch name {
	case physicalDrivesFile:
		text = c.renderPhysicalDrives(text)
	case logicalDrivesFile:
		text = c.renderLogicalDrives(text)
	case configFile:
		text = c.renderConfig(text)
	}

	return []byte(text), nil
}

func (c *Controller) loadCapture() error {
	text, err := c.read(physicalDrivesFile)
	if err != nil {
		return err
	}

	_, sections := splitSections(text)
	for _, sec := range sections {
		for _, m := range pdBlockRegexp.FindAllStringSubmatch(sec.body, -1) {
			c.drives[m[1]] = &drive{
				status: field(m[0], "Status"), serial: field(m[0], "Serial Number"),
				wwid: field(m[0], "WWID"), array: sec.array, captured: sec.array,
			}
		}
	}

	if text, err = c.read(logicalDrivesFile); err != nil {
		return err
	}

	_, sections = splitSections(text)
	for _, sec := range sections {
		if m := ldIDRegexp.FindStringSubmatch(sec.body); m != nil {
			device := field(sec.body, "Disk Name")
			c.volumes[m[1]] = &volume{
				status: field(sec.body, "Status"), uniqueID: field(sec.body, "Unique Identifier"),
				device: device, exposed: device != "",
			}
		}
	}

	if text, err = c.read(configFile); err != nil {
		return err
	}

	for _, m := range configPDRegexp.FindAllStringSubmatch(text, -1) {
		c.configLines[m[2]] = m[0]
	}

	return nil
}

func (c *Controller) apply(changes scenario.Changes) error {
	for id, change := range changes.Drives {
		if err := c.applyDrive(id, change); err != nil {
			return err
		}
	}

	for id, change := range changes.Volumes {
		v, ok := c.volumes[id]
		if !ok {
			return errors.Errorf("unknown volume %s", id)
		}

		set(&v.status, change.State)
		set(&v.exposed, change.Exposed)
		set(&v.deleted, change.Deleted)
		set(&v.device, change.Device)
		set(&v.uniqueID, change.WWN)
	}

	return nil
}

func (c *Controller) applyDrive(id string, change scenario.DriveChange) error {
	d, ok := c.drives[id]
	if !ok {
		return errors.Errorf("unknown drive %s", id)
	}

	if change.Status != nil || change.DeviceID != nil {
		return errors.Errorf("drive %s: status and deviceID are not replayed for ssacli", id)
	}

	// The captured sections are only reused: a drive can leave its array
	// (group "-") and come back, not move to another one.
	if change.Group != nil && *change.Group != unassigned && *change.Group != d.captured {
		return errors.Errorf("drive %s: group %s is not replayed for ssacli, use %s or %s",
			id, *change.Group, d.captured, unassigned)
	}

	set(&d.status, change.State)
	set(&d.serial, change.Serial)
	set(&d.wwid, change.WWN)
	set(&d.array, change.Group)

	return nil
}

// renderPhysicalDrives lists each drive under its array, or under
// "Unassigned" once it belongs to none.
func (c *Controller) renderPhysicalDrives(text string) string {
	header, sections := splitSections(text)

	out := []string{header}

	var spares []string

	for _, sec := range sections {
		var blocks []string

		for _, m := range pdBlockRegexp.FindAllStringSubmatch(sec.body, -1) {
			d := c.drives[m[1]]
			block := renderDrive(m[0], d)

			if d.array == unassigned {
				spares = append(spares, block)
			} else {
				blocks = append(blocks, block)
			}
		}

		if len(blocks) > 0 {
			out = append(out, sec.title+"\n\n"+strings.Join(blocks, "")+"\n")
		}
	}

	if len(spares) > 0 {
		out = append(out, "   Unassigned\n\n"+strings.Join(spares, "")+"\n")
	}

	return trailingNewlines.ReplaceAllString(strings.Join(out, ""), "\n")
}

func renderDrive(block string, d *drive) string {
	driveType := "Data Drive"
	if d.array == unassigned {
		driveType = "Unassigned Drive"
	}

	out := statusRegexp.ReplaceAllString(block, "         Status: "+d.status)
	out = serialRegexp.ReplaceAllString(out, "${1}"+d.serial)
	out = wwidRegexp.ReplaceAllString(out, "${1}"+d.wwid)

	return driveTypeRegexp.ReplaceAllString(out, "${1}"+driveType)
}

// renderLogicalDrives drops deleted logical drives and the disk name of the
// ones not exposed to the OS.
func (c *Controller) renderLogicalDrives(text string) string {
	header, sections := splitSections(text)

	out := []string{header}

	for _, sec := range sections {
		m := ldIDRegexp.FindStringSubmatch(sec.body)
		if m == nil {
			continue
		}

		v := c.volumes[m[1]]
		if v.deleted {
			continue
		}

		body := statusRegexp.ReplaceAllString(sec.body, "         Status: "+v.status)
		body = uniqueIDRegexp.ReplaceAllString(body, "${1}"+v.uniqueID)

		diskName := ""
		if v.exposed {
			diskName = "         Disk Name: " + v.device + " \n"
		}

		body = diskNameRegexp.ReplaceAllString(body, diskName)
		out = append(out, sec.title+body)
	}

	return trailingNewlines.ReplaceAllString(strings.Join(out, ""), "\n")
}

// renderConfig updates the status of logical and physical drives, drops the
// arrays of deleted logical drives and lists unassigned drives.
func (c *Controller) renderConfig(text string) string {
	header, sections := splitSections(text)

	out := []string{header}

	for _, sec := range sections {
		if m := configLDRegexp.FindStringSubmatch(sec.body); m != nil && c.volumes[m[2]].deleted {
			continue
		}

		body := configLDRegexp.ReplaceAllStringFunc(sec.body, func(line string) string {
			m := configLDRegexp.FindStringSubmatch(line)

			return m[1] + c.volumes[m[2]].status + ")"
		})
		body = configPDRegexp.ReplaceAllStringFunc(body, func(line string) string {
			m := configPDRegexp.FindStringSubmatch(line)

			d := c.drives[m[2]]
			if d.array == unassigned {
				// Listed in the Unassigned section below instead.
				return ""
			}

			return m[1] + d.status + ")"
		})

		out = append(out, sec.title+body)
	}

	if spares := c.unassignedConfigLines(); len(spares) > 0 {
		out = append(out, "\n   Unassigned\n\n"+strings.Join(spares, "\n")+"\n")
	}

	return trailingNewlines.ReplaceAllString(strings.Join(out, ""), "\n")
}

// unassignedConfigLines are the show config lines of the unassigned drives.
func (c *Controller) unassignedConfigLines() []string {
	ids := make([]string, 0, len(c.drives))
	for id := range c.drives {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	var lines []string

	for _, id := range ids {
		if d := c.drives[id]; d.array == unassigned {
			m := configPDRegexp.FindStringSubmatch(c.configLines[id])
			lines = append(lines, m[1]+d.status+")")
		}
	}

	return lines
}

// section is an "Array X" or "Unassigned" section of an output.
type section struct {
	title, body string
	// array is the array letter, or "-" for the unassigned section.
	array string
}

// splitSections splits an output into the text before the first section and
// its sections.
func splitSections(text string) (string, []section) {
	locs := sectionRegexp.FindAllStringSubmatchIndex(text, -1)
	if len(locs) == 0 {
		return text, nil
	}

	sections := make([]section, 0, len(locs))

	for i, loc := range locs {
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}

		array := unassigned
		if loc[4] >= 0 {
			array = text[loc[4]:loc[5]]
		}

		sections = append(sections, section{
			title: text[loc[0]:loc[1]], body: text[loc[1]:end], array: array,
		})
	}

	return text[:locs[0][0]], sections
}

// field returns the value of the first "Key: value" line of a block.
func field(block, key string) string {
	re := regexp.MustCompile(`(?m)^\s+` + regexp.QuoteMeta(key) + `: (.*?)\s*$`)

	m := re.FindStringSubmatch(block)
	if m == nil {
		return ""
	}

	return m[1]
}

func (c *Controller) read(name string) (string, error) {
	data, err := captures.ReadFile(path.Join(c.capture, name+".txt"))
	if err != nil {
		return "", errors.Wrapf(err, "no capture for %s", name)
	}

	return string(data), nil
}

func set[T any](dst, src *T) {
	if src != nil {
		*dst = *src
	}
}
