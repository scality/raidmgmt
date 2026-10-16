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
	replaced := scenario.Event{Kind: scenario.DriveReplaced, Slot: "1I:1:4", Serial: "S", WWN: "W"}
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
		scenario.Event{Kind: scenario.DriveReplaced, Slot: "1I:1:4", Serial: "S", WWN: "W"},
	)
	require.NoError(t, err)

	out, err := ctrl.SSACLI().Run([]string{"controller", "slot=0", "show", "config"})
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(out), "physicaldrive 1I:1:4 "), "1I:1:4 listings in:\n%s", out)
}
