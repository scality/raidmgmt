package core

import (
	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
)

// physicalDrives returns a list of physical drives for a given RAID controller.
func physicalDrives(iface ports.PhysicalDrivesGetter, metadata *raidcontroller.Metadata) (
	[]*physicaldrive.PhysicalDrive,
	error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, errors.Wrap(err, ErrInvalidRAIDControllerMetadata.Error())
	}

	physicalDrives, err := iface.PhysicalDrives(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get physical drives")
	}

	return physicalDrives, nil
}

func physicalDrive(iface ports.PhysicalDrivesGetter, metadata *physicaldrive.Metadata) (
	*physicaldrive.PhysicalDrive,
	error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
	}

	physicalDrive, err := iface.PhysicalDrive(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get physical drive")
	}

	return physicalDrive, nil
}

func logicalVolumes(iface ports.LogicalVolumesGetter, metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume,
	error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, errors.Wrap(err, ErrInvalidRAIDControllerMetadata.Error())
	}

	logicalVolumes, err := iface.LogicalVolumes(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical volumes")
	}

	return logicalVolumes, nil
}

func logicalVolume(iface ports.LogicalVolumesGetter, metadata *logicalvolume.Metadata) (
	*logicalvolume.LogicalVolume,
	error,
) {
	if err := metadata.Validate(); err != nil {
		return nil, errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	logicalVolume, err := iface.LogicalVolume(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get logical volume: %s", metadata.ID)
	}

	return logicalVolume, nil
}

func createLV(iface ports.LogicalVolumesManager, request *logicalvolume.Request) (
	*logicalvolume.LogicalVolume,
	error,
) {
	err := request.Validate()
	if err != nil {
		return nil, errors.Wrap(err, ErrInvalidLogicalVolumeRequest.Error())
	}

	logicalVolume, err := iface.CreateLV(request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create logical volume: %s", request.Name)
	}

	return logicalVolume, nil
}

func deleteLV(iface ports.LogicalVolumesManager, metadata *logicalvolume.Metadata) error {
	err := metadata.Validate()
	if err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	err = iface.DeleteLV(metadata)
	if err != nil {
		return errors.Wrapf(err, "failed to delete logical volume: %s", metadata.ID)
	}

	return nil
}

func addPDsToLV(
	iface ports.LogicalVolumesManager,
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	if err := lvMetadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	for _, pd := range pdsMetadata {
		if err := pd.Validate(); err != nil {
			return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
		}
	}

	err := iface.AddPDsToLV(lvMetadata, pdsMetadata...)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to add physical drives to logical volume : %s",
			lvMetadata.ID,
		)
	}

	return nil
}

func deletePDsFromLV(
	iface ports.LogicalVolumesManager,
	lvMetadata *logicalvolume.Metadata,
	pdsMetadata ...*physicaldrive.Metadata,
) error {
	if err := lvMetadata.Validate(); err != nil {
		return errors.Wrap(err, ErrInvalidLogicalVolumeMetadata.Error())
	}

	for _, pdMetadata := range pdsMetadata {
		if err := pdMetadata.Validate(); err != nil {
			return errors.Wrap(err, ErrInvalidPhysicalDriveMetadata.Error())
		}
	}

	err := iface.DeletePDsFromLV(lvMetadata, pdsMetadata...)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to delete physical drives from logical volume: %s",
			lvMetadata.ID,
		)
	}

	return nil
}
