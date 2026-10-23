package storcli2sim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunInconsistentCapture checks that a capture listing a drive or a
// volume the controller does not know fails with an error, not a panic.
func TestRunInconsistentCapture(t *testing.T) {
	t.Parallel()

	forgetDrive := func(c *Controller) { delete(c.drives, "306:2") }
	forgetVolume := func(c *Controller) { delete(c.volumes, "3") }

	tests := []struct {
		name    string
		forget  func(c *Controller)
		cmd     []string
		wantErr string
	}{
		{name: "drive, controller", forget: forgetDrive, cmd: []string{"/c0", "show", "all"}, wantErr: "unknown drive 306:2"},
		{name: "volume, controller", forget: forgetVolume, cmd: []string{"/c0", "show", "all"}, wantErr: "unknown virtual drive 3"},
		{name: "drive, drives", forget: forgetDrive, cmd: []string{"/c0/eall/sall", "show", "all"}, wantErr: "unknown drive 306:2"},
		{name: "drive, volumes", forget: forgetDrive, cmd: []string{"/c0/vall", "show", "all"}, wantErr: "unknown drive 306:2"},
		{name: "volume, volumes", forget: forgetVolume, cmd: []string{"/c0/vall", "show", "all"}, wantErr: "unknown virtual drive 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New("storcli2-4hdd")
			require.NoError(t, err)

			tt.forget(c)

			_, err = c.Run(tt.cmd)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
