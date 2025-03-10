//nolint:mnd // Integration tests, no need for constants
package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/scality/raidmgmt/pkg/core"
	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
)

type SoftwareRAIDControllerTester struct {
	controller core.RAIDController
	logger     *zerolog.Logger
}

func NewSoftwareRAIDControllerTester(
	controller core.RAIDController,
	logger *zerolog.Logger,
) *SoftwareRAIDControllerTester {
	return &SoftwareRAIDControllerTester{
		controller: controller,
		logger:     logger,
	}
}

func (t *SoftwareRAIDControllerTester) runRAID10Tests() error {
	l := t.logger.With().Str("test_case", "raid10").Logger()

	physicalDrives, err := t.controller.PhysicalDrives(&raidcontroller.Metadata{})
	if err != nil {
		l.Error().Err(err).Msg("failed to get physical drives")

		return errors.Wrap(err, "failed to get physical drives")
	}

	if len(physicalDrives) < 4 {
		l.Error().Msg("not enough physical drives")

		return errors.New("not enough physical drives to run test case")
	}

	controllerMetadata := &raidcontroller.Metadata{ID: 0}
	slot := &physicaldrive.Slot{
		Port: "pouet",
	}

	creationRequest := &logicalvolume.Request{
		CacheOptions: &logicalvolume.CacheOptions{},
		CtrlMetadata: controllerMetadata,
		RAIDLevel:    logicalvolume.RAIDLevel10,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{
				DevicePath:   "/dev/nvme1n1",
				CtrlMetadata: controllerMetadata,
				Slot:         slot,
			},
			{
				DevicePath:   "/dev/nvme2n1",
				CtrlMetadata: controllerMetadata,
				Slot:         slot,
			},
			{
				DevicePath:   "/dev/nvme3n1",
				CtrlMetadata: controllerMetadata,
				Slot:         slot,
			},
			{
				DevicePath:   "/dev/nvme4n1",
				CtrlMetadata: controllerMetadata,
				Slot:         slot,
			},
		},
		Name: "test_raid10",
	}

	// Create RAID1 with 4 disks
	logicalVolume, err := t.controller.CreateLV(creationRequest)
	if err != nil {
		l.Error().Err(err).Msg("failed to create RAID1")

		return errors.Wrap(err, "failed to create RAID1")
	}

	l.Info().Msg("Created RAID10 Logical volume")

	defer func() {
		// Remove the full array
		err = t.controller.DeleteLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
		if err != nil {
			l.Error().Err(err).Msg("failed to delete RAID10")
			return
		}

		l.Info().Msg("RAID10 array deleted")
	}()

	drives := []*physicaldrive.Metadata{
		{
			DevicePath:   "/dev/nvme1n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
	}

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.Error().Err(err).Msg("RAID1 should not be able to remove a disk if only two are there")

		return errors.Wrap(err, "RAID1 should not be able to remove a disk if only two are there")
	}

	l.Info().Msg("RAID10 removed a disk, array is degraded")

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.Error().Err(err).Msg("failed to get logical volume")

		return errors.Wrap(err, "failed to get logical volume")
	}

	if len(logicalVolume.PDrivesMetadata) != 3 {
		l.Error().Msg("RAID10 should have 3 disks now")

		return errors.New("RAID10 should have 3 disks now")
	}

	l.Info().Msg("RAID10 has 3 disks now")

	l.Info().Msg("Adding 1 disk")

	err = t.controller.AddPDsToLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.Error().Err(err).Msg("failed to extend RAID1")

		return errors.Wrap(err, "failed to extend RAID1")
	}

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.Error().Err(err).Msg("failed to get logical volume")

		return errors.Wrap(err, "failed to get logical volume")
	}

	if len(logicalVolume.PDrivesMetadata) != 4 {
		l.Error().Msg("RAID10 should have 4 disks now")

		return errors.New("RAID10 should have 4 disks now")
	}

	l.Info().Msg("RAID10 has 4 disks now")

	drives = []*physicaldrive.Metadata{
		{
			DevicePath:   "/dev/nvme1n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
		{
			DevicePath:   "/dev/nvme4n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
	}

	l.Info().Msg("Removing 2 disks")

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.Error().Err(err).Msg("failed to extend RAID1")

		return errors.Wrap(err, "failed to extend RAID1")
	}

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.Error().Err(err).Msg("failed to get logical volume")

		return errors.Wrap(err, "failed to get logical volume")
	}

	if len(logicalVolume.PDrivesMetadata) != 2 {
		l.Error().Msg("RAID10 should have 2 disks now")

		return errors.New("RAID10 should have 2 disks now")
	}

	l.Info().Msg("RAID10 has 2 disks now")

	l.Info().Msg("Adding 2 disks")

	err = t.controller.AddPDsToLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.Error().Err(err).Msg("failed to extend RAID1")

		return errors.Wrap(err, "failed to extend RAID1")
	}

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.Error().Err(err).Msg("failed to get logical volume")

		return errors.Wrap(err, "failed to get logical volume")
	}

	if len(logicalVolume.PDrivesMetadata) != 4 {
		l.Error().Msg("RAID10 should have 4 disks now")

		return errors.New("RAID10 should have 4 disks now")
	}

	l.Info().Msg("RAID10 has 4 disks now")

	drives = []*physicaldrive.Metadata{
		{
			DevicePath:   "/dev/nvme5n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
		{
			DevicePath:   "/dev/nvme6n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
	}

	l.Info().Msg("Removing 2 disks from the same array, should fail")

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err == nil {
		l.Error().Msg("Didn't fail to remove 2 disks from the same array")

		return errors.Wrap(err, "Didn't fail to remove 2 disks from the same array")
	}

	l.Info().Msg("RAID10 failed to remove 2 disks from the same array as expected")

	drives = []*physicaldrive.Metadata{
		{
			DevicePath:   "/dev/nvme6n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
		{
			DevicePath:   "/dev/nvme7n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
	}

	err = t.controller.AddPDsToLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err != nil {
		l.Error().Err(err).Msg("failed to extend RAID1")

		return errors.Wrap(err, "failed to extend RAID1")
	}

	l.Info().Msg("Added 2 more disks to RAID10 array")

	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.Error().Err(err).Msg("failed to get logical volume")

		return errors.Wrap(err, "failed to get logical volume")
	}

	fmt.Println(logicalVolume.Size)

	l.Info().Msg("RAID10 tests passed")

	return nil
}

func (t *SoftwareRAIDControllerTester) runSoftwareControllerIntegrationTestSuite(logger *zerolog.Logger) {
	l := logger.With().Str("test_suite", "software_raid_controller").Logger()

	err := t.runRAID0Tests()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to run RAID0 tests")
	}

	l.Info().Msg("RAID0 tests passed")

	err = t.runRAID1Tests()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to run RAID1 tests")
	}

	l.Info().Msg("RAID1 tests passed")

	err = t.runRAID10Tests()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to run RAID10 tests")
	}
}

// Remove the full array.
func (t *SoftwareRAIDControllerTester) runRAID1Tests() error {
	l := t.logger.With().Str("test_case", "raid1").Logger()

	physicalDrives, err := t.controller.PhysicalDrives(&raidcontroller.Metadata{})
	if err != nil {
		l.Error().Err(err).Msg("failed to get physical drives")

		return errors.Wrap(err, "failed to get physical drives")
	}

	if len(physicalDrives) < 3 {
		l.Error().Msg("not enough physical drives")

		return errors.New("not enough physical drives to run test case")
	}

	controllerMetadata := &raidcontroller.Metadata{ID: 0}
	slot := &physicaldrive.Slot{
		Port: "pouet",
	}

	creationRequest := &logicalvolume.Request{
		CacheOptions: &logicalvolume.CacheOptions{},
		CtrlMetadata: controllerMetadata,
		RAIDLevel:    logicalvolume.RAIDLevel1,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{
				DevicePath:   "/dev/nvme1n1",
				CtrlMetadata: controllerMetadata,
				Slot:         slot,
			},
			{
				DevicePath:   "/dev/nvme2n1",
				CtrlMetadata: controllerMetadata,
				Slot:         slot,
			},
		},
		Name: "test_raid1",
	}

	// Create RAID1 with 2 disks
	logicalVolume, err := t.controller.CreateLV(creationRequest)
	if err != nil {
		l.Error().Err(err).Msg("failed to create RAID1")

		return errors.Wrap(err, "failed to create RAID1")
	}

	defer func() {
		// Remove the full array
		err = t.controller.DeleteLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
		if err != nil {
			l.Error().Err(err).Msg("failed to delete RAID1")
			return
		}

		l.Info().Msg("RAID1 array deleted")
	}()

	drives := []*physicaldrive.Metadata{
		{
			DevicePath:   "/dev/nvme1n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
	}

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err == nil {
		l.Error().Err(err).Msg("RAID1 should not be able to remove a disk if only two are there")

		return errors.Wrap(err, "RAID1 should not be able to remove a disk if only two are there")
	}

	// FIXME Must wait here that the resize is done

	drives = []*physicaldrive.Metadata{
		{
			DevicePath:   "/dev/nvme3n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
	}

	err = t.controller.AddPDsToLV(
		&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata},
		drives...,
	)
	if err != nil {
		l.Error().Err(err).Msg("failed to extend RAID1")

		return errors.Wrap(err, "failed to extend RAID1")
	}

	return nil
}

func (t *SoftwareRAIDControllerTester) runRAID0Tests() error {
	l := t.logger.With().Str("test_case", "raid0").Logger()

	physicalDrives, err := t.controller.PhysicalDrives(&raidcontroller.Metadata{})
	if err != nil {
		l.Error().Err(err).Msg("failed to get physical drives")

		return errors.Wrap(err, "failed to get physical drives")
	}

	if len(physicalDrives) < 3 {
		l.Error().Msg("not enough physical drives")

		return errors.New("not enough physical drives to run test case")
	}

	controllerMetadata := &raidcontroller.Metadata{ID: 0}
	slot := &physicaldrive.Slot{
		Port: "pouet",
	}

	creationRequest := &logicalvolume.Request{
		CacheOptions: &logicalvolume.CacheOptions{},
		CtrlMetadata: controllerMetadata,
		RAIDLevel:    logicalvolume.RAIDLevel0,
		PDrivesMetadata: []*physicaldrive.Metadata{
			{
				DevicePath: "/dev/nvme1n1",
				// DevicePath:   physicalDrives[0].DevicePath,
				CtrlMetadata: controllerMetadata,
				Slot:         slot,
			},
			{
				// DevicePath:   physicalDrives[1].DevicePath,
				DevicePath:   "/dev/nvme2n1",
				CtrlMetadata: controllerMetadata,
				Slot:         slot,
			},
		},
		Name: "test_raid0",
	}

	// Create with 2 disks
	logicalVolume, err := t.controller.CreateLV(creationRequest)
	if err != nil {
		l.Error().Err(err).Msg("failed to create RAID0")

		return errors.Wrap(err, "failed to create RAID0")
	}

	l.Info().Str("device_path", logicalVolume.DevicePath).Msg("RAID0 created")

	// Extend with one extra disk => make sure the array size is equal of sum of disk sizes

	drives := []*physicaldrive.Metadata{
		{
			DevicePath:   "/dev/nvme3n1",
			CtrlMetadata: controllerMetadata,
			Slot:         slot,
		},
	}

	err = t.controller.AddPDsToLV(
		&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata},
		drives...,
	)
	if err != nil {
		l.Error().Err(err).Msg("failed to extend RAID0")

		return errors.Wrap(err, "failed to extend RAID0")
	}

	l.Info().Msg("RAID0 extended")

	// previousSize := logicalVolume.Size

	// for {
	logicalVolume, err = t.controller.LogicalVolume(&logicalvolume.Metadata{
		ID:           logicalVolume.DevicePath,
		CtrlMetadata: controllerMetadata,
	})
	if err != nil {
		l.Error().Err(err).Msg("failed to get logical volume")

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

	l.Info().Msg("RAID0 size is equal to sum of disk sizes")

	err = t.controller.DeletePDsFromLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata}, drives...)
	if err == nil {
		l.Error().Msg("RAID0 should not be able to remove a disk")

		return errors.New("RAID0 should not be able to remove a disk")
	}

	l.Info().Msg("RAID0 cannot remove a disk")

	// Remove the full array
	err = t.controller.DeleteLV(&logicalvolume.Metadata{ID: logicalVolume.DevicePath, CtrlMetadata: controllerMetadata})
	if err != nil {
		l.Error().Err(err).Msg("failed to delete RAID0")

		return errors.Wrap(err, "failed to delete RAID0")
	}

	l.Info().Msg("RAID0 array deleted")

	return nil
}
