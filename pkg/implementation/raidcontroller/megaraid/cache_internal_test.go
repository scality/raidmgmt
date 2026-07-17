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
			want: []string{"ra", "wb", "direct"},
		},
		{
			name: "unset io policy fails closed (mandatory for v1 add vd)",
			cache: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
			},
			wantErr: true,
		},
		{
			name: "unknown io policy fails closed (mandatory for v1 add vd)",
			cache: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicyUnknown,
			},
			wantErr: true,
		},
		{
			name: "invalid io policy fails closed",
			cache: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicy("bogus"),
			},
			wantErr: true,
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

// TestMegaraidCacheTokens pins the adapter-owned mapping from domain cache
// policies to megaraid CLI tokens, so a change to the logicalvolume enum
// constant values cannot silently alter the emitted command.
func TestMegaraidCacheTokens(t *testing.T) {
	t.Parallel()

	t.Run("read", func(t *testing.T) {
		t.Parallel()

		cases := map[logicalvolume.ReadPolicy]struct {
			token string
			ok    bool
		}{
			logicalvolume.ReadPolicyReadAhead:   {"ra", true},
			logicalvolume.ReadPolicyNoReadAhead: {"nora", true},
			logicalvolume.ReadPolicyUnknown:     {"", false},
			logicalvolume.ReadPolicy("bogus"):   {"", false},
		}
		for policy, want := range cases {
			token, ok := megaraidReadCacheToken(policy)
			require.Equal(t, want.ok, ok, "policy %q", policy)
			require.Equal(t, want.token, token, "policy %q", policy)
		}
	})

	t.Run("write", func(t *testing.T) {
		t.Parallel()

		cases := map[logicalvolume.WritePolicy]struct {
			token string
			ok    bool
		}{
			logicalvolume.WritePolicyWriteThrough:    {"wt", true},
			logicalvolume.WritePolicyWriteBack:       {"wb", true},
			logicalvolume.WritePolicyAlwaysWriteBack: {"awb", true},
			logicalvolume.WritePolicyUnknown:         {"", false},
			logicalvolume.WritePolicy("bogus"):       {"", false},
		}
		for policy, want := range cases {
			token, ok := megaraidWriteCacheToken(policy)
			require.Equal(t, want.ok, ok, "policy %q", policy)
			require.Equal(t, want.token, token, "policy %q", policy)
		}
	})

	t.Run("io", func(t *testing.T) {
		t.Parallel()

		cases := map[logicalvolume.IOPolicy]struct {
			token string
			ok    bool
		}{
			logicalvolume.IOPolicyDirect:  {"direct", true},
			logicalvolume.IOPolicyCached:  {"cached", true},
			logicalvolume.IOPolicyUnknown: {"", false},
			logicalvolume.IOPolicy(""):    {"", false},
		}
		for policy, want := range cases {
			token, ok := megaraidIOPolicyToken(policy)
			require.Equal(t, want.ok, ok, "policy %q", policy)
			require.Equal(t, want.token, token, "policy %q", policy)
		}
	})
}

// TestMegaraidSetCacheFlags covers the diff-based "set" helper: only changed
// policies are emitted, a changed-but-unmappable IO policy is silently skipped
// (it is optional), and an unmappable read/write policy fails closed.
func TestMegaraidSetCacheFlags(t *testing.T) {
	t.Parallel()

	current := &logicalvolume.CacheOptions{
		ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
		WritePolicy: logicalvolume.WritePolicyWriteBack,
		IOPolicy:    logicalvolume.IOPolicyDirect,
	}

	tests := []struct {
		name    string
		desired *logicalvolume.CacheOptions
		want    []string
		wantErr bool
	}{
		{
			name:    "nil desired yields no options",
			desired: nil,
			want:    nil,
		},
		{
			name: "no diff yields no options",
			desired: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicyDirect,
			},
			want: nil,
		},
		{
			name: "read-only diff",
			desired: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyNoReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicyDirect,
			},
			want: []string{"rdcache=nora"},
		},
		{
			name: "io-only diff emits token",
			desired: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicyCached,
			},
			want: []string{"iopolicy=cached"},
		},
		{
			name: "io diff to unknown is skipped",
			desired: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicyWriteBack,
				IOPolicy:    logicalvolume.IOPolicyUnknown,
			},
			want: nil,
		},
		{
			name: "unmappable write policy fails closed",
			desired: &logicalvolume.CacheOptions{
				ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
				WritePolicy: logicalvolume.WritePolicy("bogus"),
				IOPolicy:    logicalvolume.IOPolicyDirect,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := megaraidSetCacheFlags(tt.desired, current)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
