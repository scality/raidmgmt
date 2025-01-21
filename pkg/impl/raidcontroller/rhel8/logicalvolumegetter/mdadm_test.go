package logicalvolumegetter_test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/rhel8/logicalvolumegetter"
)

const (
	mdadmExportOutput = `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=0030d06e:fd0fa07d:0d04737a:dc97e22c
MD_NAME=0
MD_DEVICE_dev_nvme1n1_ROLE=1
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1`

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
	assert.Equal(t, "0030d06e:fd0fa07d:0d04737a:dc97e22c", logicalVolumes[0].ID)
	assert.Equal(t, 2, len(logicalVolumes[0].PDrivesMetadata))
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

	assert.Equal(t, "2324eedd:1728e4cd:9436cae5:3bc05c63", logicalVolumes[0].ID)
	assert.Equal(t, 2, len(logicalVolumes[0].PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolumes[0].RAIDLevel)

	assert.Equal(t, "ce9f3ef6:917f16d7:900f8175:652f76d9", logicalVolumes[1].ID)
	assert.Equal(t, 2, len(logicalVolumes[1].PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolumes[1].RAIDLevel)
}

func TestMDADMLogicalVolume(t *testing.T) {
	// // Create a mock object
	mockRunner := &MockCommandRunner{}

	// Set up expected behavior of the mock
	mockRunner.On("Run", []string{"--detail", "--scan", "--export"}).Return([]byte(mdadmExportOutput), nil)

	// Use the mock object in your test
	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolume, err := mdadm.LogicalVolume(&logicalvolume.Metadata{
		ID: "0030d06e:fd0fa07d:0d04737a:dc97e22c",
	})
	spew.Dump(logicalVolume)

	assert.Nil(t, err)
	assert.Equal(t, "0030d06e:fd0fa07d:0d04737a:dc97e22c", logicalVolume.ID)
	assert.Equal(t, 2, len(logicalVolume.PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolume.RAIDLevel)
}
