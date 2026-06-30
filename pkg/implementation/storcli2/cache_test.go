package storcli2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/implementation/storcli2"
)

func TestReadCacheToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		policy    logicalvolume.ReadPolicy
		wantToken string
		wantOK    bool
	}{
		{logicalvolume.ReadPolicyReadAhead, "RA", true},
		{logicalvolume.ReadPolicyNoReadAhead, "NoRA", true},
		{logicalvolume.ReadPolicyUnknown, "", false},
		{logicalvolume.ReadPolicy(""), "", false},
		{logicalvolume.ReadPolicy("bogus"), "", false},
	}

	for _, tt := range tests {
		token, ok := storcli2.ReadCacheToken(tt.policy)
		assert.Equal(t, tt.wantOK, ok, "policy %q", tt.policy)
		assert.Equal(t, tt.wantToken, token, "policy %q", tt.policy)
	}
}

func TestWriteCacheToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		policy    logicalvolume.WritePolicy
		wantToken string
		wantOK    bool
	}{
		{logicalvolume.WritePolicyWriteThrough, "WT", true},
		{logicalvolume.WritePolicyWriteBack, "WB", true},
		{logicalvolume.WritePolicyAlwaysWriteBack, "AWB", true},
		{logicalvolume.WritePolicyUnknown, "", false},
		{logicalvolume.WritePolicy(""), "", false},
		{logicalvolume.WritePolicy("bogus"), "", false},
	}

	for _, tt := range tests {
		token, ok := storcli2.WriteCacheToken(tt.policy)
		assert.Equal(t, tt.wantOK, ok, "policy %q", tt.policy)
		assert.Equal(t, tt.wantToken, token, "policy %q", tt.policy)
	}
}
