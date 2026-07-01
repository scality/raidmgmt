package logicalvolumemanager

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/storcli2"
)

const (
	// storcli2CmdAdd and storcli2CmdVD compose the "add vd" command that creates
	// a virtual drive.
	storcli2CmdAdd = "add"
	storcli2CmdVD  = "vd"
	// storcli2CmdDelete deletes the virtual drive addressed by the selector.
	storcli2CmdDelete = "delete"
	// storcli2CmdExpand adds physical drives to a virtual drive (online capacity
	// expansion). storcli2 dropped "start migrate", so this is the only way to
	// grow a volume; the RAID level is preserved by the firmware.
	storcli2CmdExpand = "expand"
	// storcli2ControllerSelector addresses a whole controller (used by "add vd").
	storcli2ControllerSelector = "/c%d"
	// storcli2VolumeSelector addresses a single virtual drive by its number.
	storcli2VolumeSelector = "/c%d/v%s"
	// storcli2DrivesFlag prefixes the "e:s,s,..." drive list of "add vd".
	storcli2DrivesFlag = "drives="
	// storcli2RAIDLevelFormat builds the bare RAID-level token of "add vd"
	// (e.g. "r0"); storcli's "type=raidN" form is gone in storcli2.
	storcli2RAIDLevelFormat = "r%d"
)

// StorCLI2 manages logical volumes through a storcli2/perccli2 command runner.
// A single implementation serves both binaries; the concrete runner is injected
// at construction time. The current state is read through injected getters: the
// physical-drive getter fills the request drives so creation can be validated,
// and the logical-volume getter discovers the virtual drive created by "add vd".
type StorCLI2 struct {
	ports.PhysicalDrivesGetter
	ports.LogicalVolumesGetter

	runner commandrunner.CommandRunner
}

var _ ports.LogicalVolumesManager = &StorCLI2{}

// NewStorCLI2 returns a logical-volume manager backed by the given storcli2 /
// perccli2 command runner and physical-drive and logical-volume getters.
func NewStorCLI2(
	runner commandrunner.CommandRunner,
	physicalDrivesGetter ports.PhysicalDrivesGetter,
	logicalVolumesGetter ports.LogicalVolumesGetter,
) *StorCLI2 {
	return &StorCLI2{
		PhysicalDrivesGetter: physicalDrivesGetter,
		LogicalVolumesGetter: logicalVolumesGetter,
		runner:               runner,
	}
}

// CreateLV creates a logical volume from a request through "add vd". The
// request drives are filled via the physical-drive getter so the RAID creation
// can be validated, formatted into storcli2's "e:s,s,..." drive list (a single
// enclosure only) and submitted with the bare RAID-level and cache tokens.
// storcli2 reports the new virtual drive's number but not its full state, so
// the volume is rediscovered by matching the request's physical-drive set.
func (s *StorCLI2) CreateLV(request *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	physicalDrives := make([]*physicaldrive.PhysicalDrive, 0, len(request.PDrivesMetadata))

	for _, pdMetadata := range request.PDrivesMetadata {
		pd, err := s.PhysicalDrive(pdMetadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get physical drive %s", pdMetadata.ID)
		}

		physicalDrives = append(physicalDrives, pd)
	}

	if err := logicalvolume.ValidateRAIDCreation(physicalDrives, request.RAIDLevel); err != nil {
		return nil, errors.Wrap(err, "failed to validate RAID creation")
	}

	drives, err := storcli2FormatDrives(request.PDrivesMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to format drives")
	}

	cacheFlags, err := storcli2CreateCacheFlags(request.CacheOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve cache options")
	}

	args := []string{
		fmt.Sprintf(storcli2ControllerSelector, request.CtrlMetadata.ID),
		storcli2CmdAdd,
		storcli2CmdVD,
		fmt.Sprintf(storcli2RAIDLevelFormat, request.RAIDLevel.Level()),
		drives,
	}
	args = append(args, cacheFlags...)

	output, err := s.runner.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run add vd command")
	}

	if _, err := storcli2.Decode(output); err != nil {
		return nil, errors.Wrap(err, "add vd command failed")
	}

	newLV, err := s.findNewLogicalVolume(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find new logical volume")
	}

	return newLV, nil
}

// DeleteLV deletes the logical volume addressed by the given metadata. A
// nonexistent virtual drive yields a failure payload surfaced by Decode.
func (s *StorCLI2) DeleteLV(metadata *logicalvolume.Metadata) error {
	selector := fmt.Sprintf(storcli2VolumeSelector, metadata.CtrlMetadata.ID, metadata.ID)

	output, err := s.runner.Run([]string{selector, storcli2CmdDelete})
	if err != nil {
		return errors.Wrapf(err, "failed to run delete command for logical volume %s", metadata.ID)
	}

	if _, err := storcli2.Decode(output); err != nil {
		return errors.Wrapf(err, "failed to delete logical volume %s", metadata.ID)
	}

	return nil
}

// AddPDsToLV grows a logical volume with the given physical drives through
// "expand" (online capacity expansion). storcli2 dropped "start migrate", so
// expansion is the only supported path; the RAID level is preserved by the
// firmware. The drives must share a single enclosure.
func (s *StorCLI2) AddPDsToLV(
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	drives, err := storcli2FormatDrives(pdsMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to format drives")
	}

	selector := fmt.Sprintf(storcli2VolumeSelector, lvMetadata.CtrlMetadata.ID, lvMetadata.ID)

	output, err := s.runner.Run([]string{selector, storcli2CmdExpand, drives})
	if err != nil {
		return errors.Wrapf(err, "failed to run expand command for logical volume %s", lvMetadata.ID)
	}

	if _, err := storcli2.Decode(output); err != nil {
		return errors.Wrapf(err, "failed to expand logical volume %s", lvMetadata.ID)
	}

	return nil
}

// DeletePDsFromLV is not supported by storcli2: the storcli-to-storcli2 command
// map drops "start migrate" with no replacement for removing drives from a
// volume (see DESIGN.md).
func (*StorCLI2) DeletePDsFromLV(
	_ *logicalvolume.Metadata,
	_ ...*physicaldrive.Metadata,
) error {
	return ports.ErrFunctionNotSupportedByImplementation
}

// findNewLogicalVolume returns the volume whose physical-drive set is exactly
// the request's. A physical drive belongs to a single virtual drive, so the
// freshly created volume is the only match.
func (s *StorCLI2) findNewLogicalVolume(request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume,
	error,
) {
	logicalVolumes, err := s.LogicalVolumes(request.CtrlMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list logical volumes")
	}

	wanted := make(map[string]struct{}, len(request.PDrivesMetadata))
	for _, pdMetadata := range request.PDrivesMetadata {
		wanted[pdMetadata.ID] = struct{}{}
	}

	for _, lv := range logicalVolumes {
		if storcli2MatchesPDSet(lv.PDrivesMetadata, wanted) {
			return lv, nil
		}
	}

	return nil, errors.New("no logical volume matches the requested physical drives")
}

// storcli2MatchesPDSet reports whether the physical drives of a volume are
// exactly the wanted set.
func storcli2MatchesPDSet(lvPDs []*physicaldrive.Metadata, wanted map[string]struct{}) bool {
	if len(lvPDs) != len(wanted) {
		return false
	}

	for _, pd := range lvPDs {
		if _, ok := wanted[pd.ID]; !ok {
			return false
		}
	}

	return true
}

// storcli2FormatDrives renders physical-drive metadata into the "drives=e:s,..."
// argument of "add vd". storcli2 names every drive by its "EID:Slt" pair and
// accepts a single shared enclosure, so drives spanning multiple enclosures are
// rejected.
func storcli2FormatDrives(pdsMetadata []*physicaldrive.Metadata) (string, error) {
	if len(pdsMetadata) == 0 {
		return "", errors.New("no physical drives")
	}

	var enclosure string

	slots := make([]string, 0, len(pdsMetadata))

	for _, pdMetadata := range pdsMetadata {
		slot, err := physicaldrive.ParseSlot(pdMetadata.ID)
		if err != nil {
			return "", errors.Wrapf(err, "failed to parse slot %s", pdMetadata.ID)
		}

		if slot.Enclosure == "" {
			return "", errors.Errorf("missing enclosure in drive %s", pdMetadata.ID)
		}

		switch enclosure {
		case "":
			enclosure = slot.Enclosure
		case slot.Enclosure:
		default:
			return "", errors.New("multiple enclosures not supported")
		}

		slots = append(slots, slot.Bay)
	}

	return fmt.Sprintf("%s%s:%s", storcli2DrivesFlag, enclosure, strings.Join(slots, ",")), nil
}

// storcli2CreateCacheFlags returns the bare write- and read-policy tokens of
// "add vd". storcli2 takes the policy as a bare token at creation time (the
// "wrcache="/"rdpolicy=" forms are gone) and has no IO policy. The request is
// validated upstream (Request.Validate rejects Unknown policies), so each
// policy is expected to map; an unmappable one is an error rather than emitted
// verbatim (the shared mapping fails closed, see DESIGN.md § Adapters).
//
// The shared mapping yields the canonical token used by the "set" command
// (e.g. "WB"); "add vd" documents the lowercase form ("wb"), so it is
// lowercased here. The mapping is reused only for validation and the token
// value, not its case.
func storcli2CreateCacheFlags(cache *logicalvolume.CacheOptions) ([]string, error) {
	if cache == nil {
		return nil, nil
	}

	writeToken, ok := storcli2.WriteCacheToken(cache.WritePolicy)
	if !ok {
		return nil, errors.Errorf("unsettable write policy %q", cache.WritePolicy)
	}

	readToken, ok := storcli2.ReadCacheToken(cache.ReadPolicy)
	if !ok {
		return nil, errors.Errorf("unsettable read policy %q", cache.ReadPolicy)
	}

	return []string{strings.ToLower(writeToken), strings.ToLower(readToken)}, nil
}
