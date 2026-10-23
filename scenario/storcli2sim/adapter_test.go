package storcli2sim_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	raidcontrollers "github.com/scality/raidmgmt/pkg/implementation/raidcontroller"
	"github.com/scality/raidmgmt/scenario"
	"github.com/scality/raidmgmt/scenario/storcli2sim"
)

const capture = "storcli2-4hdd"

// TestScenarios plays every @storcli2 scenario through the raidmgmt storcli2
// adapter of this repository and checks the drives it reports, with the
// device and permanent paths of the volume holding each drive.
func TestScenarios(t *testing.T) {
	scenario.Run(t, "storcli2", func(ctx context.Context, capture string, events []scenario.Event) ([]scenario.Disk, error) {
		ctrl, err := storcli2sim.New(capture, events...)
		if err != nil {
			return nil, err
		}

		disks, volumesErr, err := scenario.Report(raidcontrollers.NewStorCLI2(ctrl), &raidcontroller.Metadata{ID: 0})
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

	fails := scenario.Event{Kind: scenario.DriveFails, Slot: "306:2"}
	replaced := scenario.Event{Kind: scenario.DriveReplaced, Slot: "306:2", Serial: "S", WWN: "W"}
	created := scenario.Event{Kind: scenario.VolumeCreated, Slot: "306:2", Device: "/dev/sdz", VolumeWWN: "1"}

	exposedReplacement, usedReplacement := replaced, replaced
	exposedReplacement.Device = "/dev/sdy"
	usedReplacement.CarriesData = true

	tests := []struct {
		name    string
		capture string
		events  []scenario.Event
		wantErr string
	}{
		{name: "unknown capture", capture: "no-such-capture", wantErr: "failed to load capture"},
		{name: "unknown drive", capture: capture, events: []scenario.Event{{Kind: scenario.DriveFails, Slot: "9:9"}}, wantErr: "unknown drive"},
		{name: "fails twice", capture: capture, events: []scenario.Event{fails, fails}, wantErr: "cannot fail"},
		{name: "replaced while healthy", capture: capture, events: []scenario.Event{replaced}, wantErr: "cannot be replaced"},
		{name: "volume created on a failed drive", capture: capture, events: []scenario.Event{fails, created}, wantErr: "still exists"},
		{name: "replacement fails before its volume", capture: capture, events: []scenario.Event{fails, replaced, fails}, wantErr: "cannot fail"},
		{name: "replacement exposed to the OS", capture: capture, events: []scenario.Event{fails, exposedReplacement}, wantErr: "not replayed"},
		{name: "replacement carrying data", capture: capture, events: []scenario.Event{fails, usedReplacement}, wantErr: "not replayed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := storcli2sim.New(tt.capture, tt.events...)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestDriveGroupIsNumeric checks that a drive group is written as storcli2
// prints it: "-" for an unconfigured drive, else a number.
func TestDriveGroupIsNumeric(t *testing.T) {
	t.Parallel()

	fails := scenario.Event{Kind: scenario.DriveFails, Slot: "306:2"}
	replaced := scenario.Event{Kind: scenario.DriveReplaced, Slot: "306:2", Serial: "S", WWN: "W"}
	created := scenario.Event{Kind: scenario.VolumeCreated, Slot: "306:2", Device: "/dev/sdf"}

	tests := []struct {
		name   string
		events []scenario.Event
		want   any
	}{
		{name: "replaced", events: []scenario.Event{fails, replaced}, want: "-"},
		{name: "recreated", events: []scenario.Event{fails, replaced, created}, want: float64(2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl, err := storcli2sim.New(capture, tt.events...)
			require.NoError(t, err)

			out, err := ctrl.Run([]string{"/c0/eall/sall", "show", "all"})
			require.NoError(t, err)

			var doc struct {
				Controllers []struct {
					ResponseData struct {
						Drives []struct {
							Information map[string]any `json:"Drive Information"`
						} `json:"Drives List"`
					} `json:"Response Data"`
				}
			}
			require.NoError(t, json.Unmarshal(out, &doc))
			require.NotEmpty(t, doc.Controllers)

			var found bool

			for _, d := range doc.Controllers[0].ResponseData.Drives {
				if d.Information["EID:Slt"] == "306:2" {
					found = true

					assert.Equal(t, tt.want, d.Information["DG"])
				}
			}

			assert.True(t, found, "drive 306:2 not listed")
		})
	}
}
