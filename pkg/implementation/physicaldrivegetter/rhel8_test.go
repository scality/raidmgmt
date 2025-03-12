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
	lsblkTestOutput = `NAME         ROTA        SIZE TYPE
/dev/nvme5n1    0  8589934592 disk
/dev/nvme2n1    0  8589934592 disk
/dev/nvme4n1    0  8589934592 disk
/dev/nvme1n1    0  8589934592 disk
/dev/nvme0n1    0 16106127360 disk
/dev/nvme3n1    0  8589934592 disk`

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

func TestParseUDevADMOutput(t *testing.T) {
	output := []byte(uDevADMTestOutput)

	expected := &physicaldrive.PhysicalDrive{
		Model:  "Amazon Elastic Block Store",
		Serial: "vol05ece746e40ff492f",
		ID:     "nvme.1d0f-766f6c3035656365373436653430666634393266-416d617a6f6e20456c617374696320426c6f636b2053746f7265-00000001",
		Metadata: &physicaldrive.Metadata{
			DevicePath: "/dev/nvme1n1",
		},
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
	})).Return([]byte(`NAME         ROTA       SIZE TYPE TRAN   MOUNTPOINT FSTYPE PARTTYPE
/dev/nvme1n1    0 8589934592 disk nvme`), nil)

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

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	// Verify result
	assert.NoError(t, err)
	assert.Equal(t, "Amazon Elastic Block Store", physicalDrive.Model)
	assert.Equal(t, "vol05ece746e40ff492f", physicalDrive.Serial)
	assert.Equal(t, "nvme.1d0f-123456", physicalDrive.ID)
	assert.Equal(t, uint64(8589934592), physicalDrive.Size)
	assert.Equal(t, physicaldrive.DiskTypeNVMe, physicalDrive.Type)
	assert.Equal(t, physicaldrive.PDStatusUnassignedGood, physicalDrive.Status)
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

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`NAME        ROTA       SIZE TYPE TRAN MOUNTPOINT FSTYPE PARTTYPE
/dev/sda       0 1000000000 disk sata`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Samsung SSD
E: ID_SERIAL_SHORT=S12345
E: ID_WWN=wwn.500
E: DEVNAME=/dev/sda
E: DEVLINKS=/dev/disk/by-id/ssd-123`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: PASSED`), nil)

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/sda"}
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

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`NAME        ROTA       SIZE TYPE TRAN MOUNTPOINT FSTYPE PARTTYPE
/dev/sdb       1 2000000000 disk sata`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Seagate HDD
E: ID_SERIAL_SHORT=HD12345
E: ID_WWN=wwn.600
E: DEVNAME=/dev/sdb`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: PASSED`), nil)

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/sdb"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	assert.NoError(t, err)
	assert.Equal(t, "Seagate HDD", physicalDrive.Model)
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

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}
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

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`NAME         ROTA       SIZE TYPE TRAN
/dev/nvme1n1    0 8589934592 disk nvme`), nil)

	// Simulate error from udevadm
	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte{}, errors.New("udevadm command failed"))

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}
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

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`NAME         ROTA       SIZE TYPE TRAN
/dev/nvme1n1    0 8589934592 disk nvme`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Test Model
E: ID_SERIAL_SHORT=123456
E: ID_WWN=wwn.123
E: DEVNAME=/dev/nvme1n1`), nil)

	// Simulate error from smartctl
	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte{}, errors.New("smartctl command failed"))

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	// FIXME Ignore errors for now
	assert.NoError(t, err)
	assert.Equal(t, physicalDrive.Status, physicaldrive.PDStatusUnassignedGood)
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

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`NAME   MAJ:MIN RM  SIZE RO TYPE MOUNTPOINT
vda1 253:1    0  4000000  0 part /`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Test Model
E: ID_SERIAL_SHORT=123456
E: ID_WWN=wwn.123
E: DEVNAME=/dev/nvme1n1`), nil)

	// Simulate error from smartctl
	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte{}, errors.New("smartctl command failed"))

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}
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

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`NAME        ROTA       SIZE TYPE TRAN
/dev/xda       2 1000000000 disk other`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Unknown Disk
E: ID_SERIAL_SHORT=X12345
E: ID_WWN=wwn.700
E: DEVNAME=/dev/xda`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: PASSED`), nil)

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/xda"}
	physicalDrive, err := r.PhysicalDrive(metadata)

	assert.NoError(t, err)
	assert.Equal(t, "Unknown Disk", physicalDrive.Model)
	assert.Equal(t, physicaldrive.DiskTypeUnknown, physicalDrive.Type)
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

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`NAME        ROTA       SIZE TYPE TRAN MOUNTPOINT  FSTYPE    PARTTYPE
/dev/sda       0 1000000000 disk sata /mnt/data ext4      linux`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Test SSD
E: ID_SERIAL_SHORT=123456
E: ID_WWN=wwn.800
E: DEVNAME=/dev/sda`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: PASSED`), nil)

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/sda"}
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

	mockLSBLK.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`NAME        ROTA       SIZE TYPE TRAN
/dev/sda       0 1000000000 disk sata`), nil)

	mockUDevADM.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`E: ID_MODEL=Test SSD
E: ID_SERIAL_SHORT=123456
E: ID_WWN=wwn.900
E: DEVNAME=/dev/sda`), nil)

	mockSmartCTL.On("Run", mock.AnythingOfType("[]string")).Return([]byte(`SMART overall-health self-assessment test result: FAILED
Reallocated Sector Count: 5`), nil)

	metadata := &physicaldrive.Metadata{DevicePath: "/dev/sda"}
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
	})).Return([]byte(`NAME         ROTA        SIZE TYPE TRAN   MOUNTPOINT FSTYPE PARTTYPE
/dev/nvme1n1    0  8589934592 disk nvme
/dev/nvme2n1    0  8589934592 disk nvme`), nil)

	// Setup mocks for first device
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "/dev/nvme1n1"
	})).Return([]byte(`NAME         ROTA       SIZE TYPE TRAN   MOUNTPOINT FSTYPE PARTTYPE
/dev/nvme1n1    0 8589934592 disk nvme`), nil)

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

	// Setup mocks for second device
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "/dev/nvme2n1"
	})).Return([]byte(`NAME         ROTA       SIZE TYPE TRAN   MOUNTPOINT FSTYPE PARTTYPE
/dev/nvme2n1    0 8589934592 disk nvme`), nil)

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
	assert.Equal(t, 2, len(physicalDrives))

	// First drive assertions
	assert.Equal(t, "Amazon Elastic Block Store", physicalDrives[0].Model)
	assert.Equal(t, "vol05ece746e40ff492f", physicalDrives[0].Serial)
	assert.Equal(t, "/dev/nvme1n1", physicalDrives[0].DevicePath)
	assert.Equal(t, physicaldrive.DiskTypeNVMe, physicalDrives[0].Type)

	// Second drive assertions
	assert.Equal(t, "Amazon Elastic Block Store", physicalDrives[1].Model)
	assert.Equal(t, "vol05ece746e40ff493g", physicalDrives[1].Serial)
	assert.Equal(t, "/dev/nvme2n1", physicalDrives[1].DevicePath)
	assert.Equal(t, physicaldrive.DiskTypeNVMe, physicalDrives[1].Type)

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
	})).Return([]byte(`NAME         ROTA        SIZE TYPE TRAN
/dev/nvme1n1    0  8589934592 disk nvme
/dev/nvme2n1    0  8589934592 disk nvme`), nil)

	// Setup mock for first device
	mockLSBLK.On("Run", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "/dev/nvme1n1"
	})).Return([]byte(`NAME         ROTA       SIZE TYPE TRAN
/dev/nvme1n1    0 8589934592 disk nvme`), nil)

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
