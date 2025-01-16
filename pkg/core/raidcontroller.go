package core

import (
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
	return r.iface.Controllers()
}

// Controller returns a RAID controller for a given metadata.
func (r *RAIDController) Controller(metadata *raidcontroller.Metadata) (*raidcontroller.RAIDController, error) {
	r.log.Debug().Msgf("Getting RAID controller %d", metadata.ID)
	return r.iface.Controller(metadata)
}

// PhysicalDrives returns a list of physical drives for a given RAID controller.
func (r *RAIDController) PhysicalDrives(metadata *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error) {
	r.log.Debug().Msgf("Getting physical drives for RAID controller %d", metadata.ID)
	return r.iface.PhysicalDrives(metadata)
}

// PhysicalDrive returns a physical drive for a given metadata.
func (r *RAIDController) PhysicalDrive(metadata *physicaldrive.Metadata) (*physicaldrive.PhysicalDrive, error) {
	r.log.Debug().Msgf("Getting physical drive %s", metadata.Slot.String())
	return r.iface.PhysicalDrive(metadata)
}

// LogicalVolumes returns a list of logical volumes for a given RAID controller.
func (r *RAIDController) LogicalVolumes(metadata *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error) {
	r.log.Debug().Msgf("Getting logical volumes for RAID controller %d", metadata.ID)
	return r.iface.LogicalVolumes(metadata)
}

// LogicalVolume returns a logical volume for a given metadata.
func (r *RAIDController) LogicalVolume(metadata *logicalvolume.Metadata) (*logicalvolume.LogicalVolume, error) {
	r.log.Debug().Msgf("Getting logical volume %s", metadata.ID)
	return r.iface.LogicalVolume(metadata)
}

// EnableJBOD enables JBOD mode on a physical drive.
func (r *RAIDController) EnableJBOD(metadata *physicaldrive.Metadata) error {
	r.log.Debug().Msgf("Enabling JBOD for physical drive %s", metadata.Slot.String())
	return r.iface.EnableJBOD(metadata)
}

// DisableJBOD disables JBOD mode on a physical drive.
func (r *RAIDController) DisableJBOD(metadata *physicaldrive.Metadata) error {
	r.log.Debug().Msgf("Disabling JBOD for physical drive %s", metadata.Slot.String())
	return r.iface.DisableJBOD(metadata)
}

// SetLVCacheOptions sets cache options on a logical volume.
func (r *RAIDController) SetLVCacheOptions(metadata *logicalvolume.Metadata, cacheOpts *logicalvolume.CacheOptions) error {
	r.log.Debug().Msgf("Setting cache options for logical volume %s", metadata.ID)
	return r.iface.SetLVCacheOptions(metadata, cacheOpts)
}

// CreateLV creates a logical volume from a request.
func (r *RAIDController) CreateLV(lvRequest *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	r.log.Debug().Msg("Creating logical volume")
	return r.iface.CreateLV(lvRequest)
}

// AddPDToLV adds a physical drive to a logical volume.
func (r *RAIDController) AddPDToLV(lvMetadata *logicalvolume.Metadata, pdMetadata *physicaldrive.Metadata) error {
	r.log.Debug().Msgf("Adding physical drive %s to logical volume %s", pdMetadata.Slot.String(), lvMetadata.ID)
	return r.iface.AddPDToLV(lvMetadata, pdMetadata)
}

// DeleteLV deletes a logical volume.
func (r *RAIDController) DeleteLV(metadata *logicalvolume.Metadata) error {
	r.log.Debug().Msgf("Deleting logical volume %s", metadata.ID)
	return r.iface.DeleteLV(metadata)
}

// DeletePDFromLV deletes a physical drive from a logical volume.
func (r *RAIDController) DeletePDFromLV(lvMetadata *logicalvolume.Metadata, pdMetadata *physicaldrive.Metadata) error {
	r.log.Debug().Msgf("Removing physical drive %s from logical volume %s", pdMetadata.Slot.String(), lvMetadata.ID)
	return r.iface.DeletePDFromLV(lvMetadata, pdMetadata)
}

// StartBlink starts blinking for a physical drive.
func (r *RAIDController) StartBlink(metadata *physicaldrive.Metadata) error {
	r.log.Debug().Msgf("Starting blinking for physical drive %s", metadata.Slot.String())
	return r.iface.StartBlink(metadata)
}

// StopBlink stops blinking for a physical drive.
func (r *RAIDController) StopBlink(metadata *physicaldrive.Metadata) error {
	r.log.Debug().Msgf("Stopping blinking for physical drive %s", metadata.Slot.String())
	return r.iface.StopBlink(metadata)
}
