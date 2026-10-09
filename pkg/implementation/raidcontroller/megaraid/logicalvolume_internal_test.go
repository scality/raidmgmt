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
			name:        "not exposed, padded: empty paths, no error",
			exposedToOS: " No ",
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

// TestSelectNewLogicalVolume checks that a volume not exposed to the OS, such
// as the offline volume of a failed drive still listing a requested slot,
// does not hide the new, exposed volume.
func TestSelectNewLogicalVolume(t *testing.T) {
	t.Parallel()

	volume := func(id, permanentPath string, slots ...string) *logicalvolume.LogicalVolume {
		lv := &logicalvolume.LogicalVolume{
			Metadata:      &logicalvolume.Metadata{ID: id},
			PermanentPath: permanentPath,
		}
		for _, slot := range slots {
			lv.PDrivesMetadata = append(lv.PDrivesMetadata, &physicaldrive.Metadata{ID: slot})
		}

		return lv
	}

	offline := volume("1", "", "251:3")
	exposed := volume("5", "/dev/disk/by-id/wwn-0x5", "251:3")
	other := volume("2", "/dev/disk/by-id/wwn-0x2", "251:2")

	tests := []struct {
		name    string
		lvs     []*logicalvolume.LogicalVolume
		wantID  string
		wantErr string
	}{
		{name: "exposed after an offline volume", lvs: []*logicalvolume.LogicalVolume{other, offline, exposed}, wantID: "5"},
		{name: "exposed before an offline volume", lvs: []*logicalvolume.LogicalVolume{exposed, offline}, wantID: "5"},
		{name: "not exposed yet", lvs: []*logicalvolume.LogicalVolume{other, offline}, wantErr: "has no device yet"},
		{name: "not listed", lvs: []*logicalvolume.LogicalVolume{other}, wantErr: "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lv, err := selectNewLogicalVolume(tt.lvs, []*physicaldrive.Metadata{{ID: "251:3"}})
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantID, lv.ID)
		})
	}
}
