package logicalvolumegetter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
)

const (
	mdadmExportOutput = `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=0030d06e:fd0fa07d:0d04737a:dc97e22c
MD_NAME=0
MD_DEVICE_dev_nvme1n1_ROLE=1
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1`

	mdadmMultiplePDsExportOutput = `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=0030d06e:fd0fa07d:0d04737a:dc97e22c
MD_NAME=0
MD_DEVICE_dev_nvme1n1_ROLE=1
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1
MD_DEVICE_dev_nvme2n1_ROLE=2
MD_DEVICE_dev_nvme2n1_DEV=/dev/nvme2n1`

	mdadmMultipleLogicalVolumesExportOutput = `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=2324eedd:1728e4cd:9436cae5:3bc05c63
MD_NAME=0
MD_DEVICE_dev_nvme2n1_ROLE=0
MD_DEVICE_dev_nvme2n1_DEV=/dev/nvme2n1
MD_DEVICE_dev_nvme1n1_ROLE=1
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1
MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=ce9f3ef6:917f16d7:900f8175:652f76d9
MD_NAME=1
MD_DEVICE_dev_nvme4n1_ROLE=0
MD_DEVICE_dev_nvme4n1_DEV=/dev/nvme4n1
MD_DEVICE_dev_nvme3n1_ROLE=1
MD_DEVICE_dev_nvme3n1_DEV=/dev/nvme3n1`
)

type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)

	return arguments.Get(0).([]byte), arguments.Error(1)
}

func TestMDADMLogicalVolumesSingle(t *testing.T) {
	// // Create a mock object
	mockRunner := &MockCommandRunner{}

	// Set up expected behavior of the mock
	mockRunner.On("Run", []string{"--detail", "--scan", "--export"}).Return([]byte(mdadmExportOutput), nil)

	// Use the mock object in your test
	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolumes, err := mdadm.LogicalVolumes(nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(logicalVolumes))
	assert.Equal(t, "0", logicalVolumes[0].ID)
	assert.Equal(t, 1, len(logicalVolumes[0].PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolumes[0].RAIDLevel)
}

func TestMDADMLogicalVolumes(t *testing.T) {
	// // Create a mock object
	mockRunner := &MockCommandRunner{}

	// Set up expected behavior of the mock
	mockRunner.On("Run", []string{"--detail", "--scan", "--export"}).Return([]byte(mdadmMultipleLogicalVolumesExportOutput), nil)

	// Use the mock object in your test
	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolumes, err := mdadm.LogicalVolumes(nil)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(logicalVolumes))

	assert.Equal(t, "0", logicalVolumes[0].ID)
	assert.Equal(t, 2, len(logicalVolumes[0].PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolumes[0].RAIDLevel)

	assert.Equal(t, "1", logicalVolumes[1].ID)
	assert.Equal(t, 2, len(logicalVolumes[1].PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolumes[1].RAIDLevel)
}

func TestMDADMLogicalVolume(t *testing.T) {
	// // Create a mock object
	mockRunner := &MockCommandRunner{}

	// Set up expected behavior of the mock
	mockRunner.On("Run", []string{"--detail", "/dev/md0", "--export"}).Return([]byte(mdadmExportOutput), nil)

	// Use the mock object in your test
	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolume, err := mdadm.LogicalVolume(&logicalvolume.Metadata{
		ID: "0",
	})

	assert.Nil(t, err)
	assert.Equal(t, "0", logicalVolume.ID)
	assert.Equal(t, 1, len(logicalVolume.PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolume.RAIDLevel)
}

func TestMDADMLogicalVolumeMultiplePDs(t *testing.T) {
	// // Create a mock object
	mockRunner := &MockCommandRunner{}

	// Set up expected behavior of the mock
	mockRunner.On("Run", []string{"--detail", "/dev/md0", "--export"}).Return([]byte(mdadmMultiplePDsExportOutput), nil)

	// Use the mock object in your test
	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolume, err := mdadm.LogicalVolume(&logicalvolume.Metadata{
		ID: "0",
	})

	assert.Nil(t, err)
	assert.Equal(t, "0", logicalVolume.ID)
	assert.Equal(t, 2, len(logicalVolume.PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolume.RAIDLevel)
}

const (
	mdadmSingleLogicalVolumeExportOutput = `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=0030d06e:fd0fa07d:0d04737a:dc97e22c
MD_NAME=0
MD_DEVICE_dev_nvme1n1_ROLE=1
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1`
)

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
					UUID:         "2324eedd:1728e4cd:9436cae5:3bc05c63",
					Name:         "0",
					ArraySize:    "",
					DeviceName:   "",
					Devices: map[string]logicalvolumegetter.MDADMDevices{
						"dev_nvme1n1": {
							Role:  "1",
							State: "",
							Path:  "/dev/nvme1n1",
						},
						"dev_nvme2n1": {
							Role:  "0",
							State: "",
							Path:  "/dev/nvme2n1",
						},
					},
				},
				{
					RaidLevel:    logicalvolume.RAIDLevel1,
					DevicesCount: 2,
					Metadata:     "1.2",
					UUID:         "ce9f3ef6:917f16d7:900f8175:652f76d9",
					Name:         "1",
					ArraySize:    "",
					DeviceName:   "",
					Devices: map[string]logicalvolumegetter.MDADMDevices{
						"dev_nvme3n1": {
							Role:  "1",
							State: "",
							Path:  "/dev/nvme3n1",
						},
						"dev_nvme4n1": {
							Role:  "0",
							State: "",
							Path:  "/dev/nvme4n1",
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
	}

	for _, testCase := range testCases {
		details, err := logicalvolumegetter.ParseMDADMExportOutput(testCase.output)
		if (err != nil) != testCase.wantErr {
			t.Errorf("TestParseMDADMExportOutput(%s) error = %v, wantErr %v", testCase.name, err, testCase.wantErr)
			t.FailNow()
		}

		assert.Equal(t, testCase.want, details)
		assert.Nil(t, err)
	}
}
