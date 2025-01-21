package logicalvolumemanager_test

import (
	"fmt"
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
						"--force",
						"/dev/nvme1n1",
						"/dev/nvme2n1",
					},
					returnValues: []any{[]byte(""), nil},
				},
				{
					mocker:  mockLogicalVolumeGetter,
					command: "LogicalVolumes",
					parameters: &raidcontroller.Metadata{
						ID: 1,
					},
					returnValues: []any{[]*logicalvolume.LogicalVolume{}, nil},
				},
				{
					mocker:  mockLogicalVolumeGetter,
					command: "LogicalVolume",
					parameters: &logicalvolume.Metadata{
						ID: "/dev/md0",
					},
					returnValues: []any{&logicalvolume.LogicalVolume{
						ID:         "/dev/md0",
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
				CtrlMetadata: controllerMetadata,
				RAIDLevel:    logicalvolume.RAIDLevel0,
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
					mocker:       mockLogicalVolumeGetter,
					command:      "LogicalVolume",
					parameters:   &logicalvolume.Metadata{ID: "md0"},
					returnValues: []any{&logicalvolume.LogicalVolume{ID: "md0", DevicePath: "/dev/md0"}, nil},
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
			input:       &logicalvolume.Metadata{ID: "md0"},
			expectError: false,
		},
		{
			name: "Logical volume deletion request: remove volume failed, need zero superblock",
			mockings: []mocking{
				{
					mocker:       mockLogicalVolumeGetter,
					command:      "LogicalVolume",
					parameters:   &logicalvolume.Metadata{ID: "md0"},
					returnValues: []any{&logicalvolume.LogicalVolume{ID: "md0", DevicePath: "/dev/md0"}, nil},
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
			input:       &logicalvolume.Metadata{ID: "md0"},
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

func TestMDADM_AddPVToLV(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	type input struct {
		lv *logicalvolume.Metadata
		pv *physicaldrive.Metadata
	}

	testCases := []struct {
		name        string
		mockings    []mocking
		input       input
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
		{
			name: "Add physical drive to logical volume",
			mockings: []mocking{
				{
					mocker:     mockLogicalVolumeGetter,
					command:    "LogicalVolume",
					parameters: &logicalvolume.Metadata{ID: "md0"},
					returnValues: []any{
						&logicalvolume.LogicalVolume{
							ID:         "md0",
							DevicePath: "/dev/md0",
							RAIDLevel:  0,
							PDrivesMetadata: []*physicaldrive.Metadata{
								{
									DevicePath: "/dev/nvme1n1",
								},
								{
									DevicePath: "/dev/nvme2n1",
								},
							},
						}, nil,
					},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--add",
						"/dev/md0",
						"/dev/nvme1n1",
					},
					returnValues: []any{[]byte(""), nil},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--grow",
						"/dev/md0",
						"--raid-devices", fmt.Sprintf("%d", 2+1), // 2 existing drives + 1 new drive, cf l336
					},
					returnValues: []any{[]byte(""), nil},
				},
			},
			input: input{
				lv: &logicalvolume.Metadata{ID: "md0"},
				pv: &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Log(tc.name)

		for _, m := range tc.mockings {
			t.Log("mockings to apply: ", m.command, m.parameters, m.returnValues)
			m.mocker.On(m.command, m.parameters).Return(m.returnValues...)
		}

		mdadm := &logicalvolumemanager.MDADM{
			CommandRunner:        mockCommandRunner,
			LogicalVolumesGetter: mockLogicalVolumeGetter,
		}

		err := mdadm.AddPDToLV(tc.input.lv, tc.input.pv)
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

func TestMDADM_DeletePVFromLV(t *testing.T) {
	mockCommandRunner, mockLogicalVolumeGetter := &MockCommandRunner{}, &MockLogicalVolumesGetter{}

	type input struct {
		lv *logicalvolume.Metadata
		pv *physicaldrive.Metadata
	}

	testCases := []struct {
		name        string
		mockings    []mocking
		input       input
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
		{
			name: "Remove physical drive from logical volume",
			mockings: []mocking{
				{
					mocker:     mockLogicalVolumeGetter,
					command:    "LogicalVolume",
					parameters: &logicalvolume.Metadata{ID: "md0"},
					returnValues: []any{
						&logicalvolume.LogicalVolume{
							ID:         "md0",
							DevicePath: "/dev/md0",
							RAIDLevel:  0,
							PDrivesMetadata: []*physicaldrive.Metadata{
								{
									DevicePath: "/dev/nvme1n1",
								},
								{
									DevicePath: "/dev/nvme2n1",
								},
							},
						}, nil,
					},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"/dev/md0",
						"--fail",
						"/dev/nvme1n1",
					},
					returnValues: []any{[]byte(""), nil},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--remove",
						"/dev/md0",
						"/dev/nvme1n1",
					},
					returnValues: []any{[]byte(""), nil},
				},
				{
					mocker:  mockCommandRunner,
					command: "Run",
					parameters: []string{
						"--grow",
						"/dev/md0",
						"--raid-devices", fmt.Sprintf("%d", 2-1), // 2 existing drives - 1 removed drive
					},
					returnValues: []any{[]byte(""), nil},
				},
			},
			input: input{
				lv: &logicalvolume.Metadata{ID: "md0"},
				pv: &physicaldrive.Metadata{DevicePath: "/dev/nvme1n1"},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Log(tc.name)

		for _, m := range tc.mockings {
			t.Log("mockings to apply: ", m.command, m.parameters, m.returnValues)
			m.mocker.On(m.command, m.parameters).Return(m.returnValues...)
		}

		mdadm := &logicalvolumemanager.MDADM{
			CommandRunner:        mockCommandRunner,
			LogicalVolumesGetter: mockLogicalVolumeGetter,
		}

		err := mdadm.DeletePDFromLV(tc.input.lv, tc.input.pv)
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
