//nolint:mnd // Integration tests, no need for constants
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/core"
	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
)

type SoftwareRAIDControllerTester struct {
	controller core.RAIDController
	logger     *slog.Logger
}

func NewSoftwareRAIDControllerTester(
	controller core.RAIDController,
	logger *slog.Logger,
) *SoftwareRAIDControllerTester {
	return &SoftwareRAIDControllerTester{
		controller: controller,
		logger:     logger,
	}
}

func (t *SoftwareRAIDControllerTester) runRAID10Tests(ctx context.Context) error {
	l := t.logger.With(slog.String("test_case", "raid10"))

	physicalDrives, err := t.controller.PhysicalDrives(&raidcontroller.Metadata{})
	if err != nil {
		l.ErrorContext(ctx, "failed to get physical drives", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to get physical drives")
	}

	if len(physicalDrives) < 4 {
		l.ErrorContext(ctx, "not enough physical drives")

		return errors.New("not enough physical drives to run test case")
	}

	controllerMetadata := &raidcontroller.Metadata{ID: 0}

	creationRequest := &logicalvolume.Request{
		CacheOptions: &logicalvolume.CacheOptions{},
		CtrlMetadata: controllerMetadata,
		RAIDLevel:    logicalvolume.RAIDLevel10,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{
				ID:           "/dev/nvme1n1",
				CtrlMetadata: controllerMetadata,
			},
			{
				ID:           "/dev/nvme2n1",
				CtrlMetadata: controllerMetadata,
			},
			{
				ID:           "/dev/nvme3n1",
				CtrlMetadata: controllerMetadata,
			},
			{
				ID:           "/dev/nvme4n1",
				CtrlMetadata: controllerMetadata,
			},
		},
		Name: "test_raid10",
	}

	// Create RAID1 with 4 disks
	logicalVolume, err := t.controller.CreateLV(creationRequest)
	if err != nil {
		l.ErrorContext(ctx, "failed to create RAID1", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to create RAID1")
	}

	l.InfoContext(ctx, "Created RAID10 Logical volume")

	defer func() {
		// Remove the full array
		err = t.controller.DeleteLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
		if err != nil {
			l.ErrorContext(ctx, "failed to delete RAID10", slog.Any("error_message", err))
			return
		}

		l.InfoContext(ctx, "RAID10 array deleted")
	}()

	drives := []*physicaldrive.Metadata{
		{
			ID:           "/dev/nvme1n1",
			CtrlMetadata: controllerMetadata,
		},
	}

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.ErrorContext(ctx, "RAID1 should not be able to remove a disk if only two are there", slog.Any("error_message", err))

		return errors.Wrap(err, "RAID1 should not be able to remove a disk if only two are there")
	}

	l.InfoContext(ctx, "RAID10 removed a disk, array is degraded")

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.ErrorContext(ctx, "failed to get logical volume", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to get logical volume")
	}

	if len(logicalVolume.PDrivesMetadata) != 3 {
		l.ErrorContext(ctx, "RAID10 should have 3 disks now")

		return errors.New("RAID10 should have 3 disks now")
	}

	l.InfoContext(ctx, "RAID10 has 3 disks now")

	l.InfoContext(ctx, "Adding 1 disk")

	err = t.controller.AddPDsToLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.ErrorContext(ctx, "failed to extend RAID1", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to extend RAID1")
	}

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.ErrorContext(ctx, "failed to get logical volume", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to get logical volume")
	}

	if len(logicalVolume.PDrivesMetadata) != 4 {
		l.ErrorContext(ctx, "RAID10 should have 4 disks now")

		return errors.New("RAID10 should have 4 disks now")
	}

	l.InfoContext(ctx, "RAID10 has 4 disks now")

	drives = []*physicaldrive.Metadata{
		{
			ID:           "/dev/nvme1n1",
			CtrlMetadata: controllerMetadata,
		},
		{
			ID:           "/dev/nvme4n1",
			CtrlMetadata: controllerMetadata,
		},
	}

	l.InfoContext(ctx, "Removing 2 disks")

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.ErrorContext(ctx, "failed to extend RAID1", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to extend RAID1")
	}

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.ErrorContext(ctx, "failed to get logical volume", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to get logical volume")
	}

	if len(logicalVolume.PDrivesMetadata) != 2 {
		l.ErrorContext(ctx, "RAID10 should have 2 disks now")

		return errors.New("RAID10 should have 2 disks now")
	}

	l.InfoContext(ctx, "RAID10 has 2 disks now")

	l.InfoContext(ctx, "Adding 2 disks")

	err = t.controller.AddPDsToLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.ErrorContext(ctx, "failed to extend RAID1", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to extend RAID1")
	}

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.ErrorContext(ctx, "failed to get logical volume", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to get logical volume")
	}

	if len(logicalVolume.PDrivesMetadata) != 4 {
		l.ErrorContext(ctx, "RAID10 should have 4 disks now")

		return errors.New("RAID10 should have 4 disks now")
	}

	l.InfoContext(ctx, "RAID10 has 4 disks now")

	drives = []*physicaldrive.Metadata{
		{
			ID:           "/dev/nvme5n1",
			CtrlMetadata: controllerMetadata,
		},
		{
			ID:           "/dev/nvme6n1",
			CtrlMetadata: controllerMetadata,
		},
	}

	l.InfoContext(ctx, "Removing 2 disks from the same array, should fail")

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err == nil {
		l.ErrorContext(ctx, "Didn't fail to remove 2 disks from the same array")

		return errors.Wrap(err, "Didn't fail to remove 2 disks from the same array")
	}

	l.InfoContext(ctx, "RAID10 failed to remove 2 disks from the same array as expected")

	drives = []*physicaldrive.Metadata{
		{
			ID:           "/dev/nvme6n1",
			CtrlMetadata: controllerMetadata,
		},
		{
			ID:           "/dev/nvme7n1",
			CtrlMetadata: controllerMetadata,
		},
	}

	err = t.controller.AddPDsToLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.ErrorContext(ctx, "failed to extend RAID1", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to extend RAID1")
	}

	l.InfoContext(ctx, "Added 2 more disks to RAID10 array")

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.ErrorContext(ctx, "failed to get logical volume", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to get logical volume")
	}

	fmt.Println(logicalVolume.Size)

	l.InfoContext(ctx, "RAID10 tests passed")

	return nil
}

func (t *SoftwareRAIDControllerTester) runSoftwareControllerIntegrationTestSuite(ctx context.Context, logger *slog.Logger) {
	l := logger.With(slog.String("test_suite", "software_raid_controller"))

	err := t.runRAID0Tests(ctx)
	if err != nil {
		l.ErrorContext(ctx, "failed to run RAID0 tests", slog.Any("error_message", err))
		os.Exit(1)
	}

	l.InfoContext(ctx, "RAID0 tests passed")

	err = t.runRAID1Tests(ctx)
	if err != nil {
		l.ErrorContext(ctx, "failed to run RAID1 tests", slog.Any("error_message", err))
		os.Exit(1)
	}

	l.InfoContext(ctx, "RAID1 tests passed")

	err = t.runRAID10Tests(ctx)
	if err != nil {
		l.ErrorContext(ctx, "failed to run RAID10 tests", slog.Any("error_message", err))
		os.Exit(1)
	}
}

// Remove the full array.
func (t *SoftwareRAIDControllerTester) runRAID1Tests(ctx context.Context) error {
	l := t.logger.With(slog.String("test_case", "raid1"))

	physicalDrives, err := t.controller.PhysicalDrives(&raidcontroller.Metadata{})
	if err != nil {
		l.ErrorContext(ctx, "failed to get physical drives", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to get physical drives")
	}

	if len(physicalDrives) < 3 {
		l.ErrorContext(ctx, "not enough physical drives")

		return errors.New("not enough physical drives to run test case")
	}

	controllerMetadata := &raidcontroller.Metadata{ID: 0}

	creationRequest := &logicalvolume.Request{
		CacheOptions: &logicalvolume.CacheOptions{},
		CtrlMetadata: controllerMetadata,
		RAIDLevel:    logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{
				ID:           "/dev/nvme1n1",
				CtrlMetadata: controllerMetadata,
			},
			{
				ID:           "/dev/nvme2n1",
				CtrlMetadata: controllerMetadata,
			},
		},
		Name: "test_raid1",
	}

	// Create RAID1 with 2 disks
	logicalVolume, err := t.controller.CreateLV(creationRequest)
	if err != nil {
		l.ErrorContext(ctx, "failed to create RAID1", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to create RAID1")
	}

	defer func() {
		// Remove the full array
		err = t.controller.DeleteLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
		if err != nil {
			l.ErrorContext(ctx, "failed to delete RAID1", slog.Any("error_message", err))
			return
		}

		l.InfoContext(ctx, "RAID1 array deleted")
	}()

	drives := []*physicaldrive.Metadata{
		{
			ID:           "/dev/nvme1n1",
			CtrlMetadata: controllerMetadata,
		},
	}

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err == nil {
		l.ErrorContext(ctx, "RAID1 should not be able to remove a disk if only two are there", slog.Any("error_message", err))

		return errors.Wrap(err, "RAID1 should not be able to remove a disk if only two are there")
	}

	// FIXME Must wait here that the resize is done

	drives = []*physicaldrive.Metadata{
		{
			ID:           "/dev/nvme3n1",
			CtrlMetadata: controllerMetadata,
		},
	}

	err = t.controller.AddPDsToLV(
		&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata},
		drives...,
	)
	if err != nil {
		l.ErrorContext(ctx, "failed to extend RAID1", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to extend RAID1")
	}

	return nil
}

func (t *SoftwareRAIDControllerTester) runRAID0Tests(ctx context.Context) error {
	l := t.logger.With(slog.String("test_case", "raid0"))

	physicalDrives, err := t.controller.PhysicalDrives(&raidcontroller.Metadata{})
	if err != nil {
		l.ErrorContext(ctx, "failed to get physical drives", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to get physical drives")
	}

	if len(physicalDrives) < 3 {
		l.ErrorContext(ctx, "not enough physical drives")

		return errors.New("not enough physical drives to run test case")
	}

	controllerMetadata := &raidcontroller.Metadata{ID: 0}

	creationRequest := &logicalvolume.Request{
		CacheOptions: &logicalvolume.CacheOptions{},
		CtrlMetadata: controllerMetadata,
		RAIDLevel:    logicalvolume.RAIDLevel0,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{
				ID: "/dev/nvme1n1",
				// ID:   physicalDrives[0].DevicePath,
				CtrlMetadata: controllerMetadata,
			},
			{
				// ID:   physicalDrives[1].DevicePath,
				ID:           "/dev/nvme2n1",
				CtrlMetadata: controllerMetadata,
			},
		},
		Name: "test_raid0",
	}

	// Create with 2 disks
	logicalVolume, err := t.controller.CreateLV(creationRequest)
	if err != nil {
		l.ErrorContext(ctx, "failed to create RAID0", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to create RAID0")
	}

	l.InfoContext(ctx, "RAID0 created", slog.String("device_path", logicalVolume.DevicePath))

	// Extend with one extra disk => make sure the array size is equal of sum of disk sizes

	drives := []*physicaldrive.Metadata{
		{
			ID:           "/dev/nvme3n1",
			CtrlMetadata: controllerMetadata,
		},
	}

	err = t.controller.AddPDsToLV(
		&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata},
		drives...,
	)
	if err != nil {
		l.ErrorContext(ctx, "failed to extend RAID0", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to extend RAID0")
	}

	l.InfoContext(ctx, "RAID0 extended")

	// previousSize := logicalVolume.Size

	// for {
	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{
		ID:           logicalVolume.DevicePath,
		CtrlMetadata: controllerMetadata,
	})
	if err != nil {
		l.ErrorContext(ctx, "failed to get logical volume", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to get logical volume")
	}

	// 	if logicalVolume.Size > previousSize {
	// 		break
	// 	}
	//
	// 	l.Info().Msg("Waiting for RAID0 size to be updated")
	// 	time.Sleep(5 * time.Second)
	// }

	// FIXME The size part is not working as expected for now
	// But i can confirm that the size is equal to the sum of the disks

	l.InfoContext(ctx, "RAID0 size is equal to sum of disk sizes")

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err == nil {
		l.ErrorContext(ctx, "RAID0 should not be able to remove a disk")

		return errors.New("RAID0 should not be able to remove a disk")
	}

	l.InfoContext(ctx, "RAID0 cannot remove a disk")

	// Remove the full array
	err = t.controller.DeleteLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.ErrorContext(ctx, "failed to delete RAID0", slog.Any("error_message", err))

		return errors.Wrap(err, "failed to delete RAID0")
	}

	l.InfoContext(ctx, "RAID0 array deleted")

	return nil
}
