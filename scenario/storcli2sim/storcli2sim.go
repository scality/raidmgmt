// Package storcli2sim is the scenario backend for MegaRAID controllers driven
// by storcli2/perccli2: it answers the storcli2 commands of the raidmgmt
// storcli2 adapter from a real capture with the changes of a phase applied.
//
// A capture is a directory under captures/ holding one JSON file per storcli2
// command, named after it: "show all" -> show_all.json, "/c0 show all" ->
// c0.json, "/c0 show aso" -> c0_aso.json, "/c0/eall/sall show all" ->
// c0_eall_sall.json, "/c0/vall show all" -> c0_vall.json. Drives and volumes
// are rendered with the changes of the phase; other commands are answered as
// captured.
//
// The storcli2 adapter reads the paths of a RAID volume from the controller
// output only, so a phase needs no host file. (A JBOD drive would be resolved
// on the real filesystem; captures only hold RAID volumes.)
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

const yes, no = "Yes", "No"

type (
	// Controller replays one phase of a scenario. It implements
	// commandrunner.CommandRunner.
	Controller struct {
		capture string
		drives  map[string]*drive  // by "enclosure:slot"
		volumes map[string]*volume // by virtual drive ID
	}

	drive struct {
		state, status, serial, wwn string
		group                      any
	}

	volume struct {
		state, device, wwn string
		exposed, deleted   bool
	}

	// doc is a storcli2 JSON output.
	doc = map[string]any
)

var _ commandrunner.CommandRunner = &Controller{}

// New returns the controller of the given phase of a scenario.
func New(s *scenario.Scenario, phase int) (*Controller, error) {
	if phase < 0 || phase >= len(s.Phases) {
		return nil, errors.Errorf("scenario %s has no phase %d", s.Name, phase)
	}

	c := &Controller{
		capture: path.Join("captures", s.Capture),
		drives:  map[string]*drive{},
		volumes: map[string]*volume{},
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
			state: str(info["State"]), status: str(info["Status"]), group: info["DG"],
			serial: strings.TrimSpace(str(details["Serial Number"])), wwn: str(details["WWN"]),
		}
	}

	volumes, err := c.read("c0_vall")
	if err != nil {
		return err
	}

	for _, entry := range rows(responseData(volumes), "Virtual Drives") {
		info, props := object(entry["VD Info"]), object(entry["VD Properties"])
		c.volumes[volumeID(info)] = &volume{
			state: str(info["State"]), device: str(props["OS Drive Name"]),
			wwn: str(props["SCSI NAA Id"]), exposed: str(props["Exposed to OS"]) == yes,
		}
	}

	return nil
}

func (c *Controller) apply(changes scenario.Changes) error {
	for slot, change := range changes.Drives {
		if err := c.applyDrive(slot, change); err != nil {
			return err
		}
	}

	for id, change := range changes.Volumes {
		v, ok := c.volumes[id]
		if !ok {
			return errors.Errorf("unknown volume %s", id)
		}

		set(&v.state, change.State)
		set(&v.exposed, change.Exposed)
		set(&v.deleted, change.Deleted)
		set(&v.device, change.Device)
		set(&v.wwn, change.WWN)
	}

	return nil
}

func (c *Controller) applyDrive(slot string, change scenario.DriveChange) error {
	d, ok := c.drives[slot]
	if !ok {
		return errors.Errorf("unknown drive %s", slot)
	}

	if change.DeviceID != nil {
		return errors.Errorf("drive %s: deviceID is not replayed for storcli2", slot)
	}

	set(&d.state, change.State)
	set(&d.status, change.Status)
	set(&d.serial, change.Serial)
	set(&d.wwn, change.WWN)

	if change.Group != nil {
		group, err := scenario.NumericGroup(*change.Group)
		if err != nil {
			return errors.Wrapf(err, "drive %s", slot)
		}

		d.group = group
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
		c.drives[str(row["EID:Slt"])].update(row)
	}

	var vds []any

	for _, row := range rows(data, "VD LIST") {
		if v := c.volumes[volumeID(row)]; !v.deleted {
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
		d := c.drives[str(info["EID:Slt"])]
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
		info, props := object(entry["VD Info"]), object(entry["VD Properties"])

		v := c.volumes[volumeID(info)]
		if v.deleted {
			continue
		}

		info["State"] = v.state

		for _, pd := range rows(entry, "PDs") {
			c.drives[str(pd["EID:Slt"])].update(pd)
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

		vds = append(vds, entry)
	}

	data["Virtual Drives"] = vds

	return out, nil
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

func set[T any](dst, src *T) {
	if src != nil {
		*dst = *src
	}
}
