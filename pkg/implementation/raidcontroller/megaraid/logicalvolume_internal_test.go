package megaraid

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
)

// TestGetPathsExposedToOS pins that only a volume explicitly not exposed to the
// OS skips path resolution. The volume has no by-id link and the paths of its
// drive cannot be computed on the test host, so resolving them fails.
func TestGetPathsExposedToOS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		exposedToOS string
		wantErr     bool
	}{
		{
			// The offline RAID0 volume of a failed drive.
			name:        "not exposed: empty paths, no error",
			exposedToOS: "No",
		},
		{
			name:        "exposed: paths are resolved",
			exposedToOS: "Yes",
			wantErr:     true,
		},
		{
			// Older storcli versions may not report the field.
			name:        "not reported: paths are resolved",
			exposedToOS: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vdp := &VDProperties{ExposedToOS: tt.exposedToOS, SCSINAAID: "600062b2000000000000000000000237"}
			pds := []*physicaldrive.PhysicalDrive{{WWN: "5000C50000000003"}}

			devicePath, permanentPath, err := getPaths(vdp, pds)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Empty(t, devicePath)
			require.Empty(t, permanentPath)
		})
	}
}

func TestVDLVStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state string
		want  logicalvolume.LVStatus
	}{
		{state: "Optl", want: logicalvolume.LVStatusOptimal},
		{state: "Dgrd", want: logicalvolume.LVStatusDegraded},
		{state: "OfLn", want: logicalvolume.LVStatusFailed},
		{state: "Rec", want: logicalvolume.LVStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, (&VD{State: tt.state}).LVStatus())
		})
	}
}
