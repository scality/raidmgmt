package logicalvolumegetter

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
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

	// storcli2AllVolumesSelector lists every virtual drive of a controller.
	storcli2AllVolumesSelector = "/c%d/vall"
	// storcli2VolumeSelector addresses a single virtual drive by its number.
	storcli2VolumeSelector = "/c%d/v%s"

	// storcli2VirtualDrivesKey is the "Response Data" key holding the virtual
	// drives returned by "show all" on a volume selector.
	storcli2VirtualDrivesKey = "Virtual Drives"

	// storcli2OptimalState, storcli2DegradedState, storcli2PartiallyDegradedState,
	// storcli2RecoveryState and storcli2OfflineState are the "State" values
	// reported for a virtual drive, per the StorCLI2 User Guide v1.1 legend
	// ("Rec=Recovery|OfLn=OffLine|Pdgd=Partially Degraded|Dgrd=Degraded|
	// Optl=Optimal"). storcli2FailedState is not in that legend (storcli1 had
	// it) but is kept as a defensive guard.
	storcli2OptimalState           = "Optl"
	storcli2DegradedState          = "Dgrd"
	storcli2PartiallyDegradedState = "Pdgd"
	storcli2RecoveryState          = "Rec"
	storcli2OfflineState           = "OfLn"
	storcli2FailedState            = "Fail"

	// storcli2PermanentPathPrefix is the /dev/disk/by-id prefix combined with a
	// virtual drive's SCSI NAA Id to form its permanent path.
	storcli2PermanentPathPrefix = "/dev/disk/by-id/wwn-0x"
)

type (
	// StorCLI2 reads logical-volume information through a storcli2 / perccli2
	// command runner. A single implementation serves both binaries; the concrete
	// runner is injected at construction time.
	StorCLI2 struct {
		runner commandrunner.CommandRunner
	}

	// storcli2VirtualDrive is one entry of the "Virtual Drives" section returned
	// by "show all" on a volume selector.
	storcli2VirtualDrive struct {
		Info       storcli2VDInfo            `json:"VD Info"`
		PDs        []storcli2VDPhysicalDrive `json:"PDs"`
		Properties storcli2VDProperties      `json:"VD Properties"`
	}

	// storcli2VDInfo is the summary block of a virtual drive. DGVD is the
	// "<disk group>/<virtual drive>" pair; only the virtual drive part is the ID.
	// CurrentCache is the active cache policy as comma-separated tokens
	// (e.g. "NR,WB").
	storcli2VDInfo struct {
		DGVD         string `json:"DG/VD"`
		Type         string `json:"TYPE"`
		State        string `json:"State"`
		CurrentCache string `json:"CurrentCache"`
		Size         string `json:"Size"`
	}

	// storcli2VDPhysicalDrive is one physical drive backing a virtual drive.
	storcli2VDPhysicalDrive struct {
		EIDSlot string `json:"EID:Slt"`
	}

	// storcli2VDProperties holds the path-related properties of a virtual drive.
	storcli2VDProperties struct {
		OSDriveName string `json:"OS Drive Name"`
		SCSINAAID   string `json:"SCSI NAA Id"`
	}
)

var _ ports.LogicalVolumesGetter = &StorCLI2{}

// NewStorCLI2 returns a logical-volume getter backed by the given storcli2 /
// perccli2 command runner.
func NewStorCLI2(runner commandrunner.CommandRunner) *StorCLI2 {
	return &StorCLI2{
		runner: runner,
	}
}

// LogicalVolumes returns every logical volume of the given controller, read
// from a single "/cN/vall show all" call.
func (s *StorCLI2) LogicalVolumes(metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume,
	error,
) {
	output, err := s.runner.Run([]string{
		fmt.Sprintf(storcli2AllVolumesSelector, metadata.ID),
		storcli2CmdShow,
		storcli2CmdAll,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show logical volumes for controller %d", metadata.ID)
	}

	vds, err := decodeVirtualDrives(output)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode logical volumes for controller %d", metadata.ID)
	}

	logicalVolumes := make([]*logicalvolume.LogicalVolume, 0, len(vds))

	for _, vd := range vds {
		logicalVolume, err := parseVirtualDrive(vd, metadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse logical volume %s", vd.Info.DGVD)
		}

		logicalVolumes = append(logicalVolumes, logicalVolume)
	}

	return logicalVolumes, nil
}

// LogicalVolume returns the logical volume addressed by the given metadata.
func (s *StorCLI2) LogicalVolume(metadata *logicalvolume.Metadata) (
	*logicalvolume.LogicalVolume,
	error,
) {
	output, err := s.runner.Run([]string{
		fmt.Sprintf(storcli2VolumeSelector, metadata.CtrlMetadata.ID, metadata.ID),
		storcli2CmdShow,
		storcli2CmdAll,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show details for logical volume %s", metadata.ID)
	}

	vds, err := decodeVirtualDrives(output)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode logical volume %s", metadata.ID)
	}

	if len(vds) == 0 {
		return nil, errors.Errorf("logical volume %s not found", metadata.ID)
	}

	logicalVolume, err := parseVirtualDrive(vds[0], metadata.CtrlMetadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse logical volume %s", metadata.ID)
	}

	return logicalVolume, nil
}

// decodeVirtualDrives decodes a storcli2 envelope and extracts its "Virtual
// Drives" section. Per the StorCLI2 User Guide, showing a nonexistent object
// reports success, so an absent section means an empty inventory (e.g. a
// controller whose drives are all JBOD), not an error; LogicalVolume()'s
// not-found guard then handles the single-volume case.
func decodeVirtualDrives(output []byte) ([]storcli2VirtualDrive, error) {
	cmd, err := storcli2.Decode(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode storcli2 output")
	}

	vds, err := utils.UnmarshalToSlice[storcli2VirtualDrive](
		cmd.Controllers[0].ResponseData, storcli2VirtualDrivesKey,
	)
	if err != nil {
		if errors.Is(err, utils.ErrKeyNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to unmarshal virtual drives")
	}

	return vds, nil
}

// parseVirtualDrive maps a storcli2 virtual drive to a LogicalVolume entity
// owned by the given controller.
func parseVirtualDrive(vd storcli2VirtualDrive, ctrl *raidcontroller.Metadata) (
	*logicalvolume.LogicalVolume,
	error,
) {
	id, err := parseVDID(vd.Info.DGVD)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse virtual drive id")
	}

	size, err := utils.ConvertSizeBytes(vd.Info.Size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert size")
	}

	pDrivesMetadata := make([]*physicaldrive.Metadata, 0, len(vd.PDs))
	for _, pd := range vd.PDs {
		pDrivesMetadata = append(pDrivesMetadata, &physicaldrive.Metadata{
			CtrlMetadata: ctrl,
			ID:           pd.EIDSlot,
		})
	}

	return &logicalvolume.LogicalVolume{
		Metadata: &logicalvolume.Metadata{
			CtrlMetadata: ctrl,
			ID:           id,
		},
		RAIDLevel:       logicalvolume.RAIDLevelMap(vd.Info.Type),
		PDrivesMetadata: pDrivesMetadata,
		CacheOptions:    parseCacheOptions(vd.Info.CurrentCache),
		Status:          lvStatus(vd.Info.State),
		Size:            size,
		DevicePath:      strings.TrimSpace(vd.Properties.OSDriveName),
		PermanentPath:   permanentPath(vd.Properties.SCSINAAID),
	}, nil
}

// parseVDID extracts the virtual drive number from a "<disk group>/<virtual
// drive>" pair such as "0/1". The disk group is not used.
func parseVDID(dgvd string) (string, error) {
	const expectedParts = 2

	parts := strings.Split(dgvd, "/")
	if len(parts) != expectedParts || parts[1] == "" {
		return "", errors.Errorf("invalid DG/VD value %q", dgvd)
	}

	return parts[1], nil
}

// lvStatus maps a storcli2 "State" value to an LVStatus. An offline virtual
// drive (the documented terminal state, e.g. a RAID0 that lost a member) is
// failed; a recovering one is still operational but not optimal, so it is
// degraded. Unknown states soft-fail to LVStatusUnknown rather than erroring,
// so firmware revisions that add states do not break inventory.
func lvStatus(state string) logicalvolume.LVStatus {
	switch {
	case strings.EqualFold(state, storcli2OptimalState):
		return logicalvolume.LVStatusOptimal
	case strings.EqualFold(state, storcli2DegradedState),
		strings.EqualFold(state, storcli2PartiallyDegradedState),
		strings.EqualFold(state, storcli2RecoveryState):
		return logicalvolume.LVStatusDegraded
	case strings.EqualFold(state, storcli2OfflineState),
		strings.EqualFold(state, storcli2FailedState):
		return logicalvolume.LVStatusFailed
	}

	return logicalvolume.LVStatusUnknown
}

// parseCacheOptions maps a storcli2 cache string to CacheOptions. storcli2
// reports the active policy as comma-separated tokens (e.g. "NR,WB"), unlike
// storcli1's concatenated form. Each token is matched independently; unknown or
// missing tokens leave the corresponding policy unknown.
func parseCacheOptions(cache string) *logicalvolume.CacheOptions {
	options := &logicalvolume.CacheOptions{
		ReadPolicy:  logicalvolume.ReadPolicyUnknown,
		WritePolicy: logicalvolume.WritePolicyUnknown,
		IOPolicy:    logicalvolume.IOPolicyUnknown,
	}

	for token := range strings.SplitSeq(cache, ",") {
		switch strings.ToUpper(strings.TrimSpace(token)) {
		case "R":
			options.ReadPolicy = logicalvolume.ReadPolicyReadAhead
		case "NR":
			options.ReadPolicy = logicalvolume.ReadPolicyNoReadAhead
		case "WB":
			options.WritePolicy = logicalvolume.WritePolicyWriteBack
		case "AWB":
			options.WritePolicy = logicalvolume.WritePolicyAlwaysWriteBack
		case "WT":
			options.WritePolicy = logicalvolume.WritePolicyWriteThrough
		case "C":
			options.IOPolicy = logicalvolume.IOPolicyCached
		case "D":
			options.IOPolicy = logicalvolume.IOPolicyDirect
		}
	}

	return options
}

// permanentPath builds the /dev/disk/by-id path of a virtual drive from its
// SCSI NAA Id. udev "wwn-" links are lowercase hex while the firmware is
// case-inconsistent across sections, so the id is lowercased. An empty id
// yields an empty path.
func permanentPath(naaID string) string {
	trimmed := strings.TrimSpace(naaID)
	if trimmed == "" {
		return ""
	}

	return storcli2PermanentPathPrefix + strings.ToLower(trimmed)
}
