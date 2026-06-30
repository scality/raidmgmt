package lvcachesetter

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/storcli2"
)

const (
	// storcli2CmdSet is the storcli2 "set" command token.
	storcli2CmdSet = "set"
	// storcli2VolumeSelector addresses a single virtual drive by its number.
	storcli2VolumeSelector = "/c%d/v%s"
	// storcli2RdCacheFlag and storcli2WrCacheFlag are the read/write cache flags
	// of the "set" command. storcli2 has no IO policy flag.
	storcli2RdCacheFlag = "rdcache="
	storcli2WrCacheFlag = "wrcache="
)

// StorCLI2 sets cache options on a logical volume through a storcli2 /
// perccli2 command runner. A single implementation serves both binaries; the
// concrete runner is injected at construction time. The current state is read
// through an injected LogicalVolumesGetter so setters only emit the flags that
// actually change.
type StorCLI2 struct {
	ports.LogicalVolumesGetter

	runner commandrunner.CommandRunner
}

var _ ports.LVCacheSetter = &StorCLI2{}

// NewStorCLI2 returns a cache setter backed by the given storcli2 / perccli2
// command runner and logical-volume getter.
func NewStorCLI2(
	runner commandrunner.CommandRunner,
	logicalVolumesGetter ports.LogicalVolumesGetter,
) *StorCLI2 {
	return &StorCLI2{
		LogicalVolumesGetter: logicalVolumesGetter,
		StorCLI2:             runner,
	}
}

// SetLVCacheOptions applies the desired cache options to a logical volume,
// emitting only the flags that differ from the current state. storcli2 rejects
// the combined cache syntax and dropped the IO policy, so the read and write
// policies are set through two independent "set" commands and the IO policy is
// ignored.
func (s *StorCLI2) SetLVCacheOptions(
	metadata *logicalvolume.Metadata,
	desired *logicalvolume.CacheOptions,
) error {
	// Read-before-write is deliberate: it is not about idempotency (the "set"
	// commands are idempotent) but about emitting only the changed flags. This
	// minimizes real mutations and, crucially, skips fields the lossy getter
	// reports as Unknown (which the token funcs reject as unsettable) when the
	// caller did not actually change them. See DESIGN.md § Adapters.
	current, err := s.LogicalVolume(metadata)
	if err != nil {
		return errors.Wrapf(err, "failed to get logical volume %s", metadata.ID)
	}

	options, err := storcli2CacheOptions(current.CacheOptions, desired)
	if err != nil {
		return errors.Wrap(err, "failed to resolve cache options")
	}

	selector := fmt.Sprintf(storcli2VolumeSelector, metadata.CtrlMetadata.ID, metadata.ID)

	for _, option := range options {
		if err := s.set(selector, option); err != nil {
			return errors.Wrapf(err, "failed to set %s", option)
		}
	}

	return nil
}

// storcli2CacheOptions returns the "set" options for the policies that differ
// between current and desired, one per command (storcli2 rejects the combined
// syntax). A changed but unsettable (unknown) policy is an error.
func storcli2CacheOptions(current, desired *logicalvolume.CacheOptions) ([]string, error) {
	var options []string

	if desired.ReadPolicy != current.ReadPolicy {
		token, ok := storcli2ReadCacheToken(desired.ReadPolicy)
		if !ok {
			return nil, errors.Errorf("unsettable read policy %q", desired.ReadPolicy)
		}

		options = append(options, storcli2RdCacheFlag+token)
	}

	if desired.WritePolicy != current.WritePolicy {
		token, ok := storcli2WriteCacheToken(desired.WritePolicy)
		if !ok {
			return nil, errors.Errorf("unsettable write policy %q", desired.WritePolicy)
		}

		options = append(options, storcli2WrCacheFlag+token)
	}

	return options, nil
}

// set runs a single "set" command on a volume selector and surfaces the in-JSON
// failure that storcli2 may report regardless of its exit code.
func (s *StorCLI2) set(selector, option string) error {
	output, err := s.StorCLI2.Run([]string{selector, storcli2CmdSet, option})
	if err != nil {
		return errors.Wrap(err, "failed to run set command")
	}

	if _, err := storcli2.Decode(output); err != nil {
		return errors.Wrap(err, "set command failed")
	}

	return nil
}

// storcli2ReadCacheToken maps a read policy to its "rdcache" token. An unknown
// policy is not settable and yields ok=false.
func storcli2ReadCacheToken(policy logicalvolume.ReadPolicy) (string, bool) {
	switch policy { //nolint:exhaustive // unknown handled by the default
	case logicalvolume.ReadPolicyReadAhead:
		return "RA", true
	case logicalvolume.ReadPolicyNoReadAhead:
		return "NoRA", true
	default:
		return "", false
	}
}

// storcli2WriteCacheToken maps a write policy to its "wrcache" token. An unknown
// policy is not settable and yields ok=false.
func storcli2WriteCacheToken(policy logicalvolume.WritePolicy) (string, bool) {
	switch policy { //nolint:exhaustive // unknown handled by the default
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
