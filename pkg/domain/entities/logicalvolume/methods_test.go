package logicalvolume_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
)

// validCacheOptions returns a fully-populated, valid CacheOptions.
func validCacheOptions() *logicalvolume.CacheOptions {
	return &logicalvolume.CacheOptions{
		ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
		WritePolicy: logicalvolume.WritePolicyWriteBack,
		IOPolicy:    logicalvolume.IOPolicyDirect,
	}
}

// pdMetadata returns n valid physical-drive metadata entries.
func pdMetadata(n int) []*physicaldrive.Metadata {
	pds := make([]*physicaldrive.Metadata, 0, n)
	for i := range n {
		pds = append(pds, &physicaldrive.Metadata{
			CtrlMetadata: &raidcontroller.Metadata{ID: 0},
			ID:           "252:" + string(rune('0'+i)),
		})
	}

	return pds
}

func TestCacheOptionsValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    *logicalvolume.CacheOptions
		wantErr bool
	}{
		{
			name:    "nil is rejected without panicking",
			opts:    nil,
			wantErr: true,
		},
		{
			name:    "fully valid",
			opts:    validCacheOptions(),
			wantErr: false,
		},
		{
			name: "io policy unset is allowed",
			opts: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			wantErr: false,
		},
		{
			name: "io policy unknown is allowed",
			opts: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicyUnknown,
			},
			wantErr: false,
		},
		{
			name: "io policy garbage is rejected",
			opts: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicy("garbage"),
			},
			wantErr: true,
		},
		{
			name: "read policy garbage is rejected",
			opts: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicy("garbage"),
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			wantErr: true,
		},
		{
			name: "read policy unset is rejected",
			opts: &logicalvolume.CacheOptions{
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			wantErr: true,
		},
		{
			name: "write policy garbage is rejected",
			opts: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicy("garbage"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.opts.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRequestValidate(t *testing.T) {
	t.Parallel()

	ctrl := &raidcontroller.Metadata{ID: 0}

	tests := []struct {
		name    string
		request *logicalvolume.Request
		wantErr bool
	}{
		{
			name:    "nil request is rejected",
			request: nil,
			wantErr: true,
		},
		{
			name: "valid raid 0 with nil cache options",
			request: &logicalvolume.Request{
				CtrlMetadata:    ctrl,
				RAIDLevel:       logicalvolume.RAIDLevel0,
				PDrivesMetadata: pdMetadata(1),
			},
			wantErr: false,
		},
		{
			name: "valid raid 1 with cache options",
			request: &logicalvolume.Request{
				CtrlMetadata:    ctrl,
				RAIDLevel:       logicalvolume.RAIDLevel1,
				PDrivesMetadata: pdMetadata(2),
				CacheOptions:    validCacheOptions(),
			},
			wantErr: false,
		},
		{
			name: "unknown raid level is rejected",
			request: &logicalvolume.Request{
				CtrlMetadata:    ctrl,
				RAIDLevel:       logicalvolume.RAIDLevelUnknown,
				PDrivesMetadata: pdMetadata(1),
			},
			wantErr: true,
		},
		{
			name: "out of range raid level is rejected",
			request: &logicalvolume.Request{
				CtrlMetadata:    ctrl,
				RAIDLevel:       logicalvolume.RAIDLevel(99),
				PDrivesMetadata: pdMetadata(1),
			},
			wantErr: true,
		},
		{
			name: "not enough drives for raid 1 is rejected",
			request: &logicalvolume.Request{
				CtrlMetadata:    ctrl,
				RAIDLevel:       logicalvolume.RAIDLevel1,
				PDrivesMetadata: pdMetadata(1),
			},
			wantErr: true,
		},
		{
			name: "odd drive count for raid 10 is rejected",
			request: &logicalvolume.Request{
				CtrlMetadata:    ctrl,
				RAIDLevel:       logicalvolume.RAIDLevel10,
				PDrivesMetadata: pdMetadata(5),
			},
			wantErr: true,
		},
		{
			name: "invalid cache options are rejected",
			request: &logicalvolume.Request{
				CtrlMetadata:    ctrl,
				RAIDLevel:       logicalvolume.RAIDLevel0,
				PDrivesMetadata: pdMetadata(1),
				CacheOptions: &logicalvolume.CacheOptions{
					ReadPolicy:  logicalvolume.ReadPolicy("garbage"),
					WritePolicy: logicalvolume.WritePolicyWriteBack,
				},
			},
			wantErr: true,
		},
		{
			name: "empty physical drives is rejected",
			request: &logicalvolume.Request{
				CtrlMetadata:    ctrl,
				RAIDLevel:       logicalvolume.RAIDLevel0,
				PDrivesMetadata: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestRequestValidateOptionalCacheOptions documents that a nil CacheOptions is
// treated as "use controller defaults" rather than panicking, matching the
// storcli2 adapter which emits no cache flags for a nil CacheOptions.
func TestRequestValidateOptionalCacheOptions(t *testing.T) {
	t.Parallel()

	request := &logicalvolume.Request{
		CtrlMetadata:    &raidcontroller.Metadata{ID: 0},
		RAIDLevel:       logicalvolume.RAIDLevel0,
		PDrivesMetadata: pdMetadata(1),
		CacheOptions:    nil,
	}

	assert.NoError(t, request.Validate())
}
