//nolint:dupl // Had to duplicate test's code for some to dodge some difficulties with the mocking
package logicalvolumemanager_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/rhel8/logicalvolumemanager"
)

type (
	MockCommandRunner struct {
		mock.Mock
	}

	MockLogicalVolumesGetter struct {
		mock.Mock
	}

	mocking struct {
		mocker
		command      string
		parameters   any
		returnValues []any
	}

	mocker interface {
		On(methodName string, arguments ...any) *mock.Call
	}
)

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

func TestMDADM_CreateLV(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	slot := &physicaldrive.Slot{
		Port:      "1",
		Enclosure: "1",
		Bay:       "1",
	}

	controllerMetadata := &raidcontroller.Metadata{
		ID: 1,
	}

	testCases := []struct {
		name        string
		mockings    []mocking
		input       *logicalvolume.Request
		expectError bool
	}{
		{
			name:        "Nil logical volume creation request",
			mockings:    []mocking{},
			input:       nil,
			expectError: true,
		},
		{
			name:        "Empty logical volume creation request",
			mockings:    []mocking{},
			input:       &logicalvolume.Request{},
			expectError: true,
		},
		{
			name: "Valid single logical volume output",
			mockings: []mocking{
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--create",
						"/dev/md0",
						"--level", "1",
						"--raid-devices", "2",
						"/dev/nvme1n1",
						"/dev/nvme2n1",
					},
					returnValues: []any{[]byte(""), nil},
				},
				{
					mocker:  mockLogicalVolumeGetter,
					command: "LogicalVolume",
					parameters: &logicalvolume.Metadata{
						ID: "0",
					},
					returnValues: []any{&logicalvolume.LogicalVolume{
						Metadata: &logicalvolume.Metadata{
							ID: "0",
						},
						DevicePath: "/dev/md0",
						RAIDLevel:  logicalvolume.RAIDLevel1,
						PDrivesMetadata: []*physicaldrive.Metadata{
							{
								DevicePath:   "/dev/nvme1n1",
								CtrlMetadata: controllerMetadata,
								Slot:         slot,
							},
							{
								DevicePath:   "/dev/nvme2n1",
								CtrlMetadata: controllerMetadata,
								Slot:         slot,
							},
						},
					}, nil},
				},
			},
			input: &logicalvolume.Request{
				Name:         "md0",
				ID:           "0",
				CtrlMetadata: controllerMetadata,
				RAIDLevel:    logicalvolume.RAIDLevel1,
				PDrivesMetadata: []*physicaldrive.Metadata{
					{
						DevicePath:   "/dev/nvme1n1",
						CtrlMetadata: controllerMetadata,
						Slot:         slot,
					},
					{
						DevicePath:   "/dev/nvme2n1",
						CtrlMetadata: controllerMetadata,
						Slot:         slot,
					},
				},
				CacheOptions: &logicalvolume.CacheOptions{},
			},
			expectError: false,
		},
	}

	mdadm := &logicalvolumemanager.MDADM{
		CommandRunner:        mockCommandRunner,
		LogicalVolumesGetter: mockLogicalVolumeGetter,
	}

	for _, tc := range testCases {
		t.Log(tc.name)

		for _, m := range tc.mockings {
			t.Log("mockings to apply: ", m.command, m.parameters, m.returnValues)
			m.mocker.On(m.command, m.parameters).Return(m.returnValues...)
		}

		logicalVolume, err := mdadm.CreateLV(tc.input)
		if tc.expectError {
			assert.Nil(t, logicalVolume)
			assert.NotNil(t, err)
		} else {
			assert.NotNil(t, logicalVolume)
			assert.Nil(t, err)
		}

		if tc.name == "Valid single logical volume output" {
			assert.Equal(t, "/dev/md0", logicalVolume.DevicePath)
			assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolume.RAIDLevel)
			assert.Equal(t, 2, len(logicalVolume.PDrivesMetadata))
		}

		t.Cleanup(func() {
			mockCommandRunner.AssertExpectations(t)
			mockLogicalVolumeGetter.AssertExpectations(t)
		})
	}
}

func TestMDADM_DeleteLV(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	testCases := []struct {
		name        string
		mockings    []mocking
		input       *logicalvolume.Metadata
		expectError bool
	}{
		{
			name:        "Nil logical volume deletion request",
			mockings:    []mocking{},
			input:       nil,
			expectError: true,
		},
		{
			name: "Logical volume deletion request: remove volume working",
			mockings: []mocking{
				{
					mocker:     mockLogicalVolumeGetter,
					command:    "LogicalVolume",
					parameters: &logicalvolume.Metadata{ID: "0"},
					returnValues: []any{&logicalvolume.LogicalVolume{
						Metadata:   &logicalvolume.Metadata{ID: "0"},
						DevicePath: "/dev/md0",
					}, nil},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--stop",
						"/dev/md0",
					},
					returnValues: []any{[]byte(""), nil},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--remove",
						"/dev/md0",
					},
					returnValues: []any{[]byte(""), nil},
				},
			},
			input:       &logicalvolume.Metadata{ID: "0"},
			expectError: false,
		},
		{
			name: "Logical volume deletion request: remove volume failed, need zero superblock",
			mockings: []mocking{
				{
					mocker:     mockLogicalVolumeGetter,
					command:    "LogicalVolume",
					parameters: &logicalvolume.Metadata{ID: "0"},
					returnValues: []any{&logicalvolume.LogicalVolume{
						Metadata:   &logicalvolume.Metadata{ID: "0"},
						DevicePath: "/dev/md0",
					}, nil},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--stop",
						"/dev/md0",
					},
					returnValues: []any{[]byte(""), nil},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--remove",
						"/dev/md0",
					},
					returnValues: []any{[]byte(""), errors.New("error")},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--zero-superblock",
						"/dev/md0",
					},
					returnValues: []any{[]byte(""), nil},
				},
			},
			input:       &logicalvolume.Metadata{ID: "0"},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Log(tc.name)

		for _, m := range tc.mockings {
			t.Log("mockings to apply: ", m.command, m.parameters, m.returnValues)
			m.mocker.On(m.command, m.parameters).Return(m.returnValues...).Maybe()
		}

		mdadm := &logicalvolumemanager.MDADM{
			CommandRunner:        mockCommandRunner,
			LogicalVolumesGetter: mockLogicalVolumeGetter,
		}

		err := mdadm.DeleteLV(tc.input)
		if tc.expectError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		t.Cleanup(func() {
			mockCommandRunner.AssertExpectations(t)
			mockLogicalVolumeGetter.AssertExpectations(t)
		})
	}
}

func TestMDADM_AddPVsToLV_1PhysicalDrive(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "0"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "0"},
		DevicePath: "/dev/md0",
		RAIDLevel:  logicalvolume.RAIDLevel0,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md0",
		"--level", "0",
		"--raid-devices", "3",
		"--add", "/dev/nvme3n1",
	}).Return([]byte(""), nil)

	mdadm := &logicalvolumemanager.MDADM{
		CommandRunner:        mockCommandRunner,
		LogicalVolumesGetter: mockLogicalVolumeGetter,
	}

	err := mdadm.AddPDsToLV(&logicalvolume.Metadata{ID: "0"}, &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"})
	assert.Nil(t, err)

	t.Cleanup(func() {
		mockCommandRunner.AssertExpectations(t)
		mockLogicalVolumeGetter.AssertExpectations(t)
	})
}

func TestMDADM_AddPDstoLV_NilLogicalVolumeMetadata(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	mdadm := &logicalvolumemanager.MDADM{
		CommandRunner:        mockCommandRunner,
		LogicalVolumesGetter: mockLogicalVolumeGetter,
	}

	err := mdadm.AddPDsToLV(nil, &physicaldrive.Metadata{})
	assert.NotNil(t, err)

	t.Cleanup(func() {
		mockCommandRunner.AssertExpectations(t)
		mockLogicalVolumeGetter.AssertExpectations(t)
	})
}

func TestMDADM_AddPDsToLV_NilDriveMetadata(t *testing.T) {
	t.Parallel()

	mdadm := &logicalvolumemanager.MDADM{}

	err := mdadm.AddPDsToLV(&logicalvolume.Metadata{}, nil)
	assert.NotNil(t, err)
}

func TestAddPhysicalDrivesToLogicalVolume(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "0"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "0"},
		DevicePath: "/dev/md0",
		RAIDLevel:  logicalvolume.RAIDLevel0,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md0",
		"--level", "0",
		"--raid-devices", "4",
		"--add", "/dev/nvme3n1", "/dev/nvme4n1",
	}).Return([]byte(""), nil)

	mdadm := &logicalvolumemanager.MDADM{
		CommandRunner:        mockCommandRunner,
		LogicalVolumesGetter: mockLogicalVolumeGetter,
	}

	err := mdadm.AddPDsToLV(&logicalvolume.Metadata{ID: "0"}, &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"}, &physicaldrive.Metadata{DevicePath: "/dev/nvme4n1"})
	assert.Nil(t, err)

	t.Cleanup(func() {
		mockCommandRunner.AssertExpectations(t)
		mockLogicalVolumeGetter.AssertExpectations(t)
	})
}

func TestAddPhysicalDrivesToLogicalVolumeRAID10_1Disk(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "0"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "0"},
		DevicePath: "/dev/md0",
		RAIDLevel:  logicalvolume.RAIDLevel10,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	mdadm := &logicalvolumemanager.MDADM{
		CommandRunner:        mockCommandRunner,
		LogicalVolumesGetter: mockLogicalVolumeGetter,
	}

	err := mdadm.AddPDsToLV(&logicalvolume.Metadata{ID: "0"}, &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"})
	assert.NotNil(t, err)
	assert.Equal(t, "cannot add an odd number of physical drives to a RAID10", err.Error())

	t.Cleanup(func() {
		mockCommandRunner.AssertExpectations(t)
		mockLogicalVolumeGetter.AssertExpectations(t)
	})
}

func TestAddPhysicalDrivesToLogicalVolumeRAID10_2Disk(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "0"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "0"},
		DevicePath: "/dev/md0",
		RAIDLevel:  logicalvolume.RAIDLevel10,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	mdadm := &logicalvolumemanager.MDADM{
		CommandRunner:        mockCommandRunner,
		LogicalVolumesGetter: mockLogicalVolumeGetter,
	}

	mockCommandRunner.On("Run", []string{
		"--grow", "/dev/md0",
		"--level", "10",
		"--raid-devices", "4",
		"--add", "/dev/nvme3n1", "/dev/nvme4n1",
	}).Return([]byte(""), nil)

	err := mdadm.AddPDsToLV(&logicalvolume.Metadata{ID: "0"}, &physicaldrive.Metadata{DevicePath: "/dev/nvme3n1"}, &physicaldrive.Metadata{DevicePath: "/dev/nvme4n1"})
	assert.Nil(t, err)

	t.Cleanup(func() {
		mockCommandRunner.AssertExpectations(t)
		mockLogicalVolumeGetter.AssertExpectations(t)
	})
}

func TestMDADM_DeletePVsFromLV_nil(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	type input struct {
		lv *logicalvolume.Metadata
		pv *physicaldrive.Metadata
	}

	testCases := []struct {
		name     string
		mockings []mocking
		input
		expectError bool
	}{
		{
			name:        "Nil logical volume metadata",
			mockings:    []mocking{},
			input:       input{lv: nil, pv: &physicaldrive.Metadata{}},
			expectError: true,
		},
		{
			name:        "Nil physical drive metadata",
			mockings:    []mocking{},
			input:       input{lv: &logicalvolume.Metadata{}, pv: nil},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Log(tc.name)

		for _, m := range tc.mockings {
			t.Log("mockings to apply: ", m.command, m.parameters, m.returnValues)

			m.mocker.On(m.command, m.parameters).Return(m.returnValues...).Maybe()
		}

		mdadm := &logicalvolumemanager.MDADM{
			CommandRunner:        mockCommandRunner,
			LogicalVolumesGetter: mockLogicalVolumeGetter,
		}

		err := mdadm.DeletePDsFromLV(tc.input.lv, tc.input.pv)
		if tc.expectError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		t.Cleanup(func() {
			mockCommandRunner.AssertExpectations(t)
			mockLogicalVolumeGetter.AssertExpectations(t)
		})
	}
}

func TestMDADM_DeletePDsFromLV_RAID1(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "0"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "0"},
		DevicePath: "/dev/md0",
		RAIDLevel:  logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	mockCommandRunner.On("Run", []string{
		"/dev/md0",
		"--fail",
		"/dev/nvme1n1",
	}).Return([]byte(""), nil)

	mockCommandRunner.On("Run", []string{
		"--remove",
		"/dev/md0",
		"/dev/nvme1n1",
	}).Return([]byte(""), nil)

	mockCommandRunner.On("Run", []string{
		"--grow",
		"/dev/md0",
		"--raid-devices", "1",
	}).Return([]byte(""), nil)

	mdadm := &logicalvolumemanager.MDADM{
		CommandRunner:        mockCommandRunner,
		LogicalVolumesGetter: mockLogicalVolumeGetter,
	}

	err := mdadm.DeletePDsFromLV(&logicalvolume.Metadata{ID: "0"}, &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"})
	assert.Nil(t, err)

	t.Cleanup(func() {
		mockCommandRunner.AssertExpectations(t)
		mockLogicalVolumeGetter.AssertExpectations(t)
	})
}

func TestMDADM_DeletePDsFromLV_RAID0(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "0"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "0"},
		DevicePath: "/dev/md0",
		RAIDLevel:  logicalvolume.RAIDLevel0,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	mdadm := &logicalvolumemanager.MDADM{
		CommandRunner:        mockCommandRunner,
		LogicalVolumesGetter: mockLogicalVolumeGetter,
	}

	err := mdadm.DeletePDsFromLV(&logicalvolume.Metadata{ID: "0"}, &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"})
	assert.NotNil(t, err)
	assert.Equal(t, "cannot remove physical drives from a RAID0", err.Error())

	t.Cleanup(func() {
		mockCommandRunner.AssertExpectations(t)
		mockLogicalVolumeGetter.AssertExpectations(t)
	})
}

func TestMDADM_DeletePDsFromLV_RAID10_2Disks(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	mockLogicalVolumeGetter.On("LogicalVolume", &logicalvolume.Metadata{ID: "0"}).Return(&logicalvolume.LogicalVolume{
		Metadata:   &logicalvolume.Metadata{ID: "0"},
		DevicePath: "/dev/md0",
		RAIDLevel:  logicalvolume.RAIDLevel10,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{DevicePath: "/dev/nvme1n1"},
			{DevicePath: "/dev/nvme2n1"},
		},
	}, nil)

	mdadm := &logicalvolumemanager.MDADM{
		CommandRunner:        mockCommandRunner,
		LogicalVolumesGetter: mockLogicalVolumeGetter,
	}

	err := mdadm.DeletePDsFromLV(
		&logicalvolume.Metadata{ID: "0"},
		&physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"},
		&physicaldrive.Metadata{DevicePath: "/dev/nvme2n1"},
	)
	assert.NotNil(t, err)
	assert.Equal(t, "cannot remove more than one physical drive from a RAID10", err.Error())

	t.Cleanup(func() {
		mockCommandRunner.AssertExpectations(t)
		mockLogicalVolumeGetter.AssertExpectations(t)
	})
}
