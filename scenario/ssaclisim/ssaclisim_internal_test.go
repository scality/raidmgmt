package ssaclisim

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/scenario"
)

// TestEventOnArrayWithSeveralVolumes checks that an event on a drive whose
// array holds several logical drives fails, rather than applying to whichever
// of them the map iteration reaches first.
func TestEventOnArrayWithSeveralVolumes(t *testing.T) {
	t.Parallel()

	c := &Controller{
		drives: map[string]*drive{"1I:1:1": {status: "OK", array: "B", captured: "B"}},
		volumes: map[string]*volume{
			"6": {status: "OK", array: "B", exposed: true},
			"2": {status: "OK", array: "B", exposed: true},
		},
	}

	err := c.apply(scenario.Event{Kind: scenario.DriveFails, Slot: "1I:1:1"})
	require.ErrorContains(t, err, "array B holds logical drives 2, 6: not replayed")
}

// TestRunInconsistentCapture checks that a capture listing a drive or a
// logical drive the controller does not know fails with an error, not a
// panic.
func TestRunInconsistentCapture(t *testing.T) {
	t.Parallel()

	const (
		config         = "controller slot=0 show config"
		physicalDrives = "controller slot=0 physicaldrive all show detail"
		logicalDrives  = "controller slot=0 logicaldrive all show detail"
	)

	forgetDrive := func(c *Controller) { delete(c.drives, "1I:1:4") }
	forgetVolume := func(c *Controller) { delete(c.volumes, "3") }

	tests := []struct {
		name    string
		events  []scenario.Event
		forget  func(c *Controller)
		cmd     string
		wantErr string
	}{
		{name: "drive, config", forget: forgetDrive, cmd: config, wantErr: "unknown drive 1I:1:4"},
		{name: "logical drive, config", forget: forgetVolume, cmd: config, wantErr: "unknown logical drive 3"},
		{name: "drive, physical drives", forget: forgetDrive, cmd: physicalDrives, wantErr: "unknown drive 1I:1:4"},
		{name: "logical drive, logical drives", forget: forgetVolume, cmd: logicalDrives, wantErr: "unknown logical drive 3"},
		{
			name: "unassigned drive missing from config",
			events: []scenario.Event{
				{Kind: scenario.DriveFails, Slot: "1I:1:4"},
				{Kind: scenario.DriveReplaced, Slot: "1I:1:4", Serial: "S", WWN: "W", Device: "/dev/sdi"},
			},
			forget:  func(c *Controller) { delete(c.configLines, "1I:1:4") },
			cmd:     config,
			wantErr: "drive 1I:1:4 is not in the captured show config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New("ssacli-4hdd", tt.events...)
			require.NoError(t, err)

			tt.forget(c)

			_, err = c.SSACLI().Run(strings.Fields(tt.cmd))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
