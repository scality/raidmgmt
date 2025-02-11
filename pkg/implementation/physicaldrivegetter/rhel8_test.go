package physicaldrivegetter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/physicaldrivegetter"
)

const (
	lsblkTestOutput = `NAME MAJ:MIN RM SIZE RO TYPE MOUNTPOINT
md127 9:127 0 17160994816 0 raid0
md127 9:127 0 17160994816 0 raid0
/dev/nvme1n1 259:0  0 8589934592 0 disk
/dev/nvme2n1 259:1  0 8589934592 0 disk
/dev/nvme5n1 259:2  0 8589934592 0 disk
/dev/nvme4n1 259:3  0 8589934592 0 disk
/dev/nvme0n1 259:4  0 16106127360 0 disk
/dev/nvme0n1p1 259:5  0 16105078784 0 part /
/dev/nvme3n1 259:6  0 8589934592 0 disk`

	uDevADMTestOutput = `P: /devices/pci0000:00/0000:00:1b.0/nvme/nvme1/nvme1n1
N: nvme1n1
S: disk/by-id/nvme-Amazon_Elastic_Block_Store_vol05ece746e40ff492f
S: disk/by-id/nvme-nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001
S: disk/by-path/pci-0000:00:1b.0-nvme-1
E: DEVLINKS=/dev/disk/by-id/nvme-Amazon_Elastic_Block_Store_vol05ece746e40ff492f /dev/disk/by-id/nvme-nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001 /dev/disk/by-path/pci-0000:00:1b.0-nvme-1
E: DEVNAME=/dev/nvme1n1
E: DEVPATH=/devices/pci0000:00/0000:00:1b.0/nvme/nvme1/nvme1n1
E: DEVTYPE=disk
E: ID_MODEL=Amazon Elastic Block Store
E: ID_PATH=pci-0000:00:1b.0-nvme-1
E: ID_PATH_TAG=pci-0000_00_1b_0-nvme-1
E: ID_SERIAL=Amazon Elastic Block Store_vol05ece746e40ff492f
E: ID_SERIAL_SHORT=vol05ece746e40ff492f
E: ID_WWN=nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001
E: ID_WWN_WITH_EXTENSION=nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001
E: MAJOR=259
E: MINOR=2
E: SUBSYSTEM=block
E: TAGS=:systemd:
E: USEC_INITIALIZED=4070636`
)

func TestParseLSBLKOutput(t *testing.T) {
	output := []byte(lsblkTestOutput)

	expected := []physicaldrivegetter.BlockDevice{
		{DevicePath: "/dev/nvme1n1", MajMin: "259:0", RM: "0", Size: 8589934592, RO: "0", Type: "disk", Mountpoint: ""},
		{DevicePath: "/dev/nvme2n1", MajMin: "259:1", RM: "0", Size: 8589934592, RO: "0", Type: "disk", Mountpoint: ""},
		{DevicePath: "/dev/nvme5n1", MajMin: "259:2", RM: "0", Size: 8589934592, RO: "0", Type: "disk", Mountpoint: ""},
		{DevicePath: "/dev/nvme4n1", MajMin: "259:3", RM: "0", Size: 8589934592, RO: "0", Type: "disk", Mountpoint: ""},
		{DevicePath: "/dev/nvme0n1", MajMin: "259:4", RM: "0", Size: 16106127360, RO: "0", Type: "disk", Mountpoint: ""},
		{DevicePath: "/dev/nvme3n1", MajMin: "259:6", RM: "0", Size: 8589934592, RO: "0", Type: "disk", Mountpoint: ""},
	}

	devices, err := physicaldrivegetter.ParseLSBLKOutput(output)
	assert.NoError(t, err)
	assert.Equal(t, expected, devices)
}

func TestParseUDevADMOutput(t *testing.T) {
	output := []byte(uDevADMTestOutput)

	expected := &physicaldrive.PhysicalDrive{
		Model:         "Amazon Elastic Block Store",
		Serial:        "vol05ece746e40ff492f",
		ID:            "nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001",
		DevicePath:    "/dev/nvme1n1",
		PermanentPath: "/dev/disk/by-id/nvme-nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001",
	}

	physicalDrive, err := physicaldrivegetter.ParseUDevADMOutput(output)
	assert.NoError(t, err)
	assert.Equal(t, expected, physicalDrive)
}

type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	ret := m.Called(args)
	return ret.Get(0).([]byte), ret.Error(1)
}

func TestPhysicalDrives(t *testing.T) {
	mockUDevADM := MockCommandRunner{}
	mockLSBLK := MockCommandRunner{}

	r := physicaldrivegetter.RHEL8{UDevADM: &mockUDevADM, LSBLK: &mockLSBLK}

	lsblkOutput := []byte(`NAME MAJ:MIN RM SIZE RO TYPE MOUNTPOINT
/dev/nvme1n1 259:0  0 8589934592 0 disk
/dev/nvme2n1 259:1  0 8589934592 0 disk`)

	uDevADMOutput := []byte(`E: ID_MODEL=Amazon Elastic Block Store
E: ID_SERIAL_SHORT=vol05ece746e40ff492f
E: ID_WWN=nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001
E: DEVNAME=/dev/nvme1n1`)

	mockLSBLK.On("Run", []string{"--list", "--paths", "--bytes"}).Return(lsblkOutput, nil)
	mockUDevADM.On("Run", []string{"info", "--query=all", "--name=/dev/nvme1n1"}).Return(uDevADMOutput, nil)
	mockUDevADM.On("Run", []string{"info", "--query=all", "--name=/dev/nvme2n1"}).Return(uDevADMOutput, nil)

	metadata := &raidcontroller.Metadata{}
	expected := []*physicaldrive.PhysicalDrive{
		{
			Model:      "Amazon Elastic Block Store",
			Serial:     "vol05ece746e40ff492f",
			ID:         "nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001",
			DevicePath: "/dev/nvme1n1",
			Size:       8589934592,
			Type:       physicaldrive.DiskTypeSSD,
		},
		{
			Model:      "Amazon Elastic Block Store",
			Serial:     "vol05ece746e40ff492f",
			ID:         "nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001",
			DevicePath: "/dev/nvme2n1",
			Size:       8589934592,
			Type:       physicaldrive.DiskTypeSSD,
		},
	}

	physicalDrives, err := r.PhysicalDrives(metadata)
	assert.NoError(t, err)
	assert.Equal(t, expected, physicalDrives)

	mockLSBLK.AssertExpectations(t)
	mockUDevADM.AssertExpectations(t)
}

func TestPhysicalDrive(t *testing.T) {
	mockUDevADM := MockCommandRunner{}
	mockLSBLK := MockCommandRunner{}

	r := physicaldrivegetter.RHEL8{UDevADM: &mockUDevADM, LSBLK: &mockLSBLK}

	lsblkOutput := []byte(`NAME MAJ:MIN RM SIZE RO TYPE MOUNTPOINT
/dev/nvme1n1 259:0  0 8589934592 0 disk`)

	udevadmOutput := []byte(`E: ID_MODEL=Amazon Elastic Block Store
E: ID_SERIAL_SHORT=vol05ece746e40ff492f
E: ID_WWN=nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001
E: DEVNAME=/dev/nvme1n1`)

	mockLSBLK.On("Run", []string{"--list", "--paths", "--bytes"}).Return(lsblkOutput, nil)
	mockUDevADM.On("Run", []string{"info", "--query=all", "--name=/dev/nvme1n1"}).Return(udevadmOutput, nil)

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}
	expected := &physicaldrive.PhysicalDrive{
		Model:      "Amazon Elastic Block Store",
		Serial:     "vol05ece746e40ff492f",
		ID:         "nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001",
		DevicePath: "/dev/nvme1n1",
		Size:       8589934592,
		Type:       physicaldrive.DiskTypeSSD,
	}

	physicalDrive, err := r.PhysicalDrive(metadata)
	assert.NoError(t, err)
	assert.Equal(t, expected, physicalDrive)

	mockLSBLK.AssertExpectations(t)
	mockUDevADM.AssertExpectations(t)
}
