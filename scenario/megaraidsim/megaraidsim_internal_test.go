package megaraidsim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunInconsistentCapture checks that a capture listing a drive or a
// volume the controller does not know fails with an error, not a panic.
func TestRunInconsistentCapture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		forget  func(c *Controller)
		args    []string
		wantErr string
	}{
		{name: "drive", forget: func(c *Controller) { delete(c.drives, "251:3") }, args: []string{"/c0", "show", "all"}, wantErr: "unknown drive 251:3"},
		{name: "volume", forget: func(c *Controller) { delete(c.volumes, "237") }, args: []string{"/c0", "show", "all"}, wantErr: "unknown virtual drive 237"},
		{name: "drive of a volume", forget: func(c *Controller) { delete(c.drives, "251:3") }, args: []string{"/c0/v237", "show", "all"}, wantErr: "unknown drive 251:3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New("megaraid9560-4hdd")
			require.NoError(t, err)

			tt.forget(c)

			_, err = c.Run(tt.args)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
