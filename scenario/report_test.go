package scenario_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/scenario"
)

// inventory is a fake adapter reporting fixed drives and volumes.
type inventory struct {
	pds        []*physicaldrive.PhysicalDrive
	lvs        []*logicalvolume.LogicalVolume
	volumesErr error
}

func (inventory) Controllers() ([]*raidcontroller.RAIDController, error) {
	return []*raidcontroller.RAIDController{{Metadata: &raidcontroller.Metadata{ID: 0}}}, nil
}

func (i inventory) PhysicalDrives(*raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error) {
	return i.pds, nil
}

func (i inventory) LogicalVolumes(*raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error) {
	return i.lvs, i.volumesErr
}

// drive is drive "1" of the fake controller.
func drive(devicePath, permanentPath string) *physicaldrive.PhysicalDrive {
	return &physicaldrive.PhysicalDrive{
		Metadata:   &physicaldrive.Metadata{ID: "1"},
		DevicePath: devicePath, PermanentPath: permanentPath,
	}
}

func volume(devicePath, permanentPath string, ids ...string) *logicalvolume.LogicalVolume {
	lv := &logicalvolume.LogicalVolume{DevicePath: devicePath, PermanentPath: permanentPath}
	for _, id := range ids {
		lv.PDrivesMetadata = append(lv.PDrivesMetadata, &physicaldrive.Metadata{ID: id})
	}

	return lv
}

// TestReportMergesPathsAsTheAgent checks that paths are merged as
// disk-management-agent does: a drive keeps its own paths, and only the empty
// ones are filled from the first volume holding it.
func TestReportMergesPathsAsTheAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		inv        inventory
		wantPaths  [2]string
		wantVolErr bool
	}{
		{
			name:      "drive in a volume, no paths of its own",
			inv:       inventory{pds: []*physicaldrive.PhysicalDrive{drive("", "")}, lvs: []*logicalvolume.LogicalVolume{volume("/dev/sdb", "/by-id/b", "1")}},
			wantPaths: [2]string{"/dev/sdb", "/by-id/b"},
		},
		{
			name:      "drive with its own paths keeps them",
			inv:       inventory{pds: []*physicaldrive.PhysicalDrive{drive("/dev/sda", "/by-id/a")}, lvs: []*logicalvolume.LogicalVolume{volume("/dev/sdb", "/by-id/b", "1")}},
			wantPaths: [2]string{"/dev/sda", "/by-id/a"},
		},
		{
			name:      "only the empty path is filled",
			inv:       inventory{pds: []*physicaldrive.PhysicalDrive{drive("/dev/sda", "")}, lvs: []*logicalvolume.LogicalVolume{volume("/dev/sdb", "/by-id/b", "1")}},
			wantPaths: [2]string{"/dev/sda", "/by-id/b"},
		},
		{
			name: "the first volume holding the drive wins",
			inv: inventory{pds: []*physicaldrive.PhysicalDrive{drive("", "")}, lvs: []*logicalvolume.LogicalVolume{
				volume("/dev/sdb", "/by-id/b", "1"), volume("/dev/sdc", "/by-id/c", "1"),
			}},
			wantPaths: [2]string{"/dev/sdb", "/by-id/b"},
		},
		{
			name:      "drive in no volume keeps its own paths",
			inv:       inventory{pds: []*physicaldrive.PhysicalDrive{drive("/dev/sda", "")}, lvs: []*logicalvolume.LogicalVolume{volume("/dev/sdb", "/by-id/b", "2")}},
			wantPaths: [2]string{"/dev/sda", ""},
		},
		{
			name:       "volumes not listed",
			inv:        inventory{pds: []*physicaldrive.PhysicalDrive{drive("/dev/sda", "")}, volumesErr: errors.New("boom")},
			wantPaths:  [2]string{"/dev/sda", ""},
			wantVolErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			disks, volumesErr, err := scenario.Report(tt.inv, &raidcontroller.Metadata{ID: 0})
			require.NoError(t, err)
			require.Equal(t, tt.wantVolErr, volumesErr != nil)
			require.Len(t, disks, 1)
			require.Equal(t, tt.wantPaths, [2]string{disks[0].DevicePath, disks[0].PermanentPath})
		})
	}
}
