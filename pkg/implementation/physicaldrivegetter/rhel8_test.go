package physicaldrivegetter_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/physicaldrivegetter"
)

const (
	// lsblkTestOutput is `lsblk --json` from util-linux 2.32 (RHEL 8): every
	// value is a string.
	lsblkTestOutput = `{"blockdevices": [
  {"name": "/dev/nvme5n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": null, "mountpoint": null, "fstype": null, "parttype": null, "pkname": null},
  {"name": "/dev/nvme2n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": null, "mountpoint": null, "fstype": null, "parttype": null, "pkname": null},
  {"name": "/dev/nvme4n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": null, "mountpoint": null, "fstype": null, "parttype": null, "pkname": null},
  {"name": "/dev/nvme1n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": null, "mountpoint": null, "fstype": null, "parttype": null, "pkname": null},
  {"name": "/dev/nvme0n1", "rota": "0", "size": "16106127360", "type": "disk", "tran": null, "mountpoint": null, "fstype": null, "parttype": null, "pkname": null},
  {"name": "/dev/nvme3n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": null, "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`

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
		{DevicePath: "/dev/nvme5n1", Size: 8589934592, Type: "disk", Rotational: "0"},
		{DevicePath: "/dev/nvme2n1", Size: 8589934592, Type: "disk", Rotational: "0"},
		{DevicePath: "/dev/nvme4n1", Size: 8589934592, Type: "disk", Rotational: "0"},
		{DevicePath: "/dev/nvme1n1", Size: 8589934592, Type: "disk", Rotational: "0"},
		{DevicePath: "/dev/nvme0n1", Size: 16106127360, Type: "disk", Rotational: "0"},
		{DevicePath: "/dev/nvme3n1", Size: 8589934592, Type: "disk", Rotational: "0"},
	}

	devices, err := physicaldrivegetter.ParseLSBLKOutput(output)
	assert.NoError(t, err)
	assert.Equal(t, expected, devices)
}

// TestParseLSBLKOutput_EmptyColumns checks that an empty column does not shift
// the following ones (ARTESCA-18303): virtio disks have no transport, so TRAN
// is empty on every row.
func TestParseLSBLKOutput_EmptyColumns(t *testing.T) {
	expected := []physicaldrivegetter.BlockDevice{
		{DevicePath: "/dev/vda", Size: 10737418240, Rotational: "1", Type: "disk"},
		{
			DevicePath:       "/dev/vda1",
			Size:             10736352768,
			Rotational:       "1",
			Type:             "part",
			Tran:             "",
			MountPoint:       "/",
			FileSystemType:   "ext4",
			PartitionType:    "0x83",
			ParentKernelName: "/dev/vda",
		},
	}

	tests := []struct {
		name   string
		output string
	}{
		{
			// util-linux 2.32 (RHEL 8) prints every value as a string.
			name: "util-linux 2.32",
			output: `{"blockdevices": [
  {"name": "/dev/vda", "rota": "1", "size": "10737418240", "type": "disk", "tran": null, "mountpoint": null, "fstype": null, "parttype": null, "pkname": null},
  {"name": "/dev/vda1", "rota": "1", "size": "10736352768", "type": "part", "tran": null, "mountpoint": "/", "fstype": "ext4", "parttype": "0x83", "pkname": "/dev/vda"}
]}`,
		},
		{
			// util-linux 2.37 (RHEL 9) prints booleans and numbers.
			name: "util-linux 2.37",
			output: `{"blockdevices": [
  {"name": "/dev/vda", "rota": true, "size": 10737418240, "type": "disk", "tran": null, "mountpoint": null, "fstype": null, "parttype": null, "pkname": null},
  {"name": "/dev/vda1", "rota": true, "size": 10736352768, "type": "part", "tran": null, "mountpoint": "/", "fstype": "ext4", "parttype": "0x83", "pkname": "/dev/vda"},
  {"name": "/dev/loop0", "rota": false, "size": 4096, "type": "loop", "tran": null, "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices, err := physicaldrivegetter.ParseLSBLKOutput([]byte(tt.output))
			assert.NoError(t, err)
			assert.Equal(t, expected, devices)
		})
	}
}

func TestParseLSBLKOutput_Errors(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{name: "not json", output: "NAME ROTA SIZE TYPE\n/dev/sda 0 1000 disk"},
		{
			name:   "invalid size",
			output: `{"blockdevices": [{"name": "/dev/sda", "rota": "0", "size": "1G", "type": "disk"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := physicaldrivegetter.ParseLSBLKOutput([]byte(tt.output))
			assert.Error(t, err)
		})
	}
}

func TestParseLSBLKOutput_Empty(t *testing.T) {
	devices, err := physicaldrivegetter.ParseLSBLKOutput([]byte(" \n"))
	assert.NoError(t, err)
	assert.Empty(t, devices)
}

func TestParseUDevADMOutput(t *testing.T) {
	output := []byte(uDevADMTestOutput)

	expected := &physicaldrive.PhysicalDrive{
		Model:         "Amazon Elastic Block Store",
		Serial:        "vol05ece746e40ff492f",
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

func TestRHEL8_PhysicalDrive_Success_NVMe(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	// Setup mocks
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return args[0] == "/dev/nvme1n1"
	})).Return([]byte(`{"blockdevices": [
  {"name": "/dev/nvme1n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.MatchedBy(func(args []string) bool {
		return args[2] == "--name=/dev/nvme1n1"
	})).Return([]byte(`E: ID_MODEL=Amazon Elastic Block Store
E: ID_SERIAL_SHORT=vol05ece746e40ff492f
E: ID_WWN=nvme.1d0f-123456
E: DEVNAME=/dev/nvme1n1
E: DEVLINKS=/dev/disk/by-id/nvme-123 /dev/disk/by-path/pci-0000:00:1b.0-nvme-1`), nil)

	mockSmartCTL.On("Run", []string{"-a", "/dev/nvme1n1"}).Return([]byte(`
=== START OF SMART DATA SECTION ===
SMART overall-health self-assessment test result: PASSED
`), nil)

	metadata := &physicaldrive.Metadata{ID: "/dev/nvme1n1"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	// Verify result
	assert.NoError(t, err)
	assert.Equal(t, "Amazon Elastic Block Store", physicalDrive.Model)
	assert.Equal(t, "vol05ece746e40ff492f", physicalDrive.Serial)
	assert.Equal(t, "/dev/nvme1n1", physicalDrive.ID)
	assert.Equal(t, uint64(8589934592), physicalDrive.Size)
	assert.Equal(t, physicaldrive.DiskTypeNVMe, physicalDrive.Type)
	assert.Equal(t, physicaldrive.PDStatusUnassignedGood, physicalDrive.Status)
	assert.Equal(t, "/dev/nvme1n1", physicalDrive.DevicePath)
	assert.Equal(t, "/dev/disk/by-id/nvme-123", physicalDrive.PermanentPath)

	mockLSBLK.AssertExpectations(t)
	mockUDevADM.AssertExpectations(t)
	mockSmartCTL.AssertExpectations(t)
}

func TestRHEL8_PhysicalDrive_Success_SSD(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`{"blockdevices": [
  {"name": "/dev/sda", "rota": "0", "size": "1000000000", "type": "disk", "tran": "sata", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Samsung SSD
E: ID_SERIAL_SHORT=S12345
E: ID_WWN=wwn.500
E: DEVNAME=/dev/sda
E: DEVLINKS=/dev/disk/by-id/ssd-123`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: PASSED`), nil)

	metadata := &physicaldrive.Metadata{ID: "/dev/sda"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	assert.NoError(t, err)
	assert.Equal(t, "Samsung SSD", physicalDrive.Model)
	assert.Equal(t, physicaldrive.DiskTypeSSD, physicalDrive.Type)
}

func TestRHEL8_PhysicalDrive_Success_HDD(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`{"blockdevices": [
  {"name": "/dev/sdb", "rota": "1", "size": "2000000000", "type": "disk", "tran": "sata", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Seagate HDD
E: ID_SERIAL_SHORT=HD12345
E: ID_WWN=wwn.600
E: DEVNAME=/dev/sdb`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: PASSED`), nil)

	metadata := &physicaldrive.Metadata{ID: "/dev/sdb"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	assert.NoError(t, err)
	assert.Equal(t, "Seagate HDD", physicalDrive.Model)
	assert.Equal(t, "/dev/sdb", physicalDrive.DevicePath)
	assert.Equal(t, physicaldrive.DiskTypeHDD, physicalDrive.Type)
}

func TestRHEL8_PhysicalDrive_BlockDeviceError(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	// Simulate error from lsblk
	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte{}, errors.New("lsblk command failed"))

	metadata := &physicaldrive.Metadata{ID: "/dev/nvme1n1"}
	_, err := r.PhysicalDrive(metadata)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get block device")
}

func TestRHEL8_PhysicalDrive_UDevADMError(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`{"blockdevices": [
  {"name": "/dev/nvme1n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	// Simulate error from udevadm
	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte{}, errors.New("udevadm command failed"))

	metadata := &physicaldrive.Metadata{ID: "/dev/nvme1n1"}
	_, err := r.PhysicalDrive(metadata)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run udevadm physical drive info command")
}

func TestRHEL8_PhysicalDrive_SmartCTLError(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`{"blockdevices": [
  {"name": "/dev/nvme1n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Test Model
E: ID_SERIAL_SHORT=123456
E: ID_WWN=wwn.123
E: DEVNAME=/dev/nvme1n1`), nil)

	// Simulate error from smartctl
	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte{}, errors.New("smartctl command failed"))

	metadata := &physicaldrive.Metadata{ID: "/dev/nvme1n1"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	// FIXME Ignore errors for now
	assert.NoError(t, err)
	assert.Equal(t, physicalDrive.Status, physicaldrive.PDStatusUnassignedGood)
	assert.Equal(t, physicalDrive.Reason, "smartctl command failed to get physical drive status")
	assert.Equal(t, physicalDrive.DevicePath, "/dev/nvme1n1")
}

func TestRHEL8_PhysicalDrive_SmartCTLErrorDeviceUsed(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`{"blockdevices": [
  {"name": "vda1", "rota": "0", "size": "4000000", "type": "part", "tran": null, "mountpoint": "/", "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Test Model
E: ID_SERIAL_SHORT=123456
E: ID_WWN=wwn.123
E: DEVNAME=/dev/nvme1n1`), nil)

	// Simulate error from smartctl
	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte{}, errors.New("smartctl command failed"))

	metadata := &physicaldrive.Metadata{ID: "/dev/nvme1n1"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	// FIXME Ignore errors for now
	assert.NoError(t, err)
	assert.Equal(t, physicalDrive.Status, physicaldrive.PDStatusUsed)
}

func TestRHEL8_PhysicalDrive_UnknownDiskType(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`{"blockdevices": [
  {"name": "/dev/xda", "rota": "2", "size": "1000000000", "type": "disk", "tran": "other", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Unknown Disk
E: ID_SERIAL_SHORT=X12345
E: ID_WWN=wwn.700
E: DEVNAME=/dev/xda`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: PASSED`), nil)

	metadata := &physicaldrive.Metadata{ID: "/dev/xda"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	assert.NoError(t, err)
	assert.Equal(t, "Unknown Disk", physicalDrive.Model)
	assert.Equal(t, physicaldrive.DiskTypeUnknown, physicalDrive.Type)
	assert.Equal(t, physicalDrive.DevicePath, "/dev/xda")
}

func TestRHEL8_PhysicalDrive_UsedStatus(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`{"blockdevices": [
  {"name": "/dev/sda", "rota": "0", "size": "1000000000", "type": "disk", "tran": "sata", "mountpoint": "/mnt/data", "fstype": "ext4", "parttype": "linux", "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Test SSD
E: ID_SERIAL_SHORT=123456
E: ID_WWN=wwn.800
E: DEVNAME=/dev/sda`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: PASSED`), nil)

	metadata := &physicaldrive.Metadata{ID: "/dev/sda"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	assert.NoError(t, err)
	assert.Equal(t, physicaldrive.PDStatusUsed, physicalDrive.Status)
}

func TestRHEL8_PhysicalDrive_FailedStatus(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`{"blockdevices": [
  {"name": "/dev/sda", "rota": "0", "size": "1000000000", "type": "disk", "tran": "sata", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Test SSD
E: ID_SERIAL_SHORT=123456
E: ID_WWN=wwn.900
E: DEVNAME=/dev/sda`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: FAILED
Reallocated Sector Count: 5`), nil)

	metadata := &physicaldrive.Metadata{ID: "/dev/sda"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	assert.NoError(t, err)
	assert.Equal(t, physicaldrive.PDStatusFailed, physicalDrive.Status)
}

func TestRHEL8_PhysicalDrives_Success(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	// Setup mock for listing block devices
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "--list"
	})).Return([]byte(`{"blockdevices": [
  {"name": "/dev/nvme1n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null},
  {"name": "/dev/nvme1n1p1", "rota": "0", "size": "103809024", "type": "part", "tran": "nvme", "mountpoint": "/boot/efi", "fstype": "vfat", "parttype": "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", "pkname": "/dev/nvme1n1"},
  {"name": "/dev/nvme2n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	// Setup mocks for first device
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "/dev/nvme1n1"
	})).Return([]byte(`{"blockdevices": [
  {"name": "/dev/nvme1n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 1 && args[2] == "--name=/dev/nvme1n1"
	})).Return([]byte(`E: ID_MODEL=Amazon Elastic Block Store
E: ID_SERIAL_SHORT=vol05ece746e40ff492f
E: ID_WWN=nvme.1d0f-123456
E: DEVNAME=/dev/nvme1n1
E: DEVLINKS=/dev/disk/by-id/nvme-123`), nil)

	mockSmartCTL.On("Run", []string{"-a", "/dev/nvme1n1"}).Return([]byte(`
	=== START OF SMART DATA SECTION ===
	SMART overall-health self-assessment test result: PASSED
	`), nil)

	// Setup mock for partition of the first device
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "/dev/nvme1n1p1"
	})).Return([]byte(`{"blockdevices": [
  {"name": "/dev/nvme1n1p1", "rota": "0", "size": "103809024", "type": "part", "tran": "nvme", "mountpoint": "/boot/efi", "fstype": "vfat", "parttype": "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", "pkname": "/dev/nvme1n1"}
]}`), nil)

	mockUDevADM.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 1 && args[2] == "--name=/dev/nvme1n1p1"
	})).Return([]byte(`E: ID_MODEL=Amazon Elastic Block Store
E: ID_SERIAL_SHORT=vol05ece746e40ff492f
E: ID_WWN=nvme.1d0f-123456
E: DEVNAME=/dev/nvme1n1p1
E: DEVLINKS=/dev/disk/by-id/nvme-123`), nil)

	mockSmartCTL.On("Run", []string{"-a", "/dev/nvme1n1p1"}).Return([]byte(`
=== START OF SMART DATA SECTION ===
SMART overall-health self-assessment test result: PASSED
`), nil)

	// Setup mocks for second device
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "/dev/nvme2n1"
	})).Return([]byte(`{"blockdevices": [
  {"name": "/dev/nvme2n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	mockUDevADM.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 1 && args[2] == "--name=/dev/nvme2n1"
	})).Return([]byte(`E: ID_MODEL=Amazon Elastic Block Store
E: ID_SERIAL_SHORT=vol05ece746e40ff493g
E: ID_WWN=nvme.1d0f-789012
E: DEVNAME=/dev/nvme2n1
E: DEVLINKS=/dev/disk/by-id/nvme-456`), nil)

	mockSmartCTL.On("Run", []string{"-a", "/dev/nvme2n1"}).Return([]byte(`
=== START OF SMART DATA SECTION ===
SMART overall-health self-assessment test result: PASSED
`), nil)

	// Execute test
	metadata := &raidcontroller.Metadata{}
	physicalDrives, err := r.PhysicalDrives(metadata)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 3, len(physicalDrives))

	// First drive assertions
	assert.Equal(t, "Amazon Elastic Block Store", physicalDrives[0].Model)
	assert.Equal(t, "vol05ece746e40ff492f", physicalDrives[0].Serial)
	assert.Equal(t, "/dev/nvme1n1", physicalDrives[0].ID)
	assert.Equal(t, "/dev/nvme1n1", physicalDrives[0].DevicePath)
	assert.Equal(t, physicaldrive.DiskTypeNVMe, physicalDrives[0].Type)
	assert.Equal(t, physicaldrive.PDStatusUsed, physicalDrives[0].Status)
	assert.False(t, physicalDrives[0].IsPartition)

	// Partition of the first drive assertions
	assert.Equal(t, "Amazon Elastic Block Store", physicalDrives[1].Model)
	assert.Equal(t, "vol05ece746e40ff492f", physicalDrives[1].Serial)
	assert.Equal(t, "/dev/nvme1n1p1", physicalDrives[1].ID)
	assert.Equal(t, "/dev/nvme1n1p1", physicalDrives[1].DevicePath)
	assert.Equal(t, physicaldrive.DiskTypeNVMe, physicalDrives[1].Type)
	assert.Equal(t, physicaldrive.PDStatusUsed, physicalDrives[1].Status)
	assert.True(t, physicalDrives[1].IsPartition)

	// Second drive assertions
	assert.Equal(t, "Amazon Elastic Block Store", physicalDrives[2].Model)
	assert.Equal(t, "vol05ece746e40ff493g", physicalDrives[2].Serial)
	assert.Equal(t, "/dev/nvme2n1", physicalDrives[2].ID)
	assert.Equal(t, "/dev/nvme2n1", physicalDrives[2].DevicePath)
	assert.Equal(t, physicaldrive.DiskTypeNVMe, physicalDrives[2].Type)
	assert.Equal(t, physicaldrive.PDStatusUnassignedGood, physicalDrives[2].Status)
	assert.False(t, physicalDrives[2].IsPartition)

	mockLSBLK.AssertExpectations(t)
	mockUDevADM.AssertExpectations(t)
	mockSmartCTL.AssertExpectations(t)
}

// TestRHEL8_PhysicalDrives_VirtioPartitionedDiskIsUsed checks that a disk with
// a partition is Used when TRAN is empty, as on virtio disks (ARTESCA-18303).
func TestRHEL8_PhysicalDrives_VirtioPartitionedDiskIsUsed(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	vda := `{"name": "/dev/vda", "rota": "1", "size": "10737418240", "type": "disk", "tran": null, ` +
		`"mountpoint": null, "fstype": null, "parttype": null, "pkname": null}`
	vda1 := `{"name": "/dev/vda1", "rota": "1", "size": "10736352768", "type": "part", "tran": null, ` +
		`"mountpoint": "/", "fstype": "ext4", "parttype": "0x83", "pkname": "/dev/vda"}`

	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "--list"
	})).Return([]byte(`{"blockdevices": [`+vda+`, `+vda1+`]}`), nil)

	for path, device := range map[string]string{"/dev/vda": vda, "/dev/vda1": vda1} {
		mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
			return len(args) > 0 && args[0] == path
		})).Return([]byte(`{"blockdevices": [`+device+`]}`), nil)

		mockUDevADM.On("Run", mock.MatchedBy(func(args []string) bool {
			return len(args) > 2 && args[2] == "--name="+path
		})).Return([]byte("E: ID_MODEL=QEMU HARDDISK\nE: DEVNAME="+path), nil)

		mockSmartCTL.On("Run", []string{"-a", path}).Return(
			[]byte("SMART overall-health self-assessment test result: PASSED"), nil,
		)
	}

	physicalDrives, err := r.PhysicalDrives(&raidcontroller.Metadata{})

	assert.NoError(t, err)
	assert.Len(t, physicalDrives, 2)
	assert.Equal(t, "/dev/vda", physicalDrives[0].ID)
	assert.False(t, physicalDrives[0].IsPartition)
	assert.Equal(t, physicaldrive.PDStatusUsed, physicalDrives[0].Status)
	assert.Equal(t, "/dev/vda1", physicalDrives[1].ID)
	assert.True(t, physicalDrives[1].IsPartition)
	assert.Equal(t, physicaldrive.PDStatusUsed, physicalDrives[1].Status)

	mockLSBLK.AssertExpectations(t)
	mockUDevADM.AssertExpectations(t)
	mockSmartCTL.AssertExpectations(t)
}

func TestRHEL8_PhysicalDrives_ListBlockDevicesError(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	// Setup mock to return error when listing block devices
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "--list"
	})).Return([]byte{}, errors.New("failed to list block devices"))

	// Execute test
	metadata := &raidcontroller.Metadata{}
	physicalDrives, err := r.PhysicalDrives(metadata)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, physicalDrives)
	assert.Contains(t, err.Error(), "failed to list block devices")

	mockLSBLK.AssertExpectations(t)
}

func TestRHEL8_PhysicalDrives_PhysicalDriveError(t *testing.T) {
	mockUDevADM := new(MockCommandRunner)
	mockLSBLK := new(MockCommandRunner)
	mockSmartCTL := new(MockCommandRunner)

	r := physicaldrivegetter.RHEL8{
		UDevADM:  mockUDevADM,
		LSBLK:    mockLSBLK,
		SmartCTL: mockSmartCTL,
	}

	// Setup mock for listing block devices
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "--list"
	})).Return([]byte(`{"blockdevices": [
  {"name": "/dev/nvme1n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null},
  {"name": "/dev/nvme2n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	// Setup mock for first device
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "/dev/nvme1n1"
	})).Return([]byte(`{"blockdevices": [
  {"name": "/dev/nvme1n1", "rota": "0", "size": "8589934592", "type": "disk", "tran": "nvme", "mountpoint": null, "fstype": null, "parttype": null, "pkname": null}
]}`), nil)

	// Setup mock to fail on udevadm for the first device
	mockUDevADM.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 1 && args[2] == "--name=/dev/nvme1n1"
	})).Return([]byte{}, errors.New("udevadm command failed"))

	// Execute test
	metadata := &raidcontroller.Metadata{}
	physicalDrives, err := r.PhysicalDrives(metadata)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, physicalDrives)
	assert.Contains(t, err.Error(), "failed to get physical drive")

	mockLSBLK.AssertExpectations(t)
	mockUDevADM.AssertExpectations(t)
}
