package megaraidsim_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/raidcontroller/megaraid"
	"github.com/scality/raidmgmt/scenario"
	"github.com/scality/raidmgmt/scenario/megaraidsim"
)

const capture = "megaraid9560-4hdd"

// TestScenarios plays every @megaraid scenario through the raidmgmt megaraid
// adapter of this repository and checks the drives it reports, with the
// device and permanent paths of the volume holding each drive.
func TestScenarios(t *testing.T) {
	scenario.Run(t, "megaraid", func(ctx context.Context, capture string, events []scenario.Event) ([]scenario.Disk, error) {
		ctrl, err := megaraidsim.New(capture, events...)
		if err != nil {
			return nil, err
		}

		defer ctrl.UseHost()()

		disks, volumesErr, err := scenario.Report(megaraid.New(ctrl), &raidcontroller.Metadata{ID: 0})
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

	fails := scenario.Event{Kind: scenario.DriveFails, Slot: "251:3"}
	replaced := scenario.Event{Kind: scenario.DriveReplaced, Slot: "251:3", Serial: "S", WWN: "W"}
	created := scenario.Event{Kind: scenario.VolumeCreated, Slot: "251:3", Device: "/dev/sdz", VolumeWWN: "1"}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := megaraidsim.New(tt.capture, tt.events...)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestDriveGroupIsNumeric checks that a drive group is written as storcli
// prints it: "-" for an unconfigured drive, else a number.
func TestDriveGroupIsNumeric(t *testing.T) {
	fails := scenario.Event{Kind: scenario.DriveFails, Slot: "251:3"}
	replaced := scenario.Event{Kind: scenario.DriveReplaced, Slot: "251:3", Serial: "S", WWN: "W"}
	created := scenario.Event{Kind: scenario.VolumeCreated, Slot: "251:3", Device: "/dev/sdm"}

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
			ctrl, err := megaraidsim.New(capture, tt.events...)
			require.NoError(t, err)

			out, err := ctrl.Run([]string{"/c0/e251/s3", "show", "all"})
			require.NoError(t, err)

			var data struct {
				Drive []map[string]any `json:"Drive /c0/e251/s3"`
			}
			require.NoError(t, json.Unmarshal(out.Controllers[0].ResponseData, &data))
			require.NotEmpty(t, data.Drive)
			assert.Equal(t, tt.want, data.Drive[0]["DG"])
		})
	}
}
