package lvcachesetter_test

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/lvcachesetter"
)

type (
	MockCommandRunner struct {
		mock.Mock
	}

	MockLogicalVolumesGetter struct {
		mock.Mock
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

// storcli2Fixture reads a storcli2 JSON fixture from the package testdata.
func storcli2Fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile("testdata/storcli2/" + name)
	require.NoError(t, err)

	return data
}

func storcli2Metadata() *logicalvolume.Metadata {
	return &logicalvolume.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{ID: 0},
		ID:           "25",
	}
}

func newStorCLI2LV(cache *logicalvolume.CacheOptions) *logicalvolume.LogicalVolume {
	return &logicalvolume.LogicalVolume{
		Metadata:     storcli2Metadata(),
		CacheOptions: cache,
	}
}

// TestStorCLI2SetLVCacheOptions covers the only-changed-flag behavior: each
// policy is set through its own command, and an unchanged policy emits no
// command at all.
func TestStorCLI2SetLVCacheOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current *logicalvolume.CacheOptions
		desired *logicalvolume.CacheOptions
		// calls maps the expected "set" option to the fixture it returns.
		calls map[string]string
	}{
		{
			name: "only read changed",
			current: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyNoReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			desired: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			calls: map[string]string{"rdcache=RA": "cacheoptions/success_rdcache.json"},
		},
		{
			name: "only write changed",
			current: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			desired: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteThrough,
			},
			calls: map[string]string{"wrcache=WT": "cacheoptions/success_wrcache.json"},
		},
		{
			name: "both changed",
			current: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyNoReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			desired: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteThrough,
			},
			calls: map[string]string{
				"rdcache=RA": "cacheoptions/success_rdcache.json",
				"wrcache=WT": "cacheoptions/success_wrcache.json",
			},
		},
		{
			name: "nothing changed",
			current: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteThrough,
			},
			desired: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteThrough,
			},
			calls: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockGetter := new(MockLogicalVolumesGetter)

			metadata := storcli2Metadata()
			mockGetter.On("LogicalVolume", metadata).Return(newStorCLI2LV(tt.current), nil)

			for option, fixture := range tt.calls {
				mockRunner.On("Run", []string{"/c0/v25", "set", option}).
					Return(storcli2Fixture(t, fixture), nil)
			}

			setter := lvcachesetter.NewStorCLI2(mockRunner, mockGetter)

			err := setter.SetLVCacheOptions(metadata, tt.desired)
			require.NoError(t, err)

			mockRunner.AssertExpectations(t)
			mockRunner.AssertNumberOfCalls(t, "Run", len(tt.calls))
		})
	}
}

// TestStorCLI2SetLVCacheOptionsGetterError pins that a failure to read the
// current state aborts before any command is run.
func TestStorCLI2SetLVCacheOptionsGetterError(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockGetter := new(MockLogicalVolumesGetter)

	metadata := storcli2Metadata()
	mockGetter.On("LogicalVolume", metadata).
		Return((*logicalvolume.LogicalVolume)(nil), errors.New("boom"))

	setter := lvcachesetter.NewStorCLI2(mockRunner, mockGetter)

	err := setter.SetLVCacheOptions(metadata, &logicalvolume.CacheOptions{
		ReadPolicy: logicalvolume.ReadPolicyReadAhead,
	})
	require.Error(t, err)
	mockRunner.AssertNotCalled(t, "Run")
}

// TestStorCLI2SetLVCacheOptionsCommandError pins that an in-JSON command
// failure (here storcli's rejected combined syntax, kept as a plain-text
// failure fixture) is surfaced.
func TestStorCLI2SetLVCacheOptionsCommandError(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockGetter := new(MockLogicalVolumesGetter)

	metadata := storcli2Metadata()
	mockGetter.On("LogicalVolume", metadata).Return(newStorCLI2LV(&logicalvolume.CacheOptions{
		ReadPolicy:  logicalvolume.ReadPolicyNoReadAhead,
		WritePolicy: logicalvolume.WritePolicyWriteBack,
	}), nil)
	mockRunner.On("Run", []string{"/c0/v25", "set", "rdcache=RA"}).
		Return(storcli2Fixture(t, "cacheoptions/combined_syntax_error.json"), nil)

	setter := lvcachesetter.NewStorCLI2(mockRunner, mockGetter)

	err := setter.SetLVCacheOptions(metadata, &logicalvolume.CacheOptions{
		ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
		WritePolicy: logicalvolume.WritePolicyWriteBack,
	})
	require.Error(t, err)
}

// TestStorCLI2SetLVCacheOptionsUnsettable pins that an unknown desired policy
// (e.g. round-tripped from getter output) is rejected rather than emitted.
func TestStorCLI2SetLVCacheOptionsUnsettable(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockGetter := new(MockLogicalVolumesGetter)

	metadata := storcli2Metadata()
	mockGetter.On("LogicalVolume", metadata).Return(newStorCLI2LV(&logicalvolume.CacheOptions{
		ReadPolicy: logicalvolume.ReadPolicyReadAhead,
	}), nil)

	setter := lvcachesetter.NewStorCLI2(mockRunner, mockGetter)

	err := setter.SetLVCacheOptions(metadata, &logicalvolume.CacheOptions{
		ReadPolicy: logicalvolume.ReadPolicyUnknown,
	})
	require.Error(t, err)
	mockRunner.AssertNotCalled(t, "Run")
}
