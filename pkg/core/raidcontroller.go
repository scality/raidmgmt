package core

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/domain/ports"
)

type RAIDController struct {
	log   zerolog.Logger
	iface ports.RAIDController
}

var _ ports.RAIDController = &RAIDController{}

// New returns a new RAID controller.
func New(log zerolog.Logger, iface ports.RAIDController) *RAIDController {
	return &RAIDController{
		log:   log,
		iface: iface,
	}
}

// Controllers returns a list of RAID controllers.
func (r *RAIDController) Controllers() ([]*raidcontroller.RAIDController, error) {
	r.log.Debug().Msg("Getting RAID controllers")

	controllers, err := r.iface.Controllers()
	if err != nil {
		r.log.Error().
			Err(err).
			Msg("Failed to get RAID controllers")

		return nil, errors.Wrap(err, "failed to get RAID controllers")
	}

	return controllers, nil
}

// Controller returns a RAID controller for a given metadata.
func (r *RAIDController) Controller(metadata *raidcontroller.Metadata) (
	*raidcontroller.RAIDController,
	error,
) {
	r.log.Debug().
		Int("id", metadata.ID).
		Msg("Getting RAID controller")

	controller, err := r.iface.Controller(metadata)
	if err != nil {
		r.log.Error().
			Int("id", metadata.ID).
			Err(err).
			Msg("Failed to get RAID controller")

		return nil, errors.Wrap(err, "failed to get RAID controller")
	}

	return controller, nil
}

// PhysicalDrives returns a list of physical drives for a given RAID controller.
func (r *RAIDController) PhysicalDrives(metadata *raidcontroller.Metadata) (
	[]*physicaldrive.PhysicalDrive,
	error,
) {
	r.log.Debug().
		Int("id", metadata.ID).
		Msg("Getting physical drives")

	physicalDrives, err := r.iface.PhysicalDrives(metadata)
	if err != nil {
		r.log.Error().
			Err(err).
			Int("id", metadata.ID).
			Msg("Failed to get physical drives")

		return nil, errors.Wrap(err, "failed to get physical drives")
	}

	return physicalDrives, nil
}

// PhysicalDrive returns a physical drive for a given metadata.
func (r *RAIDController) PhysicalDrive(metadata *physicaldrive.Metadata) (
	*physicaldrive.PhysicalDrive,
	error,
) {
	slot := metadata.Slot.String()

	r.log.Debug().
		Str("slot", slot).
		Msg("Getting physical drive")

	physicalDrive, err := r.iface.PhysicalDrive(metadata)
	if err != nil {
		r.log.Error().
			Err(err).
			Str("slot", slot).
			Msg("Failed to get physical drive")

		return nil, errors.Wrap(err, "failed to get physical drive")
	}

	return physicalDrive, nil
}

// LogicalVolumes returns a list of logical volumes for a given RAID controller.
func (r *RAIDController) LogicalVolumes(metadata *raidcontroller.Metadata) (
	[]*logicalvolume.LogicalVolume,
	error,
) {
	r.log.Debug().
		Int("id", metadata.ID).
		Msg("Getting logical volumes")

	logicalVolumes, err := r.iface.LogicalVolumes(metadata)
	if err != nil {
		r.log.Error().
			Err(err).
			Int("id", metadata.ID).
			Msg("Failed to get logical volumes")

		return nil, errors.Wrap(err, "failed to get logical volumes")
	}

	return logicalVolumes, nil
}

// LogicalVolume returns a logical volume for a given metadata.
func (r *RAIDController) LogicalVolume(metadata *logicalvolume.Metadata) (
	*logicalvolume.LogicalVolume,
	error,
) {
	r.log.Debug().
		Str("id", metadata.ID).
		Msg("Getting logical volume")

	logicalVolume, err := r.iface.LogicalVolume(metadata)
	if err != nil {
		r.log.Error().
			Err(err).
			Str("id", metadata.ID).
			Msg("Failed to get logical volume")

		return nil, errors.Wrap(err, "failed to get logical volume")
	}

	return logicalVolume, nil
}

// EnableJBOD enables JBOD mode on a physical drive.
func (r *RAIDController) EnableJBOD(metadata *physicaldrive.Metadata) error {
	slot := metadata.Slot.String()

	r.log.Debug().
		Str("slot", slot).
		Msg("Enabling JBOD for physical drive")

	if err := r.iface.EnableJBOD(metadata); err != nil {
		r.log.Error().
			Err(err).
			Str("slot", slot).
			Msg("Failed to enable JBOD for physical drive")

		return errors.Wrap(err, "failed to enable JBOD for physical drive")
	}

	return nil
}

// DisableJBOD disables JBOD mode on a physical drive.
func (r *RAIDController) DisableJBOD(metadata *physicaldrive.Metadata) error {
	slot := metadata.Slot.String()

	r.log.Debug().
		Str("slot", slot).
		Msg("Disabling JBOD for physical drive")

	if err := r.iface.DisableJBOD(metadata); err != nil {
		r.log.Error().
			Err(err).
			Str("slot", slot).
			Msg("Failed to disable JBOD for physical drive")

		return errors.Wrap(err, "failed to disable JBOD for physical drive")
	}

	return nil
}

// SetLVCacheOptions sets cache options on a logical volume.
func (r *RAIDController) SetLVCacheOptions(
	metadata *logicalvolume.Metadata,
	cacheOpts *logicalvolume.CacheOptions,
) error {
	r.log.Debug().
		Str("id", metadata.ID).
		Msg("Setting cache options for logical volume")

	if err := r.iface.SetLVCacheOptions(metadata, cacheOpts); err != nil {
		r.log.Error().
			Err(err).
			Str("id", metadata.ID).
			Msg("Failed to set cache options for logical volume")

		return errors.Wrap(err, "failed to set cache options for logical volume")
	}

	return nil
}

// CreateLV creates a logical volume from a request.
func (r *RAIDController) CreateLV(lvRequest *logicalvolume.Request) (
	*logicalvolume.LogicalVolume,
	error,
) {
	r.log.Debug().Msg("Creating logical volume")

	newLV, err := r.iface.CreateLV(lvRequest)
	if err != nil {
		r.log.Error().
			Err(err).
			Msg("Failed to create logical volume")

		return nil, errors.Wrap(err, "failed to create logical volume")
	}

	return newLV, nil
}

// AddPDToLV adds a physical drive to a logical volume.
func (r *RAIDController) AddPDToLV(
	lvMetadata *logicalvolume.Metadata,
	pdMetadata *physicaldrive.Metadata,
) error {
	slot := pdMetadata.Slot.String()

	r.log.Debug().
		Str("lv_id", lvMetadata.ID).
		Str("slot", slot).
		Msg("Adding physical drive to logical volume")

	if err := r.iface.AddPDToLV(lvMetadata, pdMetadata); err != nil {
		r.log.Error().
			Err(err).
			Str("lv_id", lvMetadata.ID).
			Str("slot", slot).
			Msg("Failed to add physical drive to logical volume")

		return errors.Wrap(err, "failed to add physical drive to logical volume")
	}

	return nil
}

// DeleteLV deletes a logical volume.
func (r *RAIDController) DeleteLV(metadata *logicalvolume.Metadata) error {
	r.log.Debug().
		Str("id", metadata.ID).
		Msg("Deleting logical volume")

	if err := r.iface.DeleteLV(metadata); err != nil {
		r.log.Error().
			Err(err).
			Str("id", metadata.ID).
			Msg("Failed to delete logical volume")

		return errors.Wrap(err, "failed to delete logical volume")
	}

	return nil
}

// DeletePDFromLV deletes a physical drive from a logical volume.
func (r *RAIDController) DeletePDFromLV(
	lvMetadata *logicalvolume.Metadata,
	pdMetadata *physicaldrive.Metadata,
) error {
	slot := pdMetadata.Slot.String()

	r.log.Debug().
		Str("lv_id", lvMetadata.ID).
		Str("slot", slot).
		Msg("Removing physical drive from logical volume")

	if err := r.iface.DeletePDFromLV(lvMetadata, pdMetadata); err != nil {
		r.log.Error().
			Err(err).
			Str("lv_id", lvMetadata.ID).
			Str("slot", slot).
			Msg("Failed to remove physical drive from logical volume")

		return errors.Wrap(err, "failed to remove physical drive from logical volume")
	}

	return nil
}

// StartBlink starts blinking for a physical drive.
func (r *RAIDController) StartBlink(metadata *physicaldrive.Metadata) error {
	slot := metadata.Slot.String()

	r.log.Debug().
		Str("slot", slot).
		Msg("Starting blinking for physical drive")

	if err := r.iface.StartBlink(metadata); err != nil {
		r.log.Error().
			Err(err).
			Str("slot", slot).
			Msg("Failed to start blinking for physical drive")

		return errors.Wrap(err, "failed to start blinking for physical drive")
	}

	return nil
}

// StopBlink stops blinking for a physical drive.
func (r *RAIDController) StopBlink(metadata *physicaldrive.Metadata) error {
	slot := metadata.Slot.String()

	r.log.Debug().
		Str("slot", slot).
		Msg("Stopping blinking for physical drive")

	if err := r.iface.StopBlink(metadata); err != nil {
		r.log.Error().
			Err(err).
			Str("slot", slot).
			Msg("Failed to stop blinking for physical drive")

		return errors.Wrap(err, "failed to stop blinking for physical drive")
	}

	return nil
}
