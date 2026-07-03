//nolint:mnd // Integration tests, no need for constants
package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/core"
	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
)

// HardwareRAIDControllerTester drives the storcli2/perccli2 composition adapter
// against real hardware. Inventory is read-only; create/add/delete/scenario are
// destructive and are only reached once main has seen an explicit confirmation.
type HardwareRAIDControllerTester struct {
	controller   core.RAIDController
	controllerID int
	logger       *slog.Logger
}

func NewHardwareRAIDControllerTester(
	controller core.RAIDController,
	controllerID int,
	logger *slog.Logger,
) *HardwareRAIDControllerTester {
	return &HardwareRAIDControllerTester{
		controller:   controller,
		controllerID: controllerID,
		logger:       logger,
	}
}

// Inventory reads the controllers, physical drives and logical volumes and
// prints them as markdown tables. It mutates nothing.
func (t *HardwareRAIDControllerTester) Inventory(ctx context.Context) error {
	l := t.logger.With(slog.String("command", "inventory"))

	controllers, err := t.controller.Controllers()
	if err != nil {
		return errors.Wrap(err, "failed to get controllers")
	}

	ctrlMetadata := &raidcontroller.Metadata{ID: t.controllerID}

	physicalDrives, err := t.controller.PhysicalDrives(ctrlMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get physical drives")
	}

	logicalVolumes, err := t.controller.LogicalVolumes(ctrlMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get logical volumes")
	}

	printControllers(controllers)
	printPhysicalDrives(physicalDrives)
	printLogicalVolumes(logicalVolumes)

	l.InfoContext(ctx, "inventory complete",
		slog.Int("controllers", len(controllers)),
		slog.Int("physical_drives", len(physicalDrives)),
		slog.Int("logical_volumes", len(logicalVolumes)),
	)

	return nil
}

// Create creates a logical volume from the given RAID level and drives.
func (t *HardwareRAIDControllerTester) Create(
	ctx context.Context,
	level logicalvolume.RAIDLevel,
	drives []*physicaldrive.Metadata,
) (*logicalvolume.LogicalVolume, error) {
	l := t.logger.With(slog.String("command", "create"))

	request := &logicalvolume.Request{
		CtrlMetadata:    &raidcontroller.Metadata{ID: t.controllerID},
		RAIDLevel:       level,
		PDrivesMetadata: drives,
		// storcli2 has no IO policy; Direct satisfies the request validation and
		// is ignored by the adapter.
		CacheOptions: &logicalvolume.CacheOptions{
			ReadPolicy:  logicalvolume.ReadPolicyReadAhead,
			WritePolicy: logicalvolume.WritePolicyWriteBack,
			IOPolicy:    logicalvolume.IOPolicyDirect,
		},
		Name: "raidmgmt_e2e",
	}

	logicalVolume, err := t.controller.CreateLV(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logical volume")
	}

	l.InfoContext(ctx, "created logical volume",
		slog.String("id", logicalVolume.ID),
		slog.String("raid_level", logicalVolume.RAIDLevel.String()),
		slog.String("device_path", logicalVolume.DevicePath),
	)

	return logicalVolume, nil
}

// Add expands the given volume with the given drives (online capacity
// expansion).
func (t *HardwareRAIDControllerTester) Add(
	ctx context.Context,
	vdID string,
	drives []*physicaldrive.Metadata,
) error {
	l := t.logger.With(slog.String("command", "add"))

	metadata := &logicalvolume.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{ID: t.controllerID},
		ID:           vdID,
	}

	if err := t.controller.AddPDsToLV(metadata, drives...); err != nil {
		return errors.Wrapf(err, "failed to expand logical volume %s", vdID)
	}

	l.InfoContext(ctx, "expanded logical volume", slog.String("id", vdID))

	return nil
}

// Delete deletes (clears) the given volume.
func (t *HardwareRAIDControllerTester) Delete(ctx context.Context, vdID string) error {
	l := t.logger.With(slog.String("command", "delete"))

	metadata := &logicalvolume.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{ID: t.controllerID},
		ID:           vdID,
	}

	if err := t.controller.DeleteLV(metadata); err != nil {
		return errors.Wrapf(err, "failed to delete logical volume %s", vdID)
	}

	l.InfoContext(ctx, "deleted logical volume", slog.String("id", vdID))

	return nil
}

// Scenario runs the full create -> assert-remove-unsupported -> (optional)
// expand -> delete cycle, leaving the controller as it was found. Drive removal
// is asserted to be unsupported on storcli2, so it is exercised as a negative
// case rather than a mutation.
func (t *HardwareRAIDControllerTester) Scenario(
	ctx context.Context,
	level logicalvolume.RAIDLevel,
	drives []*physicaldrive.Metadata,
	addDrives []*physicaldrive.Metadata,
) (err error) {
	l := t.logger.With(slog.String("command", "scenario"))

	logicalVolume, err := t.Create(ctx, level, drives)
	if err != nil {
		return err
	}

	defer func() {
		if deleteErr := t.Delete(ctx, logicalVolume.ID); deleteErr != nil && err == nil {
			err = deleteErr
		}
	}()

	removeErr := t.controller.DeletePDsFromLV(logicalVolume.Metadata, drives[0])
	if !errors.Is(removeErr, ports.ErrFunctionNotSupportedByImplementation) {
		return errors.Errorf("expected drive removal to be unsupported, got: %v", removeErr)
	}

	l.InfoContext(ctx, "drive removal is unsupported as expected")

	if len(addDrives) > 0 {
		if err := t.Add(ctx, logicalVolume.ID, addDrives); err != nil {
			return err
		}

		expanded, err := t.controller.LogicalVolume(logicalVolume.Metadata)
		if err != nil {
			return errors.Wrap(err, "failed to re-read expanded logical volume")
		}

		l.InfoContext(ctx, "expanded logical volume drive count",
			slog.Int("drives", len(expanded.PDrivesMetadata)),
		)
	}

	l.InfoContext(ctx, "scenario passed")

	return err
}

// printControllers prints the controllers as a markdown table.
func printControllers(controllers []*raidcontroller.RAIDController) {
	fmt.Println("\n### Controllers")
	fmt.Println("| ID | Name | Serial | JBOD supported | JBOD enabled |")
	fmt.Println("|---|---|---|---|---|")

	for _, c := range controllers {
		fmt.Printf("| %d | %s | %s | %t | %t |\n",
			c.ID, c.Name, c.Serial, c.IsJBODSupported, c.IsJBODEnabled)
	}
}

// printPhysicalDrives prints the physical drives as a markdown table.
func printPhysicalDrives(drives []*physicaldrive.PhysicalDrive) {
	fmt.Println("\n### Physical drives")
	fmt.Println("| ID | Slot | Model | Size | Type | Status | JBOD |")
	fmt.Println("|---|---|---|---|---|---|---|")

	for _, d := range drives {
		fmt.Printf("| %s | %s | %s | %s | %s | %s | %t |\n",
			d.ID, d.Slot.String(), d.Model, humanBytes(d.Size), d.Type, d.Status, d.JBOD)
	}
}

// printLogicalVolumes prints the logical volumes as a markdown table.
func printLogicalVolumes(volumes []*logicalvolume.LogicalVolume) {
	fmt.Println("\n### Logical volumes")
	fmt.Println("| ID | RAID | Status | Size | Drives | Device path |")
	fmt.Println("|---|---|---|---|---|---|")

	for _, v := range volumes {
		ids := make([]string, 0, len(v.PDrivesMetadata))
		for _, pd := range v.PDrivesMetadata {
			ids = append(ids, pd.ID)
		}

		fmt.Printf("| %s | %s | %s | %s | %s | %s |\n",
			v.ID, v.RAIDLevel.String(), v.Status, humanBytes(v.Size),
			strings.Join(ids, " "), v.DevicePath)
	}
}

// humanBytes renders a byte count in binary units for readable tables.
func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
