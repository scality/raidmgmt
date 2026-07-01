package storcli2

import "github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"

// ReadCacheToken maps a read policy to its bare storcli2 CLI token, shared by
// the "add vd" creation flags and the "set rdcache=" command. An unknown or
// otherwise unmappable policy yields ok=false so callers fail closed rather
// than emitting a token storcli2 cannot parse.
func ReadCacheToken(policy logicalvolume.ReadPolicy) (string, bool) {
	switch policy { //nolint:exhaustive // unmappable policies handled by the default
	case logicalvolume.ReadPolicyReadAhead:
		return "RA", true
	case logicalvolume.ReadPolicyNoReadAhead:
		return "NoRA", true
	default:
		return "", false
	}
}

// WriteCacheToken maps a write policy to its bare storcli2 CLI token, shared by
// the "add vd" creation flags and the "set wrcache=" command. An unknown or
// otherwise unmappable policy yields ok=false so callers fail closed rather
// than emitting a token storcli2 cannot parse.
func WriteCacheToken(policy logicalvolume.WritePolicy) (string, bool) {
	switch policy { //nolint:exhaustive // unmappable policies handled by the default
	case logicalvolume.WritePolicyWriteThrough:
		return "WT", true
	case logicalvolume.WritePolicyWriteBack:
		return "WB", true
	case logicalvolume.WritePolicyAlwaysWriteBack:
		return "AWB", true
	default:
		return "", false
	}
}
