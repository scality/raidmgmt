package physicaldrivegetter

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/storcli2"
	"github.com/scality/raidmgmt/pkg/utils"
)

const (
	// storcli2CmdShow and storcli2CmdAll are common storcli2 command tokens.
	storcli2CmdShow = "show"
	storcli2CmdAll  = "all"

	// storcli2AllDrivesSelector lists every drive of a controller.
	storcli2AllDrivesSelector = "/c%d/eall/sall"
	// storcli2EnclosureSelector and storcli2NoEnclosureSelector address a single
	// drive, with or without an enclosure component.
	storcli2EnclosureSelector   = "/c%d/e%s/s%s"
	storcli2NoEnclosureSelector = "/c%d/s%s"

	// storcli2JBODState is the base "State" of a drive in JBOD mode.
	storcli2JBODState = "JBOD"
	// storcli2ShieldedJBODState is the shielded variant of the JBOD state, the
	// only JBOD variant not prefixed with "JBOD" ("Shld-JBOD Shielded").
	storcli2ShieldedJBODState = "Shld"
	// storcli2ConfState is the base "State" of a configured drive.
	storcli2ConfState = "Conf"
	// storcli2UnconfState is the base "State" of an unconfigured drive.
	storcli2UnconfState = "UConf"
	// storcli2UnsupportedState is the "State" of an unconfigured drive the
	// controller cannot use ("UConfUnsp-Unconfigured Unsupported").
	storcli2UnsupportedState = "UConfUnsp"
	// storcli2GlobalHotSpareState and storcli2DedicatedHotSpareState are the
	// base "State" values of hot-spare drives.
	storcli2GlobalHotSpareState    = "GHS"
	storcli2DedicatedHotSpareState = "DHS"
	// storcli2BadStatus is the "Status" value of an unconfigured-bad drive.
	storcli2BadStatus = "Bad"
	// storcli2Failed is the "Failed" status. storcli2 has no "Failed" drive
	// state (unlike storcli1), but the state is still guarded for safety.
	storcli2Failed = "Failed"
	// storcli2OfflineStatus and storcli2MissingStatus are the "Status" values
	// of a drive that is offline or no longer present; like "Failed", they
	// describe a drive that is not functioning.
	storcli2OfflineStatus = "Offline"
	storcli2MissingStatus = "Missing"
	// storcli2UnusableStatus is the "Status" of a drive the controller cannot
	// use at all.
	storcli2UnusableStatus = "Unusable"
	// storcli2NVMeInterface is the "Intf" value of an NVMe drive. storcli2
	// reports the media of an NVMe drive as "SSD" and its transport in "Intf",
	// so the disk type is derived from the interface first.
	storcli2NVMeInterface = "NVMe"
)

type (
	// StorCLI2 reads physical-drive information through a storcli2 / perccli2
	// command runner. A single implementation serves both binaries; the concrete
	// runner is injected at construction time.
	StorCLI2 struct {
		runner commandrunner.CommandRunner
	}

	// storcli2DrivesListEntry is one entry of the "Drives List" section returned
	// by "show all" on a drive selector.
	storcli2DrivesListEntry struct {
		Information         storcli2DriveInformation         `json:"Drive Information"`
		DetailedInformation storcli2DriveDetailedInformation `json:"Drive Detailed Information"`
	}

	// storcli2DriveInformation is the summary block of a drive.
	storcli2DriveInformation struct {
		EIDSlot string `json:"EID:Slt"`
		Model   string `json:"Model"`
		Intf    string `json:"Intf"`
		Med     string `json:"Med"`
		Size    string `json:"Size"`
		State   string `json:"State"`
		Status  string `json:"Status"`
	}

	// storcli2DriveDetailedInformation is the detailed block of a drive.
	storcli2DriveDetailedInformation struct {
		Vendor       string `json:"Vendor"`
		SerialNumber string `json:"Serial Number"`
		WWN          string `json:"WWN"`
	}
)

var _ ports.PhysicalDrivesGetter = &StorCLI2{}

// NewStorCLI2 returns a physical-drive getter backed by the given storcli2 /
// perccli2 command runner.
func NewStorCLI2(runner commandrunner.CommandRunner) *StorCLI2 {
	return &StorCLI2{
		runner: runner,
	}
}

// PhysicalDrives returns every physical drive of the given controller. The whole
// inventory is read from a single "/cN/eall/sall show all" call, which already
// carries the detailed information for each drive.
func (s *StorCLI2) PhysicalDrives(metadata *raidcontroller.Metadata) (
	[]*physicaldrive.PhysicalDrive,
	error,
) {
	output, err := s.runner.Run([]string{
		fmt.Sprintf(storcli2AllDrivesSelector, metadata.ID),
		storcli2CmdShow,
		storcli2CmdAll,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show physical drives for controller %d", metadata.ID)
	}

	entries, err := decodeDrivesList(output)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode physical drives for controller %d", metadata.ID)
	}

	physicalDrives := make([]*physicaldrive.PhysicalDrive, 0, len(entries))

	for _, entry := range entries {
		physicalDrive, err := parseDrive(entry, metadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse physical drive %s", entry.Information.EIDSlot)
		}

		physicalDrives = append(physicalDrives, physicalDrive)
	}

	return physicalDrives, nil
}

// PhysicalDrive returns the physical drive addressed by the given metadata.
func (s *StorCLI2) PhysicalDrive(metadata *physicaldrive.Metadata) (
	*physicaldrive.PhysicalDrive,
	error,
) {
	selector, err := storcli2SelectorPD(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build drive selector")
	}

	output, err := s.runner.Run([]string{selector, storcli2CmdShow, storcli2CmdAll})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show details for physical drive %s", metadata.ID)
	}

	entries, err := decodeDrivesList(output)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode physical drive %s", metadata.ID)
	}

	if len(entries) == 0 {
		return nil, errors.Errorf("physical drive %s not found", metadata.ID)
	}

	physicalDrive, err := parseDrive(entries[0], metadata.CtrlMetadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse physical drive %s", metadata.ID)
	}

	return physicalDrive, nil
}

// decodeDrivesList decodes a storcli2 envelope and extracts its "Drives List".
// Per the StorCLI2 User Guide, showing a nonexistent object reports success,
// so an absent section means an empty inventory, not an error;
// PhysicalDrive()'s not-found guard then handles the single-drive case.
func decodeDrivesList(output []byte) ([]storcli2DrivesListEntry, error) {
	cmd, err := storcli2.Decode(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode storcli2 output")
	}

	entries, err := utils.UnmarshalToSlice[storcli2DrivesListEntry](
		cmd.Controllers[0].ResponseData, "Drives List",
	)
	if err != nil {
		if errors.Is(err, utils.ErrKeyNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to unmarshal drives list")
	}

	return entries, nil
}

// parseDrive maps a storcli2 drive entry to a PhysicalDrive entity owned by the
// given controller.
func parseDrive(entry storcli2DrivesListEntry, ctrl *raidcontroller.Metadata) (
	*physicaldrive.PhysicalDrive,
	error,
) {
	info := entry.Information
	detailed := entry.DetailedInformation

	slot, err := physicaldrive.ParseSlot(info.EIDSlot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse slot %s", info.EIDSlot)
	}

	size, err := utils.ConvertSizeBytes(info.Size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert size")
	}

	physicalDrive := &physicaldrive.PhysicalDrive{
		Metadata: &physicaldrive.Metadata{
			CtrlMetadata: ctrl,
			ID:           info.EIDSlot,
		},
		Slot:   slot,
		Vendor: strings.TrimSpace(detailed.Vendor),
		Model:  strings.TrimSpace(info.Model),
		Serial: strings.TrimSpace(detailed.SerialNumber),
		WWN:    formatWWN(detailed.WWN),
		Size:   size,
		Type:   diskType(info.Intf, info.Med),
		Status: pdStatus(info.State, info.Status),
		JBOD:   isJBODState(info.State),
		// Reason carries the raw drive status, mirroring the ssacli getter.
		Reason: info.Status,
	}

	// Only JBOD drives are exposed to the host, so only they have resolvable
	// device paths; a JBOD drive that is not functioning (failed, offline or
	// missing) may have lost its device node and must not fail the whole
	// inventory. ComputePaths reads the real filesystem (utils.FileExists), so
	// the healthy-JBOD path is exercised on hardware rather than in unit tests.
	if physicalDrive.JBOD && physicalDrive.Status == physicaldrive.PDStatusUsed {
		if err := physicalDrive.ComputePaths(); err != nil {
			return nil, errors.Wrap(err, "failed to compute paths")
		}
	}

	return physicalDrive, nil
}

// storcli2SelectorPD builds the storcli2 selector for a drive, choosing the
// enclosure or no-enclosure form from its parsed slot.
func storcli2SelectorPD(metadata *physicaldrive.Metadata) (string, error) {
	slot, err := physicaldrive.ParseSlot(metadata.ID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse slot %s", metadata.ID)
	}

	if slot.Enclosure != "" {
		return fmt.Sprintf(
			storcli2EnclosureSelector, metadata.CtrlMetadata.ID, slot.Enclosure, slot.Bay,
		), nil
	}

	return fmt.Sprintf(storcli2NoEnclosureSelector, metadata.CtrlMetadata.ID, slot.Bay), nil
}

// formatWWN normalises a storcli2 WWN to the "0x"-prefixed form used by the
// PhysicalDrive entity. An empty WWN stays empty.
func formatWWN(wwn string) string {
	trimmed := strings.TrimSpace(wwn)
	if trimmed == "" {
		return ""
	}

	return "0x" + trimmed
}

// diskType maps a storcli2 "Intf"/"Med" pair to a DiskType. storcli2 reports
// an NVMe drive with "SSD" media and the NVMe transport in its interface
// (Intf "NVMe", Med "SSD" in the User Guide sample), so the interface is
// checked before the media.
func diskType(intf, med string) physicaldrive.DiskType {
	if strings.EqualFold(intf, storcli2NVMeInterface) {
		return physicaldrive.DiskTypeNVMe
	}

	switch strings.ToUpper(med) {
	case "HDD":
		return physicaldrive.DiskTypeHDD
	case "SSD":
		return physicaldrive.DiskTypeSSD
	case "NVME":
		return physicaldrive.DiskTypeNVMe
	}

	return physicaldrive.DiskTypeUnknown
}

// pdStatus maps the storcli2 "State"/"Status" pair to a PDStatus.
//
// The mapping follows the drive-state legend printed by "show" on a drive
// selector: every base state (UConf, Conf, JBOD, GHS, DHS) also comes in
// suffixed variants (Shld for shielded diagnostics, Sntz for sanitize, Dgrd
// for degraded), so states are matched by family rather than exactly. A
// "Failed", "Offline" or "Missing" status takes precedence over any state: a
// drive that is not functioning must never be reported as in use or
// available (nor have its device paths resolved). Configured, JBOD and
// hot-spare drives are in use; unconfigured drives are available unless
// reported bad or unsupported. Any other state (Unusbl, Various, ...) is
// unknown, with the raw status kept in PhysicalDrive.Reason.
func pdStatus(state, status string) physicaldrive.PDStatus {
	switch {
	case strings.EqualFold(status, storcli2Failed),
		strings.EqualFold(status, storcli2OfflineStatus),
		strings.EqualFold(status, storcli2MissingStatus),
		strings.EqualFold(state, storcli2Failed):
		return physicaldrive.PDStatusFailed

	case strings.EqualFold(state, storcli2UnsupportedState):
		return physicaldrive.PDStatusUnassignedBad

	case hasStateFamily(state, storcli2UnconfState):
		if strings.EqualFold(status, storcli2BadStatus) ||
			strings.EqualFold(status, storcli2UnusableStatus) {
			return physicaldrive.PDStatusUnassignedBad
		}

		return physicaldrive.PDStatusUnassignedGood

	case hasStateFamily(state, storcli2ConfState),
		isJBODState(state),
		hasStateFamily(state, storcli2GlobalHotSpareState),
		hasStateFamily(state, storcli2DedicatedHotSpareState):
		return physicaldrive.PDStatusUsed
	}

	return physicaldrive.PDStatusUnknown
}

// hasStateFamily reports whether a drive state belongs to the given base-state
// family, i.e. is the base state itself or one of its suffixed variants.
func hasStateFamily(state, base string) bool {
	return len(state) >= len(base) && strings.EqualFold(state[:len(base)], base)
}

// isJBODState reports whether a drive state belongs to the JBOD family,
// including the shielded variant which is not prefixed with "JBOD".
func isJBODState(state string) bool {
	return hasStateFamily(state, storcli2JBODState) ||
		strings.EqualFold(state, storcli2ShieldedJBODState)
}
