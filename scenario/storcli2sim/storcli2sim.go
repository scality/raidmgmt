// Package storcli2sim is the scenario backend for MegaRAID controllers driven
// by storcli2/perccli2: it answers the storcli2 commands of the raidmgmt
// storcli2 adapter from a real capture with the events of a scenario applied.
//
// storcli2 reports a drive status apart from its state. A failed drive keeps
// its state (Conf) with status Failed, and its volume is OfLn, not exposed to
// the OS; a replacement drive is UConf, status Good, in no drive group; a new
// volume is Optl and its drive Conf, status Online. No storcli2 output was
// captured while a drive was failed: this follows what storcli showed on
// RD-2154.
//
// A capture is a directory under captures/ holding one JSON file per storcli2
// command, named after it: "show all" -> show_all.json, "/c0 show all" ->
// c0.json, "/c0 show aso" -> c0_aso.json, "/c0/eall/sall show all" ->
// c0_eall_sall.json, "/c0/vall show all" -> c0_vall.json. Drives and volumes
// are rendered with the events applied; other commands are answered as
// captured.
//
// The storcli2 adapter reads the paths of a RAID volume from the controller
// output only, so a scenario needs no host file. (A JBOD drive would be
// resolved on the real filesystem; captures only hold RAID volumes.)
package storcli2sim

import (
	"embed"
	"encoding/json"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/scenario"
)

//go:embed captures
var captures embed.FS //nolint:gochecknoglobals // embedded captures

const (
	yes, no = "Yes", "No"
	// noGroup is the drive group of an unconfigured drive.
	noGroup = "-"
)

type (
	// Controller replays a capture with events applied. It implements
	// commandrunner.CommandRunner.
	Controller struct {
		capture string
		drives  map[string]*drive  // by "enclosure:slot"
		volumes map[string]*volume // by virtual drive ID
	}

	drive struct {
		state, status, serial, wwn string
		group                      any
		// volume is the virtual drive holding the drive in the capture, and
		// capturedGroup its drive group there.
		volume        string
		capturedGroup any
	}

	volume struct {
		state, device, wwn string
		exposed, deleted   bool
	}

	// doc is a storcli2 JSON output.
	doc = map[string]any
)

var _ commandrunner.CommandRunner = &Controller{}

// New returns the controller of a capture once events happened, in order.
func New(capture string, events ...scenario.Event) (*Controller, error) {
	c := &Controller{
		capture: path.Join("captures", capture),
		drives:  map[string]*drive{},
		volumes: map[string]*volume{},
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

// Run answers a storcli2 command with its JSON output.
func (c *Controller) Run(args []string) ([]byte, error) {
	var (
		out doc
		err error
	)

	cmd := strings.Join(args, " ")

	switch cmd {
	case "/c0 show all":
		out, err = c.renderController()
	case "/c0/eall/sall show all":
		out, err = c.renderDrives()
	case "/c0/vall show all":
		out, err = c.renderVolumes()
	default:
		out, err = c.read(captureName(cmd))
	}

	if err != nil {
		return nil, errors.Wrapf(err, "storcli2 %s: not replayed", cmd)
	}

	data, err := json.Marshal(out)

	return data, errors.Wrap(err, "failed to marshal output")
}

func (c *Controller) loadCapture() error {
	drives, err := c.read("c0_eall_sall")
	if err != nil {
		return err
	}

	for _, entry := range rows(responseData(drives), "Drives List") {
		info, details := object(entry["Drive Information"]), object(entry["Drive Detailed Information"])
		c.drives[str(info["EID:Slt"])] = &drive{
			state: str(info["State"]), status: str(info["Status"]),
			group: info["DG"], capturedGroup: info["DG"],
			serial: strings.TrimSpace(str(details["Serial Number"])), wwn: str(details["WWN"]),
		}
	}

	volumes, err := c.read("c0_vall")
	if err != nil {
		return err
	}

	for _, entry := range rows(responseData(volumes), "Virtual Drives") {
		info, props := object(entry["VD Info"]), object(entry["VD Properties"])

		for _, pd := range rows(entry, "PDs") {
			if d, ok := c.drives[str(pd["EID:Slt"])]; ok {
				d.volume = volumeID(info)
			}
		}

		c.volumes[volumeID(info)] = &volume{
			state: str(info["State"]), device: str(props["OS Drive Name"]),
			wwn: str(props["SCSI NAA Id"]), exposed: str(props["Exposed to OS"]) == yes,
		}
	}

	return nil
}

func (c *Controller) apply(e scenario.Event) error {
	d, ok := c.drives[e.Slot]
	if !ok {
		return errors.Errorf("unknown drive %s", e.Slot)
	}

	v, ok := c.volumes[d.volume]
	if !ok {
		return errors.Errorf("drive %s is in no volume of the capture", e.Slot)
	}

	if err := checkOrder(e, d.status == "Failed", v.deleted); err != nil {
		return err
	}

	switch e.Kind {
	case scenario.DriveFails:
		d.status = "Failed"
		v.state, v.exposed = "OfLn", false
	case scenario.DriveReplaced:
		v.deleted = true
		d.state, d.status, d.serial, d.wwn, d.group = "UConf", "Good", e.Serial, e.WWN, noGroup
	case scenario.VolumeCreated:
		d.state, d.status, d.group = "Conf", "Online", d.capturedGroup
		v.deleted, v.state, v.exposed, v.device, v.wwn = false, "Optl", true, e.Device, e.VolumeWWN
	default:
		return errors.Errorf("event %d not replayed", e.Kind)
	}

	return nil
}

// checkOrder rejects an event that cannot happen to a drive in its current
// state: only a drive in a volume can fail, only a failed drive can be
// replaced, and a volume can only be created once the old one is deleted.
func checkOrder(e scenario.Event, failed, volumeDeleted bool) error {
	switch {
	case e.Kind == scenario.DriveFails && (failed || volumeDeleted):
		return errors.Errorf("drive %s cannot fail: it has failed or is in no volume", e.Slot)
	case e.Kind == scenario.DriveReplaced && !failed:
		return errors.Errorf("drive %s cannot be replaced: it has not failed", e.Slot)
	case e.Kind == scenario.VolumeCreated && !volumeDeleted:
		return errors.Errorf("no volume can be created on drive %s: its volume still exists", e.Slot)
	}

	return nil
}

func (c *Controller) renderController() (doc, error) {
	out, err := c.read("c0")
	if err != nil {
		return nil, err
	}

	data := responseData(out)

	for _, row := range rows(data, "PD LIST") {
		d, err := c.drive(str(row["EID:Slt"]))
		if err != nil {
			return nil, err
		}

		d.update(row)
	}

	var vds []any

	for _, row := range rows(data, "VD LIST") {
		v, err := c.volume(volumeID(row))
		if err != nil {
			return nil, err
		}

		if !v.deleted {
			row["State"] = v.state
			vds = append(vds, row)
		}
	}

	data["VD LIST"] = vds

	return out, nil
}

func (c *Controller) renderDrives() (doc, error) {
	out, err := c.read("c0_eall_sall")
	if err != nil {
		return nil, err
	}

	for _, entry := range rows(responseData(out), "Drives List") {
		info, details := object(entry["Drive Information"]), object(entry["Drive Detailed Information"])

		d, err := c.drive(str(info["EID:Slt"]))
		if err != nil {
			return nil, err
		}

		d.update(info)
		details["Serial Number"], details["WWN"] = d.serial, d.wwn
	}

	return out, nil
}

func (c *Controller) renderVolumes() (doc, error) {
	out, err := c.read("c0_vall")
	if err != nil {
		return nil, err
	}

	data := responseData(out)

	var vds []any

	for _, entry := range rows(data, "Virtual Drives") {
		listed, err := c.renderVolume(entry)
		if err != nil {
			return nil, err
		}

		if listed {
			vds = append(vds, entry)
		}
	}

	data["Virtual Drives"] = vds

	return out, nil
}

// drive returns a drive of the capture. A drive missing from it means an
// inconsistent capture.
func (c *Controller) drive(slot string) (*drive, error) {
	d, ok := c.drives[slot]
	if !ok {
		return nil, errors.Errorf("unknown drive %s", slot)
	}

	return d, nil
}

// volume returns a virtual drive of the capture. A volume missing from it
// means an inconsistent capture.
func (c *Controller) volume(id string) (*volume, error) {
	v, ok := c.volumes[id]
	if !ok {
		return nil, errors.Errorf("unknown virtual drive %s", id)
	}

	return v, nil
}

// renderVolume writes the state of a volume and of its drives into its entry
// of "/c0/vall show all". It returns false for a deleted volume, which is not
// listed.
func (c *Controller) renderVolume(entry doc) (listed bool, err error) {
	info, props := object(entry["VD Info"]), object(entry["VD Properties"])

	v, err := c.volume(volumeID(info))
	if err != nil || v.deleted {
		return false, err
	}

	info["State"] = v.state

	for _, pd := range rows(entry, "PDs") {
		d, err := c.drive(str(pd["EID:Slt"]))
		if err != nil {
			return false, err
		}

		d.update(pd)
	}

	exposed := no
	if v.exposed {
		exposed = yes
		props["OS Drive Name"] = v.device
	} else {
		delete(props, "OS Drive Name")
	}

	props["Exposed to OS"], props["VD Ready for OS Requests"] = exposed, exposed
	props["SCSI NAA Id"] = v.wwn

	return true, nil
}

// update writes the drive state into a storcli2 drive row.
func (d *drive) update(row doc) {
	row["State"], row["Status"], row["DG"] = d.state, d.status, d.group
}

func (c *Controller) read(name string) (doc, error) {
	data, err := captures.ReadFile(path.Join(c.capture, name+".json"))
	if err != nil {
		return nil, errors.Wrapf(err, "no capture for %s", name)
	}

	var out doc
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, errors.Wrapf(err, "failed to parse capture %s", name)
	}

	return out, nil
}

func responseData(out doc) doc {
	controllers := array(out["Controllers"])
	if len(controllers) == 0 {
		return doc{}
	}

	return object(object(controllers[0])["Response Data"])
}

func rows(data doc, key string) []doc {
	items := array(data[key])
	out := make([]doc, 0, len(items))

	for _, item := range items {
		if row, ok := item.(doc); ok {
			out = append(out, row)
		}
	}

	return out
}

// captureName is the capture file of a command: "/c0 show aso" -> "c0_aso",
// "show all" -> "show_all".
func captureName(cmd string) string {
	if cmd == "show all" {
		return "show_all"
	}

	name := strings.TrimPrefix(cmd, "/")
	name = strings.Replace(name, " show all", "", 1)

	return strings.NewReplacer(" show ", "_", "/", "_").Replace(name)
}

// volumeID reads the virtual drive ID of a "DG/VD" field.
func volumeID(info doc) string {
	dgvd := str(info["DG/VD"])

	return dgvd[strings.Index(dgvd, "/")+1:]
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	return ""
}

// object returns v as a JSON object, or an empty one.
func object(v any) doc {
	if o, ok := v.(doc); ok {
		return o
	}

	return doc{}
}

// array returns v as a JSON array, or nil.
func array(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}

	return nil
}
