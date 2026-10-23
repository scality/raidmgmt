package megaraid

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
)

// osDriveName is the OS drive name storcli reports for the test volumes.
const osDriveName = "/dev/sdj"

// TestGetPathsExposedToOS pins that only a volume explicitly not exposed to the
// OS skips path resolution. The volume has no by-id link on the test host and
// the paths of its drive cannot be computed there: an exposed volume keeps the
// OS drive name storcli reports, with no permanent path, and no error.
func TestGetPathsExposedToOS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		exposedToOS    string
		wantDevicePath string
	}{
		// The offline RAID0 volume of a failed drive.
		{name: "not exposed: no paths", exposedToOS: "No"},
		{name: "not exposed, padded: no paths", exposedToOS: " No "},
		{name: "exposed: paths are resolved", exposedToOS: "Yes", wantDevicePath: osDriveName},
		// Older storcli versions may not report the field.
		{name: "not reported: paths are resolved", exposedToOS: "", wantDevicePath: osDriveName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vdp := &VDProperties{
				ExposedToOS: tt.exposedToOS, OSDriveName: osDriveName, SCSINAAID: "600062b2000000000000000000000237",
			}
			pds := []*physicaldrive.PhysicalDrive{{WWN: "5000C50000000003"}}

			devicePath, permanentPath, err := getPaths(vdp, pds)
			require.NoError(t, err)
			require.Equal(t, tt.wantDevicePath, devicePath)
			require.Empty(t, permanentPath)
		})
	}
}

// TestGetPathsLinkMissing pins that an exposed volume whose by-id link is not
// on the host yet, before the kernel discovers it or udev settles, does not
// fail the listing of every volume of the controller: it gets no permanent
// path. Any other failure to compute the paths of its drive is still an error.
func TestGetPathsLinkMissing(t *testing.T) {
	t.Parallel()

	// A file that exists, so that ComputePaths resolves it, but is not the
	// device path of the drive: their mismatch is a real failure.
	notTheDrive := filepath.Join(t.TempDir(), "not-the-drive")
	require.NoError(t, os.WriteFile(notTheDrive, nil, 0o600))

	tests := []struct {
		name    string
		pds     []*physicaldrive.PhysicalDrive
		wantErr string
	}{
		{name: "single-drive volume, drive has no link", pds: []*physicaldrive.PhysicalDrive{{WWN: "5000C50000000003"}}},
		{name: "multi-drive volume", pds: []*physicaldrive.PhysicalDrive{{WWN: "5000C50000000003"}, {WWN: "5000C50000000004"}}},
		{
			name:    "single-drive volume, drive paths mismatch",
			pds:     []*physicaldrive.PhysicalDrive{{PermanentPath: notTheDrive, DevicePath: "/dev/sdz"}},
			wantErr: "failed to compute paths from physical drive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vdp := &VDProperties{ExposedToOS: "Yes", OSDriveName: osDriveName, SCSINAAID: "600062b2000000000000000000000237"}

			devicePath, permanentPath, err := getPaths(vdp, tt.pds)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, osDriveName, devicePath)
			require.Empty(t, permanentPath)
		})
	}
}

// TestVolumePathsOfflineVolume pins that an offline or failed volume gets
// empty paths without error whatever "Exposed to OS" says, while a healthy
// one still has its paths resolved: it keeps the OS drive name storcli
// reports, as its by-id link is not on the test host.
func TestVolumePathsOfflineVolume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		state          string
		exposedToOS    string
		wantDevicePath string
	}{
		{name: "offline, reported exposed", state: "OfLn", exposedToOS: "Yes"},
		{name: "offline, exposure not reported", state: "OfLn", exposedToOS: ""},
		{name: "failed, reported exposed", state: "Fail", exposedToOS: "Yes"},
		{name: "optimal: paths are resolved", state: "Optl", exposedToOS: "Yes", wantDevicePath: osDriveName},
		{name: "degraded: paths are resolved", state: "Dgrd", exposedToOS: "Yes", wantDevicePath: osDriveName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vdp := &VDProperties{
				ExposedToOS: tt.exposedToOS, OSDriveName: osDriveName, SCSINAAID: "600062b2000000000000000000000237",
			}
			pds := []*physicaldrive.PhysicalDrive{{WWN: "5000C50000000003"}}

			devicePath, permanentPath, err := volumePaths(&VD{State: tt.state}, vdp, pds)
			require.NoError(t, err)
			require.Equal(t, tt.wantDevicePath, devicePath)
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
