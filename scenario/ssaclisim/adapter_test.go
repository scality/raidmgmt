package ssaclisim_test

import (
	"context"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/controllergetter"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
	"github.com/scality/raidmgmt/pkg/implementation/physicaldrivegetter"
	"github.com/scality/raidmgmt/scenario"
	"github.com/scality/raidmgmt/scenario/ssaclisim"
)

const capture = "ssacli-4hdd"

// inventory composes the ssacli getters as disk-management-agent does.
type inventory struct {
	controllers    *controllergetter.SSACLI
	physicalDrives *physicaldrivegetter.SSACLI
	logicalVolumes *logicalvolumegetter.SSACLI
}

func newInventory(ctrl *ssaclisim.Controller) inventory {
	return inventory{
		controllers:    &controllergetter.SSACLI{SSACLI: ctrl.SSACLI()},
		physicalDrives: &physicaldrivegetter.SSACLI{SSACLI: ctrl.SSACLI(), LSBLK: ctrl.LSBLK()},
		logicalVolumes: &logicalvolumegetter.SSACLI{SSACLI: ctrl.SSACLI(), LSBLK: ctrl.LSBLK()},
	}
}

func (i inventory) Controllers() ([]*raidcontroller.RAIDController, error) {
	return i.controllers.Controllers()
}

func (i inventory) PhysicalDrives(m *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error) {
	return i.physicalDrives.PhysicalDrives(m)
}

func (i inventory) LogicalVolumes(m *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error) {
	return i.logicalVolumes.LogicalVolumes(m)
}

// TestScenarios plays every @ssacli scenario through the raidmgmt ssacli
// getters of this repository and checks the drives they report, with the
// device and permanent paths of the volume holding each drive.
func TestScenarios(t *testing.T) {
	scenario.Run(t, "ssacli", func(ctx context.Context, capture string, events []scenario.Event) ([]scenario.Disk, error) {
		ctrl, err := ssaclisim.New(capture, events...)
		if err != nil {
			return nil, err
		}

		disks, volumesErr, err := scenario.Report(newInventory(ctrl), &raidcontroller.Metadata{ID: 0})
		if volumesErr != nil {
			godog.Logf(ctx, "LogicalVolumes: %v", volumesErr)
		}

		return disks, err
	})
}

// TestNewRejects checks that New fails on an unknown capture or drive, and on
// events in an order that cannot happen to a drive.
func TestNewRejects(t *testing.T) {
	t.Parallel()

	fails := scenario.Event{Kind: scenario.DriveFails, Slot: "1I:1:4"}
	replaced := scenario.Event{Kind: scenario.DriveReplaced, Slot: "1I:1:4", Serial: "S", WWN: "W", Device: "/dev/sdi"}
	unexposed := replaced
	unexposed.Device = ""
	created := scenario.Event{Kind: scenario.VolumeCreated, Slot: "1I:1:4", Device: "/dev/sdz", VolumeWWN: "1"}

	tests := []struct {
		name    string
		capture string
		events  []scenario.Event
		wantErr string
	}{
		{name: "unknown capture", capture: "no-such-capture", wantErr: "failed to load capture"},
		{name: "unknown drive", capture: capture, events: []scenario.Event{{Kind: scenario.DriveFails, Slot: "9I:9:9"}}, wantErr: "unknown drive"},
		{name: "fails twice", capture: capture, events: []scenario.Event{fails, fails}, wantErr: "cannot fail"},
		{name: "replaced while healthy", capture: capture, events: []scenario.Event{replaced}, wantErr: "cannot be replaced"},
		{name: "volume created on a failed drive", capture: capture, events: []scenario.Event{fails, created}, wantErr: "still exists"},
		{name: "replacement fails before its volume", capture: capture, events: []scenario.Event{fails, replaced, fails}, wantErr: "cannot fail"},
		{name: "replacement not exposed in Mixed mode", capture: capture, events: []scenario.Event{fails, unexposed}, wantErr: "Mixed mode exposes a free drive"},
		{
			name: "two replacements exposed as the same device", capture: capture,
			events: []scenario.Event{
				fails, replaced,
				{Kind: scenario.DriveFails, Slot: "1I:1:3"},
				{Kind: scenario.DriveReplaced, Slot: "1I:1:3", Serial: "T", WWN: "X", Device: "/dev/sdi"},
			},
			wantErr: "device /dev/sdi is already exposed for another drive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ssaclisim.New(tt.capture, tt.events...)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestReplacedDriveListedOnce checks that a replacement drive is listed only
// as unassigned in show config, not also in its former array.
func TestReplacedDriveListedOnce(t *testing.T) {
	t.Parallel()

	ctrl, err := ssaclisim.New(capture,
		scenario.Event{Kind: scenario.DriveFails, Slot: "1I:1:4"},
		scenario.Event{Kind: scenario.DriveReplaced, Slot: "1I:1:4", Serial: "S", WWN: "W", Device: "/dev/sdi"},
	)
	require.NoError(t, err)

	out, err := ctrl.SSACLI().Run([]string{"controller", "slot=0", "show", "config"})
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(out), "physicaldrive 1I:1:4 "), "1I:1:4 listings in:\n%s", out)
}

// TestReplacedDriveExposedUntilItGetsAnArray checks that, in Mixed mode, a
// replacement drive is printed exposed to the OS with its disk name, as a
// real free drive is, and hidden again once a logical drive is created on it.
func TestReplacedDriveExposedUntilItGetsAnArray(t *testing.T) {
	t.Parallel()

	fails := scenario.Event{Kind: scenario.DriveFails, Slot: "1I:1:4"}
	replaced := scenario.Event{Kind: scenario.DriveReplaced, Slot: "1I:1:4", Serial: "S", WWN: "W", Device: "/dev/sdi"}
	created := scenario.Event{Kind: scenario.VolumeCreated, Slot: "1I:1:4", Device: "/dev/sdh", VolumeWWN: "1"}

	tests := []struct {
		name        string
		events      []scenario.Event
		wantExposed string
		wantDisk    bool
	}{
		{name: "replaced", events: []scenario.Event{fails, replaced}, wantExposed: "True", wantDisk: true},
		{name: "in a new array", events: []scenario.Event{fails, replaced, created}, wantExposed: "False"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl, err := ssaclisim.New(capture, tt.events...)
			require.NoError(t, err)

			out, err := ctrl.SSACLI().Run(strings.Fields("controller slot=0 physicaldrive all show detail"))
			require.NoError(t, err)

			_, block, found := strings.Cut(string(out), "physicaldrive 1I:1:4")
			require.True(t, found)

			block, _, _ = strings.Cut(block, "\n\n")

			assert.Contains(t, block, "Drive exposed to OS: "+tt.wantExposed)
			assert.Equal(t, tt.wantDisk, strings.Contains(block, "Disk Name: /dev/sdi"), block)
		})
	}
}

// TestFailedDrivePrintedAsARealController checks that a failed drive is
// printed as a real P816i-a in Mixed mode prints one, and a healthy one as
// captured.
func TestFailedDrivePrintedAsARealController(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		events   []scenario.Event
		want     []string
		wantNone []string
	}{
		{
			name:   "failed",
			events: []scenario.Event{{Kind: scenario.DriveFails, Slot: "1I:1:4"}},
			want: []string{
				"Status: Failed\n         Last Failure Reason: Unknown\n",
				"Drive Type: Unsupported for RAID/Hidden from the OS Drive",
				"Interface Type: Unknown",
			},
		},
		{
			name:     "healthy",
			want:     []string{"Status: OK\n", "Drive Type: Data Drive", "Interface Type: SAS"},
			wantNone: []string{"Last Failure Reason"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl, err := ssaclisim.New(capture, tt.events...)
			require.NoError(t, err)

			out, err := ctrl.SSACLI().Run(strings.Fields("controller slot=0 physicaldrive all show detail"))
			require.NoError(t, err)

			_, block, found := strings.Cut(string(out), "physicaldrive 1I:1:4")
			require.True(t, found)

			block, _, _ = strings.Cut(block, "\n\n")

			for _, w := range tt.want {
				assert.Contains(t, block, w)
			}

			for _, w := range tt.wantNone {
				assert.NotContains(t, block, w)
			}
		})
	}
}
