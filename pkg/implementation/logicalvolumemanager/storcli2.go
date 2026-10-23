package logicalvolumemanager

import (
	"cmp"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/storcli2"
	"github.com/scality/raidmgmt/pkg/utils"
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
	// storcli2NewVolumeSettleTimeout bounds the wait for a new volume to be
	// exposed to the OS, and storcli2NewVolumeSettleInterval spaces its checks.
	storcli2NewVolumeSettleTimeout  = 60 * time.Second
	storcli2NewVolumeSettleInterval = 1 * time.Second
	// storcli2Driver is the kernel driver of the controllers storcli2 drives
	// (MegaRAID 9600 tri-mode, Dell PERC 12), the proc_name of their SCSI
	// hosts.
	storcli2Driver = "mpi3mr"
)

// StorCLI2 manages logical volumes through a storcli2/perccli2 command runner.
// A single implementation serves both binaries; the concrete runner is injected
// at construction time. The current state is read through injected getters: the
// physical-drive getter fills the request drives so creation can be validated,
// and the logical-volume getter discovers the virtual drive created by "add vd".
type StorCLI2 struct {
	ports.PhysicalDrivesGetter
	ports.LogicalVolumesGetter

	// NewVolumeSettleTimeout and NewVolumeSettleInterval bound and space the
	// wait for a new volume to be exposed to the OS. Zero means the default
	// (60s and 1s).
	NewVolumeSettleTimeout  time.Duration
	NewVolumeSettleInterval time.Duration
	// rescan asks the kernel to probe the controller's SCSI hosts for the new
	// volume between checks, and fileExists tells whether a path exists on
	// the host, to check that the new volume is a block device of the host.
	// They are set by tests only: nil rescans the mpi3mr hosts and reads the
	// filesystem.
	rescan     func() error
	fileExists func(path string) bool

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

	args := append([]string{
		fmt.Sprintf(storcli2ControllerSelector, request.CtrlMetadata.ID),
		storcli2CmdAdd,
		storcli2CmdVD,
		fmt.Sprintf(storcli2RAIDLevelFormat, request.RAIDLevel.Level()),
		drives,
	}, cacheFlags...)

	output, err := s.runner.Run(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run add vd command")
	}

	if _, err := storcli2.Decode(output); err != nil {
		return nil, errors.Wrap(err, "add vd command failed")
	}

	newLV, err := s.findNewVolumeUntilExposed(request)
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

// errNewVolumeNotExposed is returned while the new volume is listed but is not
// a block device of the host yet.
var errNewVolumeNotExposed = errors.New("new logical volume not exposed to the OS yet")

// findNewVolumeUntilExposed retries findNewLogicalVolume while the new volume
// is listed but not exposed to the OS yet, until NewVolumeSettleTimeout
// elapses: a volume without paths cannot be used by the caller. Between checks
// it forces a SCSI bus rescan, as megaraid does: the kernel may not announce a
// new volume on its own. A rescan failure is not fatal on its own, but it is
// reported on timeout.
func (s *StorCLI2) findNewVolumeUntilExposed(request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume,
	error,
) {
	timeout := cmp.Or(s.NewVolumeSettleTimeout, storcli2NewVolumeSettleTimeout)
	interval := cmp.Or(s.NewVolumeSettleInterval, storcli2NewVolumeSettleInterval)
	deadline := time.Now().Add(timeout)

	var lastRescanErr error

	for {
		lv, err := s.findNewLogicalVolume(request)
		if !errors.Is(err, errNewVolumeNotExposed) {
			return lv, err
		}

		if time.Now().After(deadline) {
			return nil, notExposedError(err, timeout, lastRescanErr)
		}

		if rescanErr := s.rescanSCSIHosts(); rescanErr != nil {
			lastRescanErr = rescanErr
		}

		time.Sleep(interval)
	}
}

// rescanSCSIHosts asks the kernel to probe the mpi3mr SCSI hosts, or runs
// the test hook when set.
func (s *StorCLI2) rescanSCSIHosts() error {
	if s.rescan != nil {
		return s.rescan()
	}

	return errors.Wrap(utils.RescanSCSIHosts(utils.SCSIHostRoot, storcli2Driver), "failed to rescan")
}

// notExposedError is the error of a new volume still not exposed after the
// settle timeout, with the last rescan failure if any.
func notExposedError(err error, timeout time.Duration, lastRescanErr error) error {
	if lastRescanErr != nil {
		return errors.Wrapf(err, "not exposed after %s (scsi rescan failing: %v)", timeout, lastRescanErr)
	}

	return errors.Wrapf(err, "not exposed after %s", timeout)
}

// findNewLogicalVolume returns the volume whose physical-drive set is exactly
// the request's. A physical drive belongs to a single virtual drive, so the
// freshly created volume is the only match on the host. It returns
// errNewVolumeNotExposed while no matching volume is a block device of the
// host yet.
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

	// A volume not exposed to the OS, such as the offline volume of a failed
	// drive still listing a requested slot, must not hide the new, exposed one.
	var unexposed *logicalvolume.LogicalVolume

	for _, lv := range logicalVolumes {
		if !storcli2MatchesPDSet(lv.PDrivesMetadata, wanted) {
			continue
		}

		// storcli2 may report a new volume exposed, with its paths, before
		// the kernel has discovered it: it is settled only once its path
		// exists on the host.
		if !s.onHost(lv) {
			unexposed = lv

			continue
		}

		return lv, nil
	}

	if unexposed != nil {
		return nil, errors.Wrapf(errNewVolumeNotExposed, "volume %s", unexposed.ID)
	}

	return nil, errors.New("no logical volume matches the requested physical drives")
}

// onHost tells whether the host has the block device of a volume: its by-id
// link, or its device node when storcli2 reports no NAA Id. A volume not
// exposed to the OS has neither.
func (s *StorCLI2) onHost(lv *logicalvolume.LogicalVolume) bool {
	path := cmp.Or(lv.PermanentPath, lv.DevicePath)
	if path == "" {
		return false
	}

	if s.fileExists != nil {
		return s.fileExists(path)
	}

	return utils.FileExists(path)
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
