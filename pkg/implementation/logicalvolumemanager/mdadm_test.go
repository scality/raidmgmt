//nolint:dupl // Had to duplicate test's code for some to dodge some difficulties with the mocking
package logicalvolumemanager_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumemanager"
)

type (
	MockCommandRunner struct {
		mock.Mock
	}

	MockLogicalVolumesGetter struct {
		mock.Mock
	}

	MockPhysicalDrivesGetter struct {
		mock.Mock
	}
)

func (m *MockPhysicalDrivesGetter) PhysicalDrive(metadata *physicaldrive.Metadata) (*physicaldrive.PhysicalDrive, error) {
	arguments := m.Called(metadata)

	return arguments.Get(0).(*physicaldrive.PhysicalDrive), arguments.Error(1)
}

func (m *MockPhysicalDrivesGetter) PhysicalDrives(metadata *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error) {
	arguments := m.Called(metadata)

	return arguments.Get(0).([]*physicaldrive.PhysicalDrive), arguments.Error(1)
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)

	return arguments.Get(0).([]byte), arguments.Error(1)
}

func (m *MockLogicalVolumesGetter) LogicalVolumes(metadata *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error) {
	arguments := m.Called(metadata)

	return arguments.Get(0).([]*logicalvolume.LogicalVolume), arguments.Error(1)
}

func (m *MockLogicalVolumesGetter) LogicalVolume(metadata *logicalvolume.Metadata) (*logicalvolume.LogicalVolume, error) {
	arguments := m.Called(metadata)

	return arguments.Get(0).(*logicalvolume.LogicalVolume), arguments.Error(1)
}

func TestMDADM_CreateLV_RAID1(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	request := &logicalvolume.Request{
		Name:      "testlv1",
		RAIDLevel: logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}

	// Mock physical drive checks
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"},
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"},
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)

	// Mock command execution
	mockCommandRunner.On("Run", []string{
		"--create", "/dev/md/testlv1",
		"--level", "1",
		"--raid-devices", "2",
		"--metadata=0.90",
		"/dev/nvme1n1", "/dev/nvme2n1",
	}).Return([]byte(""), nil)

	// Mock logical volume retrieval after creation
	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "/dev/md/testlv1"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "/dev/md/testlv1"},
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	lv, err := mdadm.CreateLV(request)
	assert.Nil(t, err)
	assert.NotNil(t, lv)
	assert.Equal(t, "/dev/md/testlv1", lv.DevicePath)
	assert.Equal(t, logicalvolume.RAIDLevel1, lv.RAIDLevel)
	assert.Equal(t, 2, len(lv.PDrivesMetadata))

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_CreateLV_RAID0(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	request := &logicalvolume.Request{
		Name:      "testlv0",
		RAIDLevel: logicalvolume.RAIDLevel0,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}

	// Mock physical drive checks
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"},
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"},
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)

	// Mock command execution
	mockCommandRunner.On("Run", []string{
		"--create", "/dev/md/testlv0",
		"--level", "0",
		"--raid-devices", "2",
		"/dev/nvme1n1", "/dev/nvme2n1",
	}).Return([]byte(""), nil)

	// Mock logical volume retrieval after creation
	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "/dev/md/testlv0"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "/dev/md/testlv0"},
		DevicePath: "/dev/md/testlv0",
		RAIDLevel:  logicalvolume.RAIDLevel0,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	lv, err := mdadm.CreateLV(request)
	assert.Nil(t, err)
	assert.NotNil(t, lv)
	assert.Equal(t, "/dev/md/testlv0", lv.DevicePath)
	assert.Equal(t, logicalvolume.RAIDLevel0, lv.RAIDLevel)

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_CreateLV_RAID10(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	request := &logicalvolume.Request{
		Name:      "testlv10",
		RAIDLevel: logicalvolume.RAIDLevel10,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
			{DevicePath: "/dev/nvme4n1"},
		},
	}

	// Mock physical drive checks for all drives
	for _, path := range []string{"/dev/nvme1n1", "/dev/nvme2n1", "/dev/nvme3n1", "/dev/nvme4n1"} {
		mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: path}).Return(&physicaldrive.PhysicalDrive{
			Metadata: &physicaldrive.Metadata{DevicePath: path},
			Status:   physicaldrive.PDStatusUnassignedGood,
		}, nil)
	}

	// Mock command execution
	mockCommandRunner.On("Run", []string{
		"--create", "/dev/md/testlv10",
		"--level", "10",
		"--raid-devices", "4",
		"/dev/nvme1n1", "/dev/nvme2n1", "/dev/nvme3n1", "/dev/nvme4n1",
	}).Return([]byte(""), nil)

	// Mock logical volume retrieval after creation
	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "/dev/md/testlv10"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "/dev/md/testlv10"},
		DevicePath: "/dev/md/testlv10",
		RAIDLevel:  logicalvolume.RAIDLevel10,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
			{DevicePath: "/dev/nvme4n1"},
		},
	}, nil)

	lv, err := mdadm.CreateLV(request)
	assert.Nil(t, err)
	assert.NotNil(t, lv)
	assert.Equal(t, "/dev/md/testlv10", lv.DevicePath)
	assert.Equal(t, logicalvolume.RAIDLevel10, lv.RAIDLevel)
	assert.Equal(t, 4, len(lv.PDrivesMetadata))

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_CreateLV_FailedDrive(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	request := &logicalvolume.Request{
		Name:      "testlv",
		RAIDLevel: logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}

	// Mock physical drive checks - one drive is failed
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"},
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"},
		Status:   physicaldrive.PDStatusFailed,
	}, nil)

	lv, err := mdadm.CreateLV(request)
	assert.Nil(t, lv)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot create a logical volume with a failed physical drive")

	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_CreateLV_UsedDrive(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	request := &logicalvolume.Request{
		Name:      "testlv",
		RAIDLevel: logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}

	// Mock physical drive checks - one drive is already used
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"},
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"},
		Status:   physicaldrive.PDStatusUsed,
	}, nil)

	lv, err := mdadm.CreateLV(request)
	assert.Nil(t, lv)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot create a logical volume with a used physical drive")

	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_CreateLV_CommandError(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	request := &logicalvolume.Request{
		Name:      "testlv",
		RAIDLevel: logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}

	// Mock physical drive checks
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"},
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)
	mockPhysicalDrivesGetter.On("PhysicalDrive", &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}).Return(&physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"},
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)

	// Mock command execution with error
	mockCommandRunner.On("Run", []string{
		"--create", "/dev/md/testlv",
		"--level", "1",
		"--raid-devices", "2",
		"--metadata=0.90",
		"/dev/nvme1n1", "/dev/nvme2n1",
	}).Return([]byte(""), errors.New("command execution failed"))

	lv, err := mdadm.CreateLV(request)
	assert.Nil(t, lv)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "command execution failed")

	mockCommandRunner.AssertExpectations(t)
	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_DeleteLV_Success(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "/dev/md/testlv"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "/dev/md/testlv"},
		DevicePath: "/dev/md/testlv",
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	// Mock stop command
	mockCommandRunner.On("Run", []string{
		"--stop", "/dev/md/testlv",
	}).Return([]byte(""), nil)

	// Mock zero-superblock commands for each drive
	mockCommandRunner.On("Run", []string{
		"--zero-superblock", "/dev/nvme1n1",
	}).Return([]byte(""), nil)

	mockCommandRunner.On("Run", []string{
		"--zero-superblock", "/dev/nvme2n1",
	}).Return([]byte(""), nil)

	err := mdadm.DeleteLV(&logicalvolume.Metadata{ID: "/dev/md/testlv"})
	assert.Nil(t, err)

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeleteLV_GetLVError(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	// Mock logical volume retrieval with error
	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "/dev/md/testlv"}).Return(
		(*logicalvolume.LogicalVolume)(nil), errors.New("logical volume not found"))

	err := mdadm.DeleteLV(&logicalvolume.Metadata{ID: "/dev/md/testlv"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "logical volume not found")

	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeleteLV_StopError(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "/dev/md/testlv"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "/dev/md/testlv"},
		DevicePath: "/dev/md/testlv",
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	// Mock stop command with error
	mockCommandRunner.On("Run", []string{
		"--stop", "/dev/md/testlv",
	}).Return([]byte(""), errors.New("failed to stop logical volume"))

	err := mdadm.DeleteLV(&logicalvolume.Metadata{ID: "/dev/md/testlv"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to stop logical volume")

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeleteLV_ZeroSuperblockError(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "/dev/md/testlv"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "/dev/md/testlv"},
		DevicePath: "/dev/md/testlv",
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	// Mock stop command
	mockCommandRunner.On("Run", []string{
		"--stop", "/dev/md/testlv",
	}).Return([]byte(""), nil)

	// First zero-superblock succeeds
	mockCommandRunner.On("Run", []string{
		"--zero-superblock", "/dev/nvme1n1",
	}).Return([]byte(""), nil)

	// Second zero-superblock fails
	mockCommandRunner.On("Run", []string{
		"--zero-superblock", "/dev/nvme2n1",
	}).Return([]byte(""), errors.New("failed to zero superblock"))

	err := mdadm.DeleteLV(&logicalvolume.Metadata{ID: "/dev/md/testlv"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to zero superblock")

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeleteLV_FailedLogicalVolume(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	// Mock logical volume retrieval with failed status
	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "/dev/md/testlv"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "/dev/md/testlv"},
		DevicePath: "/dev/md/testlv",
		Status:     logicalvolume.LVStatusFailed,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	// Mock stop command
	mockCommandRunner.On("Run", []string{
		"--stop", "/dev/md/testlv",
	}).Return([]byte(""), nil)

	// Mock zero-superblock commands for each drive
	mockCommandRunner.On("Run", []string{
		"--zero-superblock", "/dev/nvme1n1",
	}).Return([]byte(""), nil)

	mockCommandRunner.On("Run", []string{
		"--zero-superblock", "/dev/nvme2n1",
	}).Return([]byte(""), nil)

	err := mdadm.DeleteLV(&logicalvolume.Metadata{ID: "/dev/md/testlv"})
	assert.Nil(t, err)

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_AddPDsToLV_RAID0_Success(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv0"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv0",
		RAIDLevel:  logicalvolume.RAIDLevel0,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	// Mock physical drive check
	mockPhysicalDrivesGetter.On("PhysicalDrive", pdMetadata).Return(&physicaldrive.PhysicalDrive{
		Metadata: pdMetadata,
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)

	// Mock combined add/grow command for RAID0
	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md/testlv0",
		"--level", "0",
		"--raid-devices", "3",
		"--add",
		"/dev/nvme3n1",
	}).Return([]byte(""), nil)

	err := mdadm.AddPDsToLV(lvMetadata, pdMetadata)
	assert.Nil(t, err)

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_AddPDsToLV_RAID1_Success(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	// Mock physical drive check
	mockPhysicalDrivesGetter.On("PhysicalDrive", pdMetadata).Return(&physicaldrive.PhysicalDrive{
		Metadata: pdMetadata,
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)

	// For RAID1, first add the device
	mockCommandRunner.On("Run", []string{
		"/dev/md/testlv1",
		"--add",
		"/dev/nvme3n1",
	}).Return([]byte(""), nil)

	// Then grow the array size
	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md/testlv1",
		"--array-size=max",
	}).Return([]byte(""), nil)

	err := mdadm.AddPDsToLV(lvMetadata, pdMetadata)
	assert.Nil(t, err)

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_AddPDsToLV_RAID10_Success(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv10"}
	pdMetadata1 := &physicaldrive.Metadata{DevicePath: "/dev/nvme5n1"}
	pdMetadata2 := &physicaldrive.Metadata{DevicePath: "/dev/nvme6n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv10",
		RAIDLevel:  logicalvolume.RAIDLevel10,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
			{DevicePath: "/dev/nvme4n1"},
		},
	}, nil)

	// Mock physical drive checks
	mockPhysicalDrivesGetter.On("PhysicalDrive", pdMetadata1).Return(&physicaldrive.PhysicalDrive{
		Metadata: pdMetadata1,
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)
	mockPhysicalDrivesGetter.On("PhysicalDrive", pdMetadata2).Return(&physicaldrive.PhysicalDrive{
		Metadata: pdMetadata2,
		Status:   physicaldrive.PDStatusUnassignedGood,
	}, nil)

	// For RAID10, first add the devices
	mockCommandRunner.On("Run", []string{
		"/dev/md/testlv10",
		"--add",
		"/dev/nvme5n1", "/dev/nvme6n1",
	}).Return([]byte(""), nil)

	// Then grow the array size
	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md/testlv10",
		"--array-size=max",
	}).Return([]byte(""), nil)

	err := mdadm.AddPDsToLV(lvMetadata, pdMetadata1, pdMetadata2)
	assert.Nil(t, err)

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_AddPDsToLV_FailedLV(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"}

	// Mock logical volume retrieval with failed status
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		Status:     logicalvolume.LVStatusFailed,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	err := mdadm.AddPDsToLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot add physical drives to a failed logical volume")

	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_AddPDsToLV_FailedDrive(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	// Mock physical drive check with failed status
	mockPhysicalDrivesGetter.On("PhysicalDrive", pdMetadata).Return(&physicaldrive.PhysicalDrive{
		Metadata: pdMetadata,
		Status:   physicaldrive.PDStatusFailed,
	}, nil)

	err := mdadm.AddPDsToLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot add a failed physical drive to a logical volume")

	mockLogicalVolumeGetter.AssertExpectations(t)
	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_AddPDsToLV_UsedDrive(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	// Mock physical drive check with used status
	mockPhysicalDrivesGetter.On("PhysicalDrive", pdMetadata).Return(&physicaldrive.PhysicalDrive{
		Metadata: pdMetadata,
		Status:   physicaldrive.PDStatusUsed,
	}, nil)

	err := mdadm.AddPDsToLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot add a used physical drive to a logical volume")

	mockLogicalVolumeGetter.AssertExpectations(t)
	mockPhysicalDrivesGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_RAID1_Success(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
		},
	}, nil)

	// Mock marking the drive as failed
	mockCommandRunner.On("Run", []string{
		"--fail",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	// Mock removing the drive
	mockCommandRunner.On("Run", []string{
		"--remove",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	// Mock zeroing the superblock
	mockCommandRunner.On("Run", []string{
		"--zero-superblock",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	// Mock reducing device count
	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md/testlv1",
		"--raid-devices", "2",
	}).Return([]byte(""), nil)

	// Mock reducing array size
	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md/testlv1",
		"--array-size=max",
	}).Return([]byte(""), nil)

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata)
	assert.Nil(t, err)

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_RAID10_Success(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv10"}
	pdMetadata1 := &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"}
	pdMetadata2 := &physicaldrive.Metadata{DevicePath: "/dev/nvme4n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv10",
		RAIDLevel:  logicalvolume.RAIDLevel10,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
			{DevicePath: "/dev/nvme4n1"},
		},
	}, nil)

	// Mock marking the drives as failed
	mockCommandRunner.On("Run", []string{
		"--fail",
		"/dev/md/testlv10",
		"/dev/nvme3n1", "/dev/nvme4n1",
	}).Return([]byte(""), nil)

	// Mock removing the drives
	mockCommandRunner.On("Run", []string{
		"--remove",
		"/dev/md/testlv10",
		"/dev/nvme3n1", "/dev/nvme4n1",
	}).Return([]byte(""), nil)

	// Mock zeroing the superblocks
	mockCommandRunner.On("Run", []string{
		"--zero-superblock",
		"/dev/nvme3n1", "/dev/nvme4n1",
	}).Return([]byte(""), nil)

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata1, pdMetadata2)
	assert.Nil(t, err)

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_RAID0_Fail(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv0"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv0",
		RAIDLevel:  logicalvolume.RAIDLevel0,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot remove physical drives from a RAID0")

	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_RAID1_MinimumDisksFail(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}

	// Mock logical volume retrieval with only 2 drives
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot remove physical drives from a RAID1 with a single physical drive")

	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_FailedLV(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}

	// Mock logical volume retrieval with failed status
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		Status:     logicalvolume.LVStatusFailed,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
		},
	}, nil)

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot remove physical drives from a failed logical volume")

	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_FailCommandError(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
		},
	}, nil)

	// Mock marking the drive as failed with error
	mockCommandRunner.On("Run", []string{
		"--fail",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), errors.New("failed to mark drive as failed"))

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to run mdadm fail physical drive command")

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_RemoveCommandError(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
		},
	}, nil)

	// Mock marking the drive as failed successfully
	mockCommandRunner.On("Run", []string{
		"--fail",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	// Mock removing the drive with error
	mockCommandRunner.On("Run", []string{
		"--remove",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), errors.New("failed to remove drive"))

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to run mdadm remove command")

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_ZeroSuperblockError(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
		},
	}, nil)

	// Mock marking the drive as failed
	mockCommandRunner.On("Run", []string{
		"--fail",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	// Mock removing the drive
	mockCommandRunner.On("Run", []string{
		"--remove",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	// Mock zeroing the superblock with error
	mockCommandRunner.On("Run", []string{
		"--zero-superblock",
		"/dev/nvme2n1",
	}).Return([]byte(""), errors.New("failed to zero superblock"))

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to run mdadm zero superblock command")

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_GrowDevicesError(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
		},
	}, nil)

	// Mock commands up to grow devices
	mockCommandRunner.On("Run", []string{
		"--fail",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	mockCommandRunner.On("Run", []string{
		"--remove",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	mockCommandRunner.On("Run", []string{
		"--zero-superblock",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	// Mock reducing device count with error
	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md/testlv1",
		"--raid-devices", "2",
	}).Return([]byte(""), errors.New("failed to grow array"))

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to run mdadm grow command")

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}

func TestMDADM_DeletePDsFromLV_GrowArraySizeError(t *testing.T) {
	mockCommandRunner := new(MockCommandRunner)
	mockLogicalVolumeGetter := new(MockLogicalVolumesGetter)
	mockPhysicalDrivesGetter := new(MockPhysicalDrivesGetter)

	mdadm := logicalvolumemanager.NewMDADM(mockCommandRunner, mockLogicalVolumeGetter, mockPhysicalDrivesGetter)

	lvMetadata := &logicalvolume.Metadata{ID: "/dev/md/testlv1"}
	pdMetadata := &physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"}

	// Mock logical volume retrieval
	mockLogicalVolumeGetter.On("LogicalVolume", lvMetadata).Return(&logicalvolume.LogicalVolume{
		Metadata:   lvMetadata,
		DevicePath: "/dev/md/testlv1",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
			{DevicePath: "/dev/nvme3n1"},
		},
	}, nil)

	// Mock all previous commands
	mockCommandRunner.On("Run", []string{
		"--fail",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	mockCommandRunner.On("Run", []string{
		"--remove",
		"/dev/md/testlv1",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	mockCommandRunner.On("Run", []string{
		"--zero-superblock",
		"/dev/nvme2n1",
	}).Return([]byte(""), nil)

	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md/testlv1",
		"--raid-devices", "2",
	}).Return([]byte(""), nil)

	// Mock reducing array size with error
	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md/testlv1",
		"--array-size=max",
	}).Return([]byte(""), errors.New("failed to adjust array size"))

	err := mdadm.DeletePDsFromLV(lvMetadata, pdMetadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to run mdadm grow command to adapt array size")

	mockCommandRunner.AssertExpectations(t)
	mockLogicalVolumeGetter.AssertExpectations(t)
}
