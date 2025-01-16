package logicalvolumegetter_test

import (
	"logicalvolumegetter"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
)

const (
	mdadmExportOutput = `MD_LEVEL=raid1
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

func TestMDADMLogicalVolumesSingle(t *testing.T) {
	// // Create a mock object
	mockRunner := &MockCommandRunner{}

	// Set up expected behavior of the mock
	mockRunner.On("Run", []string{"--detail", "--scan", "--export"}).Return([]byte(mdadmExportOutput), nil)

	// Use the mock object in your test
	mdadm := &logicalvolumegetter.MDADM{CommandRunner: mockRunner}

	logicalVolumes, err := mdadm.LogicalVolumes(nil)
	for _, lv := range logicalVolumes {
		spew.Dump(lv)
	}

	assert.Nil(t, err)
	assert.Equal(t, 1, len(logicalVolumes))
	assert.Equal(t, "0030d06e:fd0fa07d:0d04737a:dc97e22c", logicalVolumes[0].ID)
	assert.Equal(t, 2, len(logicalVolumes[0].PDrivesMetadata))
	assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolumes[0].RAIDLevel)
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
