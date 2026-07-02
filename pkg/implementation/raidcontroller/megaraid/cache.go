package megaraid

import (
	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
)

// The cache-policy CLI tokens are mapped explicitly here rather than derived
// from the domain enum's underlying string value: the megaraid v1 command-line
// wire format is owned by this adapter, so a change to the logicalvolume enum
// constants cannot silently alter the emitted command. Each mapper fails closed
// (ok=false) on an unmappable policy so callers never emit a token the CLI
// cannot parse. Mirrors the storcli2 adapter's token mapping.

// megaraidReadCacheToken maps a read policy to its megaraid CLI token.
func megaraidReadCacheToken(policy logicalvolume.ReadPolicy) (string, bool) {
	switch policy { //nolint:exhaustive // unmappable policies handled by the default
	case logicalvolume.ReadPolicyReadAhead:
		return "ra", true
	case logicalvolume.ReadPolicyNoReadAhead:
		return "nora", true
	default:
		return "", false
	}
}

// megaraidWriteCacheToken maps a write policy to its megaraid CLI token.
func megaraidWriteCacheToken(policy logicalvolume.WritePolicy) (string, bool) {
	switch policy { //nolint:exhaustive // unmappable policies handled by the default
	case logicalvolume.WritePolicyWriteThrough:
		return "wt", true
	case logicalvolume.WritePolicyWriteBack:
		return "wb", true
	case logicalvolume.WritePolicyAlwaysWriteBack:
		return "awb", true
	default:
		return "", false
	}
}

// megaraidIOPolicyToken maps an IO policy to its megaraid CLI token. An unset or
// Unknown value yields ok=false; whether that is an error is left to the caller
// (the "add vd" path requires it, the "set" path treats it as optional).
func megaraidIOPolicyToken(policy logicalvolume.IOPolicy) (string, bool) {
	switch policy { //nolint:exhaustive // unmappable policies handled by the default
	case logicalvolume.IOPolicyDirect:
		return "direct", true
	case logicalvolume.IOPolicyCached:
		return "cached", true
	default:
		return "", false
	}
}

// megaraidSetCacheFlags returns the "set" options for the cache policies that
// differ between the desired and current options, each mapped to its megaraid
// CLI token. A changed read or write policy that does not map is an error (fail
// closed); a changed IO policy is emitted only when it maps, since it is
// optional and left to the controller otherwise.
// The caller passes the current options read back from the controller, which is
// always populated; only desired is guarded, since a nil desired legitimately
// means "no change requested".
func megaraidSetCacheFlags(desired, current *logicalvolume.CacheOptions) ([]string, error) {
	if desired == nil {
		return nil, nil
	}

	var options []string

	if desired.ReadPolicy != current.ReadPolicy {
		token, ok := megaraidReadCacheToken(desired.ReadPolicy)
		if !ok {
			return nil, errors.Errorf("unsettable read policy %q", desired.ReadPolicy)
		}

		options = append(options, "rdcache="+token)
	}

	if desired.WritePolicy != current.WritePolicy {
		token, ok := megaraidWriteCacheToken(desired.WritePolicy)
		if !ok {
			return nil, errors.Errorf("unsettable write policy %q", desired.WritePolicy)
		}

		options = append(options, "wrcache="+token)
	}

	if desired.IOPolicy != current.IOPolicy {
		if token, ok := megaraidIOPolicyToken(desired.IOPolicy); ok {
			options = append(options, "iopolicy="+token)
		}
	}

	return options, nil
}
