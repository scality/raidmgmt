package megaraid

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
)

// TestGetPaths pins the device/permanent path resolution for a logical volume,
// in particular that a degraded-but-online multi-drive volume whose udev
// by-id/wwn link is missing still yields its OS device path instead of failing
// discovery for the whole controller.
func TestGetPaths(t *testing.T) {
	const wwnLink = "/dev/disk/by-id/wwn-0xabc123"

	twoDrives := []*physicaldrive.PhysicalDrive{{}, {}}

	tests := []struct {
		name          string
		vdp           *VDProperties
		pdrives       []*physicaldrive.PhysicalDrive
		fileExists    func(string) bool
		evalSymlinks  func(string) (string, error)
		wantDevice    string
		wantPermanent string
		wantErr       bool
	}{
		{
			name:          "wwn link present, os drive name reported",
			vdp:           &VDProperties{OSDriveName: "/dev/sdb", SCSINAAID: "abc123"},
			pdrives:       twoDrives,
			fileExists:    func(string) bool { return true },
			wantDevice:    "/dev/sdb",
			wantPermanent: wwnLink,
		},
		{
			name:          "wwn link present, os drive name empty, resolved via symlink",
			vdp:           &VDProperties{OSDriveName: "", SCSINAAID: "abc123"},
			pdrives:       twoDrives,
			fileExists:    func(string) bool { return true },
			evalSymlinks:  func(string) (string, error) { return "/dev/sdb", nil },
			wantDevice:    "/dev/sdb",
			wantPermanent: wwnLink,
		},
		{
			name:          "degraded multi-drive volume, wwn link missing, keeps os drive name",
			vdp:           &VDProperties{OSDriveName: "/dev/sdb", SCSINAAID: "abc123"},
			pdrives:       twoDrives,
			fileExists:    func(string) bool { return false },
			wantDevice:    "/dev/sdb",
			wantPermanent: "",
		},
		{
			name:    "multi-drive volume, no scsi naa id, keeps os drive name",
			vdp:     &VDProperties{OSDriveName: "/dev/sdb", SCSINAAID: ""},
			pdrives: twoDrives,
			fileExists: func(string) bool {
				t.Helper()
				require.Fail(t, "FileExists must not be called for an empty SCSI NAA Id")
				return false
			},
			wantDevice:    "/dev/sdb",
			wantPermanent: "",
		},
	}

	origFileExists, origEvalSymlinks := CustomFileExists, CustomEvalSymlinks
	defer func() {
		CustomFileExists = origFileExists
		CustomEvalSymlinks = origEvalSymlinks
	}()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			CustomFileExists = tc.fileExists
			CustomEvalSymlinks = origEvalSymlinks

			if tc.evalSymlinks != nil {
				CustomEvalSymlinks = tc.evalSymlinks
			}

			device, permanent, err := getPaths(tc.vdp, tc.pdrives)

			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantDevice, device)
			require.Equal(t, tc.wantPermanent, permanent)
		})
	}
}
