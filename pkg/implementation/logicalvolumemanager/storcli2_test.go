package logicalvolumemanager_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumemanager"
)

// storcli2Fixture reads a storcli2 JSON fixture from the package testdata.
func storcli2Fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile("testdata/storcli2/" + name)
	require.NoError(t, err)

	return data
}

// storcli2Ctrl is the controller every storcli2 manager fixture is scoped to.
func storcli2Ctrl() *raidcontroller.Metadata {
	return &raidcontroller.Metadata{ID: 0}
}

// storcli2CreateRequest builds a two-drive RAID1 create request on the given
// controller; both drives share enclosure 252.
func storcli2CreateRequest(ctrl *raidcontroller.Metadata) *logicalvolume.Request {
	return &logicalvolume.Request{
		CtrlMetadata: ctrl,
		RAIDLevel:    logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{CtrlMetadata: ctrl, ID: "252:0"},
			{CtrlMetadata: ctrl, ID: "252:1"},
		},
		CacheOptions: &logicalvolume.CacheOptions{
			WritePolicy: logicalvolume.WritePolicyWriteBack,
			ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
		},
	}
}

// storcli2AvailablePD returns an unassigned-good drive of uniform size so
// ValidateRAIDCreation accepts it.
func storcli2AvailablePD(metadata *physicaldrive.Metadata) *physicaldrive.PhysicalDrive {
	return &physicaldrive.PhysicalDrive{
		Metadata: metadata,
		Size:     1000,
		Status:   physicaldrive.PDStatusUnassignedGood,
	}
}

// TestStorCLI2CreateLV covers the happy path: each request drive is filled and
// validated, "add vd" is issued with the bare RAID-level and cache tokens and a
// single-enclosure drive list, and the new volume is rediscovered by its
// physical-drive set.
func TestStorCLI2CreateLV(t *testing.T) {
	t.Parallel()

	ctrl := storcli2Ctrl()
	request := storcli2CreateRequest(ctrl)

	mockRunner := new(MockCommandRunner)
	mockPDGetter := new(MockPhysicalDrivesGetter)
	mockLVGetter := new(MockLogicalVolumesGetter)

	for _, pdMetadata := range request.PDrivesMetadata {
		mockPDGetter.On("PhysicalDrive", pdMetadata).Return(storcli2AvailablePD(pdMetadata), nil)
	}

	mockRunner.On("Run", []string{"/c0", "add", "vd", "r1", "drives=252:0,1", "wb", "ra"}).
		Return(storcli2Fixture(t, "create/success.json"), nil)

	newLV := &logicalvolume.LogicalVolume{
		Metadata:        &logicalvolume.Metadata{CtrlMetadata: ctrl, ID: "25"},
		PDrivesMetadata: request.PDrivesMetadata,
	}
	other := &logicalvolume.LogicalVolume{
		Metadata:        &logicalvolume.Metadata{CtrlMetadata: ctrl, ID: "7"},
		PDrivesMetadata: []*physicaldrive.Metadata{{CtrlMetadata: ctrl, ID: "252:9"}},
	}
	mockLVGetter.On("LogicalVolumes", ctrl).
		Return([]*logicalvolume.LogicalVolume{other, newLV}, nil)

	manager := logicalvolumemanager.NewStorCLI2(mockRunner, mockPDGetter, mockLVGetter)

	lv, err := manager.CreateLV(request)
	require.NoError(t, err)
	require.Equal(t, "25", lv.ID)
	mockRunner.AssertExpectations(t)
}

// TestStorCLI2CreateLVCommandError pins that an "add vd" failure payload aborts
// before any volume discovery.
func TestStorCLI2CreateLVCommandError(t *testing.T) {
	t.Parallel()

	ctrl := storcli2Ctrl()
	request := storcli2CreateRequest(ctrl)

	mockRunner := new(MockCommandRunner)
	mockPDGetter := new(MockPhysicalDrivesGetter)
	mockLVGetter := new(MockLogicalVolumesGetter)

	for _, pdMetadata := range request.PDrivesMetadata {
		mockPDGetter.On("PhysicalDrive", pdMetadata).Return(storcli2AvailablePD(pdMetadata), nil)
	}

	mockRunner.On("Run", []string{"/c0", "add", "vd", "r1", "drives=252:0,1", "wb", "ra"}).
		Return(storcli2Fixture(t, "create/fail.json"), nil)

	manager := logicalvolumemanager.NewStorCLI2(mockRunner, mockPDGetter, mockLVGetter)

	_, err := manager.CreateLV(request)
	require.Error(t, err)
	mockLVGetter.AssertNotCalled(t, "LogicalVolumes")
}

// TestStorCLI2CreateLVMultipleEnclosures pins that a request spanning two
// enclosures is rejected before "add vd" is run.
func TestStorCLI2CreateLVMultipleEnclosures(t *testing.T) {
	t.Parallel()

	ctrl := storcli2Ctrl()
	request := storcli2CreateRequest(ctrl)
	request.PDrivesMetadata[1].ID = "253:1"

	mockRunner := new(MockCommandRunner)
	mockPDGetter := new(MockPhysicalDrivesGetter)
	mockLVGetter := new(MockLogicalVolumesGetter)

	for _, pdMetadata := range request.PDrivesMetadata {
		mockPDGetter.On("PhysicalDrive", pdMetadata).Return(storcli2AvailablePD(pdMetadata), nil)
	}

	manager := logicalvolumemanager.NewStorCLI2(mockRunner, mockPDGetter, mockLVGetter)

	_, err := manager.CreateLV(request)
	require.Error(t, err)
	mockRunner.AssertNotCalled(t, "Run")
}

// TestStorCLI2CreateLVUnsettableCachePolicy pins that a set but unmappable cache
// policy is rejected (the mapping fails closed) before "add vd" is run, rather
// than emitting the raw value as a token.
func TestStorCLI2CreateLVUnsettableCachePolicy(t *testing.T) {
	t.Parallel()

	ctrl := storcli2Ctrl()
	request := storcli2CreateRequest(ctrl)
	request.CacheOptions.WritePolicy = logicalvolume.WritePolicy("bogus")

	mockRunner := new(MockCommandRunner)
	mockPDGetter := new(MockPhysicalDrivesGetter)
	mockLVGetter := new(MockLogicalVolumesGetter)

	for _, pdMetadata := range request.PDrivesMetadata {
		mockPDGetter.On("PhysicalDrive", pdMetadata).Return(storcli2AvailablePD(pdMetadata), nil)
	}

	manager := logicalvolumemanager.NewStorCLI2(mockRunner, mockPDGetter, mockLVGetter)

	_, err := manager.CreateLV(request)
	require.Error(t, err)
	mockRunner.AssertNotCalled(t, "Run")
}

// TestStorCLI2DeleteLV covers the delete happy path and the two documented
// failure payloads (an invalid VD number and a nonexistent VD).
func TestStorCLI2DeleteLV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		fixture string
		wantErr bool
	}{
		{name: "success", id: "25", fixture: "delete/success.json", wantErr: false},
		{name: "invalid", id: "299", fixture: "delete/fail_invalid.json", wantErr: true},
		{name: "not exist", id: "999", fixture: "delete/fail_vdNotExist.json", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := storcli2Ctrl()
			metadata := &logicalvolume.Metadata{CtrlMetadata: ctrl, ID: tt.id}

			mockRunner := new(MockCommandRunner)
			mockRunner.On("Run", []string{"/c0/v" + tt.id, "delete"}).
				Return(storcli2Fixture(t, tt.fixture), nil)

			manager := logicalvolumemanager.NewStorCLI2(
				mockRunner, new(MockPhysicalDrivesGetter), new(MockLogicalVolumesGetter),
			)

			err := manager.DeleteLV(metadata)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			mockRunner.AssertExpectations(t)
		})
	}
}

// TestStorCLI2AddPDsToLV covers the expand happy path: the drives are formatted
// into a single-enclosure list and submitted through "expand".
func TestStorCLI2AddPDsToLV(t *testing.T) {
	t.Parallel()

	ctrl := storcli2Ctrl()
	metadata := &logicalvolume.Metadata{CtrlMetadata: ctrl, ID: "25"}
	pds := []*physicaldrive.Metadata{
		{CtrlMetadata: ctrl, ID: "252:3"},
		{CtrlMetadata: ctrl, ID: "252:4"},
	}

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/v25", "expand", "drives=252:3,4"}).
		Return(storcli2Fixture(t, "expand/success.json"), nil)

	manager := logicalvolumemanager.NewStorCLI2(
		mockRunner, new(MockPhysicalDrivesGetter), new(MockLogicalVolumesGetter),
	)

	err := manager.AddPDsToLV(metadata, pds...)
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

// TestStorCLI2AddPDsToLVCommandError pins that an "expand" failure payload is
// surfaced.
func TestStorCLI2AddPDsToLVCommandError(t *testing.T) {
	t.Parallel()

	ctrl := storcli2Ctrl()
	metadata := &logicalvolume.Metadata{CtrlMetadata: ctrl, ID: "25"}
	pd := &physicaldrive.Metadata{CtrlMetadata: ctrl, ID: "252:3"}

	mockRunner := new(MockCommandRunner)
	mockRunner.On("Run", []string{"/c0/v25", "expand", "drives=252:3"}).
		Return(storcli2Fixture(t, "expand/fail.json"), nil)

	manager := logicalvolumemanager.NewStorCLI2(
		mockRunner, new(MockPhysicalDrivesGetter), new(MockLogicalVolumesGetter),
	)

	err := manager.AddPDsToLV(metadata, pd)
	require.Error(t, err)
}

// TestStorCLI2AddPDsToLVMultipleEnclosures pins that drives spanning two
// enclosures are rejected before "expand" is run.
func TestStorCLI2AddPDsToLVMultipleEnclosures(t *testing.T) {
	t.Parallel()

	ctrl := storcli2Ctrl()
	metadata := &logicalvolume.Metadata{CtrlMetadata: ctrl, ID: "25"}
	pds := []*physicaldrive.Metadata{
		{CtrlMetadata: ctrl, ID: "252:3"},
		{CtrlMetadata: ctrl, ID: "253:4"},
	}

	mockRunner := new(MockCommandRunner)

	manager := logicalvolumemanager.NewStorCLI2(
		mockRunner, new(MockPhysicalDrivesGetter), new(MockLogicalVolumesGetter),
	)

	err := manager.AddPDsToLV(metadata, pds...)
	require.Error(t, err)
	mockRunner.AssertNotCalled(t, "Run")
}

// TestStorCLI2DeletePDsFromLV pins that drive removal is reported as unsupported:
// storcli2 has no replacement for storcli's "start migrate option=remove".
func TestStorCLI2DeletePDsFromLV(t *testing.T) {
	t.Parallel()

	ctrl := storcli2Ctrl()
	metadata := &logicalvolume.Metadata{CtrlMetadata: ctrl, ID: "25"}
	pd := &physicaldrive.Metadata{CtrlMetadata: ctrl, ID: "252:3"}

	mockRunner := new(MockCommandRunner)

	manager := logicalvolumemanager.NewStorCLI2(
		mockRunner, new(MockPhysicalDrivesGetter), new(MockLogicalVolumesGetter),
	)

	err := manager.DeletePDsFromLV(metadata, pd)
	require.ErrorIs(t, err, ports.ErrFunctionNotSupportedByImplementation)
	mockRunner.AssertNotCalled(t, "Run")
}
