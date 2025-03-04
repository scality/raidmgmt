package logicalvolumegetter_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
)

const (
	mdadmMultipleLogicalVolumesExportOutput = `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=ca573640:118fef94:7d51fb5d:84a51fe3
MD_DEVNAME=0_0
MD_NAME=0
MD_DEVICE_dev_nvme2n1_ROLE=1
MD_DEVICE_dev_nvme2n1_DEV=/dev/nvme2n1
MD_DEVICE_dev_nvme1n1_ROLE=0
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1
MD_LEVEL=raid0
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=2a1f5465:f41a66cc:48a52570:278055c7
MD_NAME=1
MD_DEVICE_dev_nvme4n1_ROLE=1
MD_DEVICE_dev_nvme4n1_DEV=/dev/nvme4n1
MD_DEVICE_dev_nvme3n1_ROLE=0
MD_DEVICE_dev_nvme3n1_DEV=/dev/nvme3n1`

	mdadmDetailOutput = `/dev/md0:
           Version : 1.2
     Creation Time : Mon Mar  3 15:23:21 2025
        Raid Level : raid1
        Array Size : 8379392 (7.99 GiB 8.58 GB)
     Used Dev Size : 8379392 (7.99 GiB 8.58 GB)
      Raid Devices : 2
     Total Devices : 2
       Persistence : Superblock is persistent

       Update Time : Mon Mar  3 15:24:26 2025
             State : active
    Active Devices : 2
   Working Devices : 2
    Failed Devices : 0
     Spare Devices : 0

Consistency Policy : resync

              Name : 0
              UUID : ca573640:118fef94:7d51fb5d:84a51fe3
            Events : 17

    Number   Major   Minor   RaidDevice State
       0     259        0        0      active sync   /dev/nvme1n1
       1     259        5        1      active sync   /dev/nvme2n1`

	mdadmDetailDegradedOutput = `/dev/md0:
           Version : 1.2
     Creation Time : Mon Mar  3 15:23:21 2025
        Raid Level : raid1
        Array Size : 8379392 (7.99 GiB 8.58 GB)
     Used Dev Size : 8379392 (7.99 GiB 8.58 GB)
      Raid Devices : 2
     Total Devices : 1
       Persistence : Superblock is persistent

       Update Time : Mon Mar  3 15:24:26 2025
             State : degraded
    Active Devices : 1
   Working Devices : 1
    Failed Devices : 1
     Spare Devices : 0

Consistency Policy : resync

              Name : 0
              UUID : ca573640:118fef94:7d51fb5d:84a51fe3
            Events : 17

    Number   Major   Minor   RaidDevice State
       0     259        0        0      active sync   /dev/nvme1n1
       1     259        5        1      failed   /dev/nvme2n1`

	mdadmSingleLogicalVolumeExportOutput = `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=0030d06e:fd0fa07d:0d04737a:dc97e22c
MD_NAME=0
MD_DEVICE_dev_nvme1n1_ROLE=1
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1`
)

type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)
	return arguments.Get(0).([]byte), arguments.Error(1)
}

func TestMDADMLogicalVolumes(t *testing.T) {
	// Create a mock object
	mockRunner := new(MockCommandRunner)

	// Set up expected behavior of the mock
	mockRunner.On("Run", []string{"--detail", "--scan", "--export"}).Return([]byte(mdadmMultipleLogicalVolumesExportOutput), nil)
	mockRunner.On("Run", []string{"--detail", "/dev/md/0"}).Return([]byte(mdadmDetailOutput), nil)
	mockRunner.On("Run", []string{"--detail", "/dev/md/0", "--export"}).Return([]byte(mdadmSingleLogicalVolumeExportOutput), nil)
	mockRunner.On("Run", []string{"--detail", "/dev/md/1"}).Return([]byte(mdadmDetailOutput), nil)
	mockRunner.On("Run", []string{"--detail", "/dev/md/1", "--export"}).Return([]byte(mdadmSingleLogicalVolumeExportOutput), nil)

	// Use the mock object in your test
	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolumes, err := mdadm.LogicalVolumes(nil)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(logicalVolumes))
	assert.Equal(t, "/dev/md/0", logicalVolumes[0].DevicePath)
	assert.Equal(t, "/dev/md/1", logicalVolumes[1].DevicePath)
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolumes[0].RAIDLevel)

	mockRunner.AssertExpectations(t)
}

func TestMDADMLogicalVolumes_ScanError(t *testing.T) {
	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"--detail", "--scan", "--export"}).Return([]byte{}, errors.New("command failed"))

	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolumes, err := mdadm.LogicalVolumes(nil)

	assert.Error(t, err)
	assert.Nil(t, logicalVolumes)
	assert.Contains(t, err.Error(), "failed to run mdadm detail scan export command")

	mockRunner.AssertExpectations(t)
}

func TestMDADMLogicalVolume(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	mockRunner.On("Run", []string{"--detail", "/dev/md/test_raid1"}).Return([]byte(mdadmDetailOutput), nil)
	mockRunner.On("Run", []string{"--detail", "/dev/md/test_raid1", "--export"}).Return([]byte(`MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=ca573640:118fef94:7d51fb5d:84a51fe3
MD_NAME=0
MD_DEVICE_dev_nvme2n1_ROLE=1
MD_DEVICE_dev_nvme2n1_DEV=/dev/nvme2n1
MD_DEVICE_dev_nvme1n1_ROLE=0
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1`), nil)

	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolume, err := mdadm.LogicalVolume(&logicalvolume.Metadata{
		ID: "test_raid1",
	})

	assert.NoError(t, err)
	assert.Equal(t, "/dev/md/test_raid1", logicalVolume.DevicePath)
	assert.Equal(t, 2, len(logicalVolume.PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolume.RAIDLevel)
	assert.Equal(t, logicalvolume.LVStatusOptimal, logicalVolume.Status)
	assert.Equal(t, uint64(8379392), logicalVolume.Size)

	mockRunner.AssertExpectations(t)
}

func TestMDADMLogicalVolume_DegradedStatus(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	mockRunner.On("Run", []string{"--detail", "/dev/md/degraded_raid"}).Return([]byte(mdadmDetailDegradedOutput), nil)
	mockRunner.On("Run", []string{"--detail", "/dev/md/degraded_raid", "--export"}).Return([]byte(`MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=ca573640:118fef94:7d51fb5d:84a51fe3
MD_NAME=0
MD_DEVICE_dev_nvme1n1_ROLE=0
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1`), nil)

	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolume, err := mdadm.LogicalVolume(&logicalvolume.Metadata{
		ID: "degraded_raid",
	})

	assert.NoError(t, err)
	assert.Equal(t, "/dev/md/degraded_raid", logicalVolume.DevicePath)
	assert.Equal(t, logicalvolume.LVStatusDegraded, logicalVolume.Status)

	mockRunner.AssertExpectations(t)
}

func TestMDADMLogicalVolume_DetailError(t *testing.T) {
	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"--detail", "/dev/md/test_raid1"}).Return([]byte{}, errors.New("command failed"))

	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolume, err := mdadm.LogicalVolume(&logicalvolume.Metadata{
		ID: "test_raid1",
	})

	assert.Error(t, err)
	assert.Nil(t, logicalVolume)
	assert.Contains(t, err.Error(), "failed to get logical volume status")

	mockRunner.AssertExpectations(t)
}

func TestMDADMLogicalVolume_ExportError(t *testing.T) {
	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"--detail", "/dev/md/test_raid1"}).Return([]byte(mdadmDetailOutput), nil)
	mockRunner.On("Run", []string{"--detail", "/dev/md/test_raid1", "--export"}).Return([]byte{}, errors.New("command failed"))

	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolume, err := mdadm.LogicalVolume(&logicalvolume.Metadata{
		ID: "test_raid1",
	})

	assert.Error(t, err)
	assert.Nil(t, logicalVolume)
	assert.Contains(t, err.Error(), "failed to run mdadm detail export command")

	mockRunner.AssertExpectations(t)
}

func TestParseMDADMExportOutput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		output  []byte
		want    []*logicalvolumegetter.MDADMExportDetails
		wantErr bool
	}{
		{
			name:   "Valid single logical volume output",
			output: []byte(mdadmSingleLogicalVolumeExportOutput),
			want: []*logicalvolumegetter.MDADMExportDetails{
				{
					RaidLevel:    logicalvolume.RAIDLevel1,
					DevicesCount: 2,
					Metadata:     "1.2",
					UUID:         "0030d06e:fd0fa07d:0d04737a:dc97e22c",
					Name:         "0",
					ArraySize:    "",
					DeviceName:   "",
					Devices: map[string]logicalvolumegetter.MDADMDevices{
						"dev_nvme1n1": {
							Role:  "1",
							State: "",
							Path:  "/dev/nvme1n1",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "Valid multiple logical volumes output",
			output: []byte(mdadmMultipleLogicalVolumesExportOutput),
			want: []*logicalvolumegetter.MDADMExportDetails{
				{
					RaidLevel:    logicalvolume.RAIDLevel1,
					DevicesCount: 2,
					Metadata:     "1.2",
					UUID:         "ca573640:118fef94:7d51fb5d:84a51fe3",
					Name:         "0",
					DeviceName:   "0_0",
					Devices: map[string]logicalvolumegetter.MDADMDevices{
						"dev_nvme2n1": {
							Role: "1",
							Path: "/dev/nvme2n1",
						},
						"dev_nvme1n1": {
							Role: "0",
							Path: "/dev/nvme1n1",
						},
					},
				},
				{
					RaidLevel:    logicalvolume.RAIDLevel0,
					DevicesCount: 2,
					Metadata:     "1.2",
					UUID:         "2a1f5465:f41a66cc:48a52570:278055c7",
					Name:         "1",
					Devices: map[string]logicalvolumegetter.MDADMDevices{
						"dev_nvme4n1": {
							Role: "1",
							Path: "/dev/nvme4n1",
						},
						"dev_nvme3n1": {
							Role: "0",
							Path: "/dev/nvme3n1",
						},
					},
				},
			},
		},
		{
			name:    "Empty output",
			output:  []byte(""),
			want:    []*logicalvolumegetter.MDADMExportDetails{},
			wantErr: false,
		},
		{
			name:    "nil output",
			output:  nil,
			want:    []*logicalvolumegetter.MDADMExportDetails{},
			wantErr: false,
		},
		{
			name:    "Invalid device line",
			output:  []byte("MD_LEVEL=raid1\nMD_DEVICE_invalid"),
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			details, err := logicalvolumegetter.ParseMDADMExportOutput(tc.output)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseMDADMExportOutput() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				assert.Equal(t, tc.want, details)
			}
		})
	}
}

func TestDeviceNameToDevicePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		deviceName string
		want       string
	}{
		{
			name:       "Full path",
			deviceName: "/dev/md0",
			want:       "/dev/md0",
		},
		{
			name:       "md with slash prefix",
			deviceName: "md/0_0",
			want:       "/dev/md/0_0",
		},
		{
			name:       "md name with underscore",
			deviceName: "test_1",
			want:       "/dev/md/test_1",
		},
		{
			name:       "Simple md name",
			deviceName: "0",
			want:       "/dev/md/0",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := logicalvolumegetter.DeviceNameToDevicePath(tt.deviceName)
			assert.Equal(t, tt.want, result)
		})
	}
}
