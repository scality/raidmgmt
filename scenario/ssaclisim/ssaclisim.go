// Package ssaclisim is the scenario backend for HPE Smart Array controllers
// driven by ssacli: it answers the ssacli commands of the raidmgmt ssacli
// getters from a real capture with the events of a scenario applied.
//
// A capture is a directory under captures/ holding the text output of each
// ssacli command, named after it with "_" for spaces and "slot=0" written
// "slot0": "controller slot=0 show config" -> controller_slot0_show_config.txt.
//
// A failed drive has status Failed, and so has its logical drive, which loses
// its "Disk Name" (it is no longer exposed to the OS). The failed drive is
// printed as a real P816i-a in Mixed mode prints one: with "Last Failure
// Reason: Unknown", "Drive Type: Unsupported for RAID/Hidden from the OS
// Drive" and "Interface Type: Unknown". A replacement drive is
// OK and unassigned, its array and logical drive gone; a new logical drive
// is OK, in the array the drive had in the capture. No ssacli output was
// captured while a drive was failed: this follows what storcli showed on
// RD-2154. The ssacli getters read volume paths from the controller output
// only, so a scenario needs no host file.
//
// A controller in Mixed mode exposes a free drive to the OS: the replacement
// drive is printed with "Drive exposed to OS: True" and its "Disk Name", as
// on a real P816i-a in Mixed mode, and LSBLK answers for that device, as an
// empty disk or, for a drive carrying data, one with a filesystem. Once a
// logical drive is created on it, the drive is hidden from the OS again.
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
	controllerFile     = "controller_slot0_show_detail"
	// mixedMode is the "Controller Mode" of a controller exposing its free
	// drives to the OS.
	mixedMode = "Mixed"
	// failed is the status of a failed drive or logical drive.
	failed = "Failed"
	// failedDriveType is the "Drive Type" of a failed drive.
	failedDriveType = "Unsupported for RAID/Hidden from the OS Drive"
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
	interfaceRegexp  = regexp.MustCompile(`(?m)^(         Interface Type: ).*$`)
	lastFailureRegex = regexp.MustCompile(`(?m)^         Last Failure Reason: .*\n`)
	uniqueIDRegexp   = regexp.MustCompile(`(?m)^(         Unique Identifier: ).*$`)
	diskNameRegexp   = regexp.MustCompile(`(?m)^         Disk Name: .*\n`)
	exposedRegexp    = regexp.MustCompile(`(?m)^(         Drive exposed to OS: ).*$`)
	mountPointsRegex = regexp.MustCompile(`(?m)^         Mount Points: .*\n`)
	// sanitizeRegexp is the line a "Disk Name" comes before in a drive
	// block, as ssacli prints it.
	sanitizeRegexp   = regexp.MustCompile(`(?m)^         Sanitize Erase Supported: `)
	configLDRegexp   = regexp.MustCompile(`(?m)^(      logicaldrive (\d+) \(.*, )[^,()]+\)$`)
	configPDRegexp   = regexp.MustCompile(`(?m)^(      physicaldrive (\S+) \(.*, )[^,()]+\)$`)
	trailingNewlines = regexp.MustCompile(`\n+$`)
)

type (
	// Controller replays a capture with events applied: SSACLI answers ssacli
	// commands, LSBLK the lsblk commands of the getters.
	Controller struct {
		capture string
		drives  map[string]*drive  // by "port:box:bay"
		volumes map[string]*volume // by logical drive ID
		// configLines are the "show config" lines of the drives, used to
		// list an unassigned drive.
		configLines map[string]string
		// mode is the "Controller Mode" of the capture.
		mode string
	}

	drive struct {
		status, serial, wwid string
		// array is the array of the drive, or "-" when it is unassigned.
		array string
		// captured is the array of the drive in the capture: a drive can only
		// leave it and come back.
		captured string
		// device is the block device of the drive while the controller
		// exposes it to the OS, and carriesData tells whether it holds data.
		device      string
		carriesData bool
	}

	volume struct {
		status, device, uniqueID string
		exposed, deleted         bool
		// array is the array of the logical drive.
		array string
	}

	// Runner answers the commands of one CLI. It implements
	// commandrunner.CommandRunner.
	Runner func(args []string) ([]byte, error)
)

var _ commandrunner.CommandRunner = Runner(nil)

// Run answers a command.
func (r Runner) Run(args []string) ([]byte, error) { return r(args) }

// New returns the controller of a capture once events happened, in order.
func New(capture string, events ...scenario.Event) (*Controller, error) {
	c := &Controller{
		capture:     path.Join("captures", capture),
		drives:      map[string]*drive{},
		volumes:     map[string]*volume{},
		configLines: map[string]string{},
	}

	if err := c.loadCapture(); err != nil {
		return nil, errors.Wrapf(err, "failed to load capture %s", capture)
	}

	for i, e := range events {
		if err := c.apply(e); err != nil {
			return nil, errors.Wrapf(err, "failed to apply event %d", i+1)
		}
	}

	return c, nil
}

// SSACLI returns the runner answering ssacli commands.
func (c *Controller) SSACLI() Runner {
	return c.runSSACLI
}

// LSBLK returns the runner answering lsblk commands. The getters only call
// lsblk for a drive exposed to the OS on its own: a free drive of a
// controller in Mixed mode.
func (c *Controller) LSBLK() Runner {
	return Runner(func(args []string) ([]byte, error) {
		if len(args) > 0 {
			if d := c.exposedDrive(args[0]); d != nil {
				fstype := ""
				if d.carriesData {
					fstype = "xfs"
				}

				return lsblkDisk(args[0], fstype), nil
			}
		}

		return nil, errors.Errorf("lsblk %s: not replayed", strings.Join(args, " "))
	})
}

// exposedDrive returns the drive exposed to the OS as a block device, if any.
func (c *Controller) exposedDrive(device string) *drive {
	for _, d := range c.drives {
		if d.device != "" && d.device == device {
			return d
		}
	}

	return nil
}

// lsblkDisk is the "lsblk --json --nodeps" output of a whole disk, with the
// given filesystem type, or none when empty.
func lsblkDisk(device, fstype string) []byte {
	fs := "null"
	if fstype != "" {
		fs = `"` + fstype + `"`
	}

	return []byte(`{"blockdevices": [{"name": "` + device + `", "rota": "1", ` +
		`"size": "6001175126016", "type": "disk", "tran": "sas", "mountpoint": null, ` +
		`"fstype": ` + fs + `, "parttype": null, "pkname": null}]}`)
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
		text, err = c.renderPhysicalDrives(text)
	case logicalDrivesFile:
		text, err = c.renderLogicalDrives(text)
	case configFile:
		text, err = c.renderConfig(text)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "ssacli %s", cmd)
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
				device: device, exposed: device != "", array: sec.array,
			}
		}
	}

	return c.loadControllerAndConfig()
}

// loadControllerAndConfig reads the controller mode and the "show config"
// lines of the drives.
func (c *Controller) loadControllerAndConfig() error {
	text, err := c.read(controllerFile)
	if err != nil {
		return err
	}

	c.mode = field(text, "Controller Mode")

	if text, err = c.read(configFile); err != nil {
		return err
	}

	for _, m := range configPDRegexp.FindAllStringSubmatch(text, -1) {
		c.configLines[m[2]] = m[0]
	}

	return nil
}

func (c *Controller) apply(e scenario.Event) error {
	d, ok := c.drives[e.Slot]
	if !ok {
		return errors.Errorf("unknown drive %s", e.Slot)
	}

	v, err := c.volumeOf(d)
	if err != nil {
		return errors.Wrapf(err, "drive %s", e.Slot)
	}

	if err := checkOrder(e, d.status == failed, v.deleted); err != nil {
		return err
	}

	if err := c.checkExposure(e); err != nil {
		return err
	}

	switch e.Kind {
	case scenario.DriveFails:
		d.status = failed
		v.status, v.exposed = failed, false
	case scenario.DriveReplaced:
		v.deleted = true
		d.status, d.serial, d.wwid, d.array = "OK", e.Serial, e.WWN, unassigned
		d.device, d.carriesData = e.Device, e.CarriesData
	case scenario.VolumeCreated:
		d.array, d.device, d.carriesData = d.captured, "", false
		v.deleted, v.status, v.exposed, v.device, v.uniqueID = false, "OK", true, e.Device, e.VolumeWWN
	default:
		return errors.Errorf("event %d not replayed", e.Kind)
	}

	return nil
}

// checkOrder rejects an event that cannot happen to a drive in its current
// state: only a drive in a logical drive can fail, only a failed drive can be
// replaced, and a logical drive can only be created once the old one is
// deleted.
func checkOrder(e scenario.Event, failed, volumeDeleted bool) error {
	switch {
	case e.Kind == scenario.DriveFails && (failed || volumeDeleted):
		return errors.Errorf("drive %s cannot fail: it has failed or is in no logical drive", e.Slot)
	case e.Kind == scenario.DriveReplaced && !failed:
		return errors.Errorf("drive %s cannot be replaced: it has not failed", e.Slot)
	case e.Kind == scenario.VolumeCreated && !volumeDeleted:
		return errors.Errorf("no logical drive can be created on drive %s: its one still exists", e.Slot)
	}

	return nil
}

// checkExposure checks that a replacement drive is exposed to the OS exactly
// when the controller exposes free drives, in Mixed mode.
func (c *Controller) checkExposure(e scenario.Event) error {
	if e.Kind != scenario.DriveReplaced {
		return nil
	}

	switch {
	case c.mode == mixedMode && e.Device == "":
		return errors.Errorf("drive %s: a controller in Mixed mode exposes a free drive to the OS, "+
			"give the device of the replacement drive", e.Slot)
	case e.Device != "" && c.exposedDrive(e.Device) != nil:
		return errors.Errorf("drive %s: device %s is already exposed for another drive", e.Slot, e.Device)
	case c.mode != mixedMode && e.Device != "":
		return errors.Errorf("drive %s: a controller in %s mode does not expose a free drive to the OS",
			e.Slot, c.mode)
	}

	return nil
}

// volumeOf returns the logical drive of the array of a drive in the capture.
// An array holding several logical drives is not replayed: which of them an
// event applies to would be a guess.
func (c *Controller) volumeOf(d *drive) (*volume, error) {
	var found []string

	for id, v := range c.volumes {
		if v.array == d.captured {
			found = append(found, id)
		}
	}

	switch len(found) {
	case 0:
		return nil, errors.New("in no logical drive of the capture")
	case 1:
		return c.volumes[found[0]], nil
	default:
		sort.Strings(found)

		return nil, errors.Errorf("array %s holds logical drives %s: not replayed",
			d.captured, strings.Join(found, ", "))
	}
}

// renderPhysicalDrives lists each drive under its array, or under
// "Unassigned" once it belongs to none.
func (c *Controller) renderPhysicalDrives(text string) (string, error) {
	header, sections := splitSections(text)

	out := []string{header}

	var spares []string

	for _, sec := range sections {
		blocks, sectionSpares, err := c.renderSectionDrives(sec)
		if err != nil {
			return "", err
		}

		spares = append(spares, sectionSpares...)

		if len(blocks) > 0 {
			out = append(out, sec.title+"\n\n"+strings.Join(blocks, "")+"\n")
		}
	}

	if len(spares) > 0 {
		out = append(out, "   Unassigned\n\n"+strings.Join(spares, "")+"\n")
	}

	return trailingNewlines.ReplaceAllString(strings.Join(out, ""), "\n"), nil
}

// renderSectionDrives renders the drive blocks of a section: the ones still
// in an array, and the unassigned ones.
func (c *Controller) renderSectionDrives(sec section) (blocks, spares []string, err error) {
	for _, m := range pdBlockRegexp.FindAllStringSubmatch(sec.body, -1) {
		d, err := c.drive(m[1])
		if err != nil {
			return nil, nil, err
		}

		block := renderDrive(m[0], d)

		if d.array == unassigned {
			spares = append(spares, block)
		} else {
			blocks = append(blocks, block)
		}
	}

	return blocks, spares, nil
}

func renderDrive(block string, d *drive) string {
	driveType := "Data Drive"
	if d.array == unassigned {
		driveType = "Unassigned Drive"
	}

	out := lastFailureRegex.ReplaceAllString(block, "")

	if d.status == failed {
		// As a real P816i-a in Mixed mode prints a failed drive.
		driveType = failedDriveType
		out = interfaceRegexp.ReplaceAllString(out, "${1}Unknown")
		out = statusRegexp.ReplaceAllString(out,
			"         Status: "+d.status+"\n         Last Failure Reason: Unknown")
	} else {
		out = statusRegexp.ReplaceAllString(out, "         Status: "+d.status)
	}

	out = serialRegexp.ReplaceAllString(out, "${1}"+d.serial)
	out = wwidRegexp.ReplaceAllString(out, "${1}"+d.wwid)
	out = driveTypeRegexp.ReplaceAllString(out, "${1}"+driveType)

	return renderExposure(out, d.device)
}

// renderExposure writes whether a drive is exposed to the OS and, when it is,
// its "Disk Name" and "Mount Points", where ssacli prints them.
func renderExposure(block, device string) string {
	out := diskNameRegexp.ReplaceAllString(block, "")
	out = mountPointsRegex.ReplaceAllString(out, "")

	if device == "" {
		return exposedRegexp.ReplaceAllString(out, "${1}False")
	}

	out = exposedRegexp.ReplaceAllString(out, "${1}True")
	lines := "         Disk Name: " + device + " \n         Mount Points: None\n"

	if loc := sanitizeRegexp.FindStringIndex(out); loc != nil {
		return out[:loc[0]] + lines + out[loc[0]:]
	}

	return strings.TrimRight(out, "\n") + "\n" + lines + "\n"
}

// renderLogicalDrives drops deleted logical drives and the disk name of the
// ones not exposed to the OS.
func (c *Controller) renderLogicalDrives(text string) (string, error) {
	header, sections := splitSections(text)

	out := []string{header}

	for _, sec := range sections {
		m := ldIDRegexp.FindStringSubmatch(sec.body)
		if m == nil {
			continue
		}

		v, err := c.volume(m[1])
		if err != nil {
			return "", err
		}

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

	return trailingNewlines.ReplaceAllString(strings.Join(out, ""), "\n"), nil
}

// renderConfig updates the status of logical and physical drives, drops the
// arrays of deleted logical drives and lists unassigned drives.
func (c *Controller) renderConfig(text string) (string, error) {
	if err := c.checkConfig(text); err != nil {
		return "", err
	}

	header, sections := splitSections(text)

	out := []string{header}

	for _, sec := range sections {
		if m := configLDRegexp.FindStringSubmatch(sec.body); m != nil && c.volumes[m[2]].deleted {
			continue
		}

		out = append(out, sec.title+c.renderConfigSection(sec.body))
	}

	spares, err := c.unassignedConfigLines()
	if err != nil {
		return "", err
	}

	if len(spares) > 0 {
		out = append(out, "\n   Unassigned\n\n"+strings.Join(spares, "\n")+"\n")
	}

	return trailingNewlines.ReplaceAllString(strings.Join(out, ""), "\n"), nil
}

// checkConfig checks that every logical and physical drive of a show config
// output is one of the capture, so that rendering it cannot fail.
func (c *Controller) checkConfig(text string) error {
	for _, m := range configLDRegexp.FindAllStringSubmatch(text, -1) {
		if _, err := c.volume(m[2]); err != nil {
			return err
		}
	}

	for _, m := range configPDRegexp.FindAllStringSubmatch(text, -1) {
		if _, err := c.drive(m[2]); err != nil {
			return err
		}
	}

	return nil
}

// renderConfigSection updates the status of the logical and physical drives
// of a show config section, and drops its unassigned drives.
func (c *Controller) renderConfigSection(body string) string {
	withLDs := configLDRegexp.ReplaceAllStringFunc(body, func(line string) string {
		m := configLDRegexp.FindStringSubmatch(line)

		return m[1] + c.volumes[m[2]].status + ")"
	})

	return configPDRegexp.ReplaceAllStringFunc(withLDs, func(line string) string {
		m := configPDRegexp.FindStringSubmatch(line)

		d := c.drives[m[2]]
		if d.array == unassigned {
			// Listed in the Unassigned section instead.
			return ""
		}

		return m[1] + d.status + ")"
	})
}

// unassignedConfigLines are the show config lines of the unassigned drives.
func (c *Controller) unassignedConfigLines() ([]string, error) {
	ids := make([]string, 0, len(c.drives))
	for id := range c.drives {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	var lines []string

	for _, id := range ids {
		if d := c.drives[id]; d.array == unassigned {
			m := configPDRegexp.FindStringSubmatch(c.configLines[id])
			if m == nil {
				return nil, errors.Errorf("drive %s is not in the captured show config", id)
			}

			lines = append(lines, m[1]+d.status+")")
		}
	}

	return lines, nil
}

// drive returns a drive of the capture. A drive missing from it means an
// inconsistent capture.
func (c *Controller) drive(id string) (*drive, error) {
	d, ok := c.drives[id]
	if !ok {
		return nil, errors.Errorf("unknown drive %s", id)
	}

	return d, nil
}

// volume returns a logical drive of the capture. A logical drive missing
// from it means an inconsistent capture.
func (c *Controller) volume(id string) (*volume, error) {
	v, ok := c.volumes[id]
	if !ok {
		return nil, errors.Errorf("unknown logical drive %s", id)
	}

	return v, nil
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
