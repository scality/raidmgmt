// Package megaraidsim is the scenario backend for MegaRAID controllers driven
// by storcli64/perccli64: it answers the storcli commands of the raidmgmt
// megaraid adapter from a real capture with the events of a scenario applied,
// and makes the host show the /dev/disk/by-id links of the exposed volumes.
//
// Events are applied as storcli showed them on a lab node (RD-2154): a
// failed drive is Failed with its serial cut to 8 characters and its volume
// OfLn, not exposed to the OS; a replacement drive is UGood, in no drive
// group, with a new device ID; a new volume is Optl and its drive Onln.
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
	// noGroup is the drive group of an unconfigured drive.
	noGroup = "-"
	// failed is the state of a failed drive.
	failed = "Failed"
	// failedSerialLength is the length of the serial of a failed drive.
	failedSerialLength = 8
)

var errVolumeDeleted = errors.New("virtual drive does not exist")

type (
	// Controller replays a capture with events applied. It implements
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
		// volume is the virtual drive holding the drive in the capture, and
		// capturedGroup its drive group there.
		volume        string
		capturedGroup any
	}

	volume struct {
		state, device, wwn string
		exposed, deleted   bool
	}

	// doc is a storcli JSON output.
	doc = map[string]any
)

var _ megaraid.Runner = &Controller{}

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
// package-level hooks, so only one controller can use them at a time.
var hostMu sync.Mutex //nolint:gochecknoglobals // guards package-level hooks

// UseHost makes the raidmgmt megaraid package see the links of this controller
// instead of the ones of the host, and returns a function restoring the host.
// The megaraid package reads the host through package-level hooks: UseHost
// blocks until the previous controller has restored the host, so scenarios
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
			state: str(row["State"]), group: row["DG"], capturedGroup: row["DG"],
			deviceID: int(num(row["DID"])), serial: str(attrs["SN"]), wwn: str(attrs["WWN"]),
		}
	}

	for _, row := range rows(responseData(ctrl), "VD LIST") {
		if err := c.loadVolume(row); err != nil {
			return err
		}
	}

	return nil
}

// loadVolume reads a virtual drive of the VD LIST and marks its drives.
func (c *Controller) loadVolume(row doc) error {
	id := volumeID(row)

	vd, err := c.read("c0_v" + id)
	if err != nil {
		return err
	}

	for _, pd := range rows(responseData(vd), "PDs for VD "+id) {
		if d, ok := c.drives[str(pd["EID:Slt"])]; ok {
			d.volume = id
		}
	}

	props := object(responseData(vd)["VD"+id+" Properties"])
	c.volumes[id] = &volume{
		state: str(row["State"]), device: str(props["OS Drive Name"]),
		wwn: str(props["SCSI NAA Id"]), exposed: str(props["Exposed to OS"]) == yes,
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

	if err := checkOrder(e, d.state == failed, v.deleted); err != nil {
		return err
	}

	switch e.Kind {
	case scenario.DriveFails:
		d.state, d.serial = failed, d.serial[:min(len(d.serial), failedSerialLength)]
		v.state, v.exposed = "OfLn", false
	case scenario.DriveReplaced:
		v.deleted = true
		d.state, d.serial, d.wwn, d.group = "UGood", e.Serial, e.WWN, noGroup
		d.deviceID = c.newDeviceID()
	case scenario.VolumeCreated:
		d.state, d.group = "Onln", d.capturedGroup
		v.deleted, v.state, v.exposed, v.device, v.wwn = false, "Optl", true, e.Device, e.VolumeWWN
	default:
		return errors.Errorf("event %d not replayed", e.Kind)
	}

	return nil
}

// checkReplacement rejects a replacement drive exposed to the OS or carrying
// data: in RAID mode, a MegaRAID controller does not expose a free drive to the
// OS, and no capture shows a drive carrying data.
func checkReplacement(e scenario.Event) error {
	if e.Kind == scenario.DriveReplaced && (e.Device != "" || e.CarriesData) {
		return errors.Errorf("drive %s: a replacement drive exposed to the OS or carrying data "+
			"is not replayed on MegaRAID", e.Slot)
	}

	return nil
}

// checkOrder rejects an event that cannot happen to a drive in its current
// state: only a drive in a volume can fail, only a failed drive can be
// replaced, and a volume can only be created once the old one is deleted.
func checkOrder(e scenario.Event, failed, volumeDeleted bool) error {
	if err := checkReplacement(e); err != nil {
		return err
	}

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

// newDeviceID is the device ID the controller gives a new drive.
func (c *Controller) newDeviceID() int {
	id := 0
	for _, d := range c.drives {
		id = max(id, d.deviceID+1)
	}

	return id
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
		if d.state == failed {
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

func (c *Controller) renderDrive(selector string) (doc, error) {
	// "/c0/e251/s3" -> "251:3"
	parts := strings.Split(selector, "/")
	if len(parts) != 4 { //nolint:mnd // "", "c0", "eE", "sS"
		return nil, errors.Errorf("selector %s not replayed", selector)
	}

	slot := strings.TrimPrefix(parts[2], "e") + ":" + strings.TrimPrefix(parts[3], "s")

	d, err := c.drive(slot)
	if err != nil {
		return nil, err
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
		d, err := c.drive(str(row["EID:Slt"]))
		if err != nil {
			return nil, err
		}

		d.update(row)
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
