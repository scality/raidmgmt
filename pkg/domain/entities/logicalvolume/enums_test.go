package logicalvolume_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
)

func TestReadPolicyIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		policy logicalvolume.ReadPolicy
		want   bool
	}{
		{name: "read ahead", policy: logicalvolume.ReadPolicyReadAhead, want: true},
		{name: "no read ahead", policy: logicalvolume.ReadPolicyNoReadAhead, want: true},
		{name: "unknown", policy: logicalvolume.ReadPolicyUnknown, want: false},
		{name: "empty", policy: logicalvolume.ReadPolicy(""), want: false},
		{name: "garbage", policy: logicalvolume.ReadPolicy("garbage"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.policy.IsValid())
		})
	}
}

func TestWritePolicyIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		policy logicalvolume.WritePolicy
		want   bool
	}{
		{name: "write through", policy: logicalvolume.WritePolicyWriteThrough, want: true},
		{name: "write back", policy: logicalvolume.WritePolicyWriteBack, want: true},
		{name: "always write back", policy: logicalvolume.WritePolicyAlwaysWriteBack, want: true},
		{name: "unknown", policy: logicalvolume.WritePolicyUnknown, want: false},
		{name: "empty", policy: logicalvolume.WritePolicy(""), want: false},
		{name: "garbage", policy: logicalvolume.WritePolicy("garbage"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.policy.IsValid())
		})
	}
}

func TestIOPolicyIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		policy logicalvolume.IOPolicy
		want   bool
	}{
		{name: "direct", policy: logicalvolume.IOPolicyDirect, want: true},
		{name: "cached", policy: logicalvolume.IOPolicyCached, want: true},
		{name: "unknown", policy: logicalvolume.IOPolicyUnknown, want: false},
		{name: "empty", policy: logicalvolume.IOPolicy(""), want: false},
		{name: "garbage", policy: logicalvolume.IOPolicy("garbage"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.policy.IsValid())
		})
	}
}

func TestRAIDLevelIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level logicalvolume.RAIDLevel
		want  bool
	}{
		{name: "raid 0", level: logicalvolume.RAIDLevel0, want: true},
		{name: "raid 1", level: logicalvolume.RAIDLevel1, want: true},
		{name: "raid 10", level: logicalvolume.RAIDLevel10, want: true},
		{name: "unknown", level: logicalvolume.RAIDLevelUnknown, want: false},
		{name: "out of range", level: logicalvolume.RAIDLevel(99), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.level.IsValid())
		})
	}
}
