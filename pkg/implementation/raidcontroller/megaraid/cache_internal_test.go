package megaraid

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
)

// TestMegaraidCreateCacheFlags pins the "add vd" cache flags: read and write
// policies must map (fail closed on an unrecognized value), while the IO policy
// is optional and emitted only when set to a valid value.
func TestMegaraidCreateCacheFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cache   *logicalvolume.CacheOptions
		want    []string
		wantErr bool
	}{
		{
			name:  "nil cache yields no flags",
			cache: nil,
			want:  nil,
		},
		{
			name: "all policies valid",
			cache: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicyDirect,
			},
			want: []string{"rdpolicy=ra", "wrcache=wb", "iopolicy=direct"},
		},
		{
			name: "unset io policy is omitted",
			cache: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			want: []string{"rdpolicy=ra", "wrcache=wb"},
		},
		{
			name: "unknown io policy is omitted",
			cache: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicyUnknown,
			},
			want: []string{"rdpolicy=ra", "wrcache=wb"},
		},
		{
			name: "invalid read policy fails closed",
			cache: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicy("bogus"),
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			wantErr: true,
		},
		{
			name: "invalid write policy fails closed",
			cache: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicy("bogus"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := megaraidCreateCacheFlags(tt.cache)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
