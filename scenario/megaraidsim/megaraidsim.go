// Package megaraidsim is the scenario backend for MegaRAID controllers driven
// by storcli64/perccli64: it answers the storcli commands of the raidmgmt
// megaraid adapter from a real capture with the changes of a phase applied,
// and makes the host show the /dev/disk/by-id links of the exposed volumes.
//
// A capture is a directory under captures/ holding one JSON file per storcli
// command, named after it: "show all" -> show_all.json, "/c0 show all" ->
// c0.json, "/c0/e251/s3 show all" -> c0_e251_s3.json, "/c0/v237 show all" ->
// c0_v237.json.
package megaraidsim

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/implementation/raidcontroller/megaraid"
	"github.com/scality/raidmgmt/scenario"
)

//go:embed captures
var captures embed.FS //nolint:gochecknoglobals // embedded captures

const (
	byIDPrefix = "/dev/disk/by-id/wwn-0x"
	yes, no    = "Yes", "No"
)

var errVolumeDeleted = errors.New("virtual drive does not exist")

type (
	// Controller replays one phase of a scenario. It implements
	// megaraid.Runner.
	Controller struct {
		capture string
		drives  map[string]*drive  // by "enclosure:slot"
		volumes map[string]*volume // by virtual drive ID
	}

	drive struct {
		state, serial, wwn string
		deviceID           int
		group              any
	}

	volume struct {
		state, device, wwn string
		exposed, deleted   bool
	}

	// doc is a storcli JSON output.
	doc = map[string]any
)

var _ megaraid.Runner = &Controller{}

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

// Run answers a storcli command.
func (c *Controller) Run(args []string) (*megaraid.CmdOutput, error) {
	cmd := strings.Join(args, " ")

	selector, ok := strings.CutSuffix(cmd, " show all")
	if cmd != "show all" && !ok {
		return nil, errors.Errorf("storcli %s: not replayed", cmd)
	}

	out, err := c.render(selector)
	if err != nil {
		return nil, errors.Wrapf(err, "storcli %s", cmd)
	}

	data, err := json.Marshal(out)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal output")
	}

	var parsed megaraid.CmdOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal output")
	}

	return &parsed, nil
}

// Links returns the /dev/disk/by-id links of the exposed volumes, with the
// block device each one points to.
func (c *Controller) Links() map[string]string {
	links := map[string]string{}

	for _, v := range c.volumes {
		if !v.deleted && v.exposed && v.wwn != "" {
			links[byIDPrefix+v.wwn] = v.device
		}
	}

	return links
}

// hostMu serializes UseHost: the megaraid package reads the host through
// package-level hooks, so only one phase can use them at a time.
var hostMu sync.Mutex //nolint:gochecknoglobals // guards package-level hooks

// UseHost makes the raidmgmt megaraid package see the links of this phase
// instead of the ones of the host, and returns a function restoring the host.
// The megaraid package reads the host through package-level hooks: UseHost
// blocks until the previous phase has called its restore function, so phases
// running in parallel are played one after the other.
func (c *Controller) UseHost() (restore func()) {
	hostMu.Lock()

	links := c.Links()
	fileExists, evalSymlinks := megaraid.CustomFileExists, megaraid.CustomEvalSymlinks

	megaraid.CustomFileExists = func(p string) bool {
		_, ok := links[p]

		return ok
	}
	megaraid.CustomEvalSymlinks = func(p string) (string, error) {
		if dev, ok := links[p]; ok {
			return dev, nil
		}

		return "", os.ErrNotExist
	}

	return func() {
		megaraid.CustomFileExists, megaraid.CustomEvalSymlinks = fileExists, evalSymlinks

		hostMu.Unlock()
	}
}

func (c *Controller) loadCapture() error {
	ctrl, err := c.read("c0")
	if err != nil {
		return err
	}

	for _, row := range rows(responseData(ctrl), "PD LIST") {
		slot := str(row["EID:Slt"])

		pd, err := c.read(driveFile(slot))
		if err != nil {
			return err
		}

		attrs := driveAttributes(pd, slot)
		c.drives[slot] = &drive{
			state: str(row["State"]), group: row["DG"], deviceID: int(num(row["DID"])),
			serial: str(attrs["SN"]), wwn: str(attrs["WWN"]),
		}
	}

	for _, row := range rows(responseData(ctrl), "VD LIST") {
		id := volumeID(row)

		vd, err := c.read("c0_v" + id)
		if err != nil {
			return err
		}

		props := object(responseData(vd)["VD"+id+" Properties"])
		c.volumes[id] = &volume{
			state: str(row["State"]), device: str(props["OS Drive Name"]),
			wwn: str(props["SCSI NAA Id"]), exposed: str(props["Exposed to OS"]) == yes,
		}
	}

	return nil
}

func (c *Controller) apply(changes scenario.Changes) error {
	for slot, change := range changes.Drives {
		d, ok := c.drives[slot]
		if !ok {
			return errors.Errorf("unknown drive %s", slot)
		}

		// MegaRAID reports a single drive state.
		if change.Status != nil {
			return errors.Errorf("drive %s: status is not replayed for megaraid, use state", slot)
		}

		set(&d.state, change.State)
		set(&d.serial, change.Serial)
		set(&d.wwn, change.WWN)
		set(&d.deviceID, change.DeviceID)

		if change.Group != nil {
			d.group = *change.Group
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

func (c *Controller) render(selector string) (doc, error) {
	switch {
	case selector == "show all":
		return c.renderOverview()
	case selector == "/c0":
		return c.renderController()
	case strings.HasPrefix(selector, "/c0/v"):
		return c.renderVolume(strings.TrimPrefix(selector, "/c0/v"))
	case strings.HasPrefix(selector, "/c0/e"):
		return c.renderDrive(selector)
	default:
		return nil, errors.Errorf("selector %s not replayed", selector)
	}
}

func (c *Controller) renderOverview() (doc, error) {
	out, err := c.read("show_all")
	if err != nil {
		return nil, err
	}

	volumes, notOptimal, failedDrives := c.counts()

	for _, row := range rows(responseData(out), "System Overview") {
		row["VDs"], row["DGs"], row["VNOpt"], row["DNOpt"] = volumes, volumes, notOptimal, failedDrives
	}

	return out, nil
}

// counts returns the number of volumes, of volumes not optimal and of failed
// drives, as shown by the system overview.
func (c *Controller) counts() (volumes, notOptimal, failedDrives int) {
	for _, v := range c.volumes {
		if v.deleted {
			continue
		}

		volumes++

		if v.state != "Optl" {
			notOptimal++
		}
	}

	for _, d := range c.drives {
		if d.state == "Failed" {
			failedDrives++
		}
	}

	return volumes, notOptimal, failedDrives
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

func (c *Controller) renderDrive(selector string) (doc, error) {
	// "/c0/e251/s3" -> "251:3"
	parts := strings.Split(selector, "/")
	if len(parts) != 4 { //nolint:mnd // "", "c0", "eE", "sS"
		return nil, errors.Errorf("selector %s not replayed", selector)
	}

	slot := strings.TrimPrefix(parts[2], "e") + ":" + strings.TrimPrefix(parts[3], "s")

	d, ok := c.drives[slot]
	if !ok {
		return nil, errors.Errorf("unknown drive %s", slot)
	}

	out, err := c.read(driveFile(slot))
	if err != nil {
		return nil, err
	}

	for _, row := range rows(responseData(out), "Drive "+selector) {
		d.update(row)
	}

	attrs := driveAttributes(out, slot)
	attrs["SN"], attrs["WWN"] = d.serial, d.wwn

	return out, nil
}

func (c *Controller) renderVolume(id string) (doc, error) {
	v, ok := c.volumes[id]
	if !ok || v.deleted {
		return nil, errors.Wrapf(errVolumeDeleted, "VD %s", id)
	}

	out, err := c.read("c0_v" + id)
	if err != nil {
		return nil, err
	}

	data := responseData(out)

	for _, row := range rows(data, "/c0/v"+id) {
		row["State"] = v.state
	}

	for _, row := range rows(data, "PDs for VD "+id) {
		c.drives[str(row["EID:Slt"])].update(row)
	}

	props := object(data["VD"+id+" Properties"])
	props["SCSI NAA Id"] = v.wwn

	exposed := no
	if v.exposed {
		exposed = yes
		props["OS Drive Name"] = v.device
	} else {
		delete(props, "OS Drive Name")
	}

	props["Exposed to OS"], props["Is LD Ready for OS Requests"] = exposed, exposed

	return out, nil
}

// update writes the drive state into a storcli drive row.
func (d *drive) update(row doc) {
	row["State"], row["DG"], row["DID"] = d.state, d.group, d.deviceID
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

	ctrl := object(controllers[0])
	data := object(ctrl["Response Data"])

	return data
}

func driveAttributes(out doc, slot string) doc {
	sel := strings.Replace("/c0/e"+slot, ":", "/s", 1)
	details := object(responseData(out)["Drive "+sel+" - Detailed Information"])
	attrs := object(details["Drive "+sel+" Device attributes"])

	return attrs
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

// driveFile is the capture file of "/c0/eE/sS show all".
func driveFile(slot string) string {
	return "c0_e" + strings.Replace(slot, ":", "_s", 1)
}

// volumeID reads the virtual drive ID of a "DG/VD" row.
func volumeID(row doc) string {
	dgvd := str(row["DG/VD"])

	return dgvd[strings.Index(dgvd, "/")+1:]
}

func str(v any) string {
	if v == nil {
		return ""
	}

	return fmt.Sprint(v)
}

func num(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}

	return 0
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
