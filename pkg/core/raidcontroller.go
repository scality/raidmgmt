package core

import (
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/domain/ports"
)

type RAIDController struct {
	// TODO Add the necessary fields
	iface ports.RAIDController
}

var _ ports.RAIDController = &RAIDController{}

func NewRAIDController() *RAIDController {
	panic("not implemented")
}

func (r *RAIDController) Controllers() ([]*raidcontroller.RAIDController, error) {
	// log.info("in")
	// return r.iface.Controllers()
	panic("not implemented")
}

func (r *RAIDController) Controller(metadata *raidcontroller.Metadata) (*raidcontroller.RAIDController, error) {
	panic("not implemented")
}

func (r *RAIDController) PhysicalDrives(metadata *raidcontroller.Metadata) ([]*physicaldrive.PhysicalDrive, error) {
	panic("not implemented")
}

func (r *RAIDController) PhysicalDrive(metadata *physicaldrive.Metadata) (*physicaldrive.PhysicalDrive, error) {
	panic("not implemented")
}

func (r *RAIDController) LogicalVolumes(metadata *raidcontroller.Metadata) ([]*logicalvolume.LogicalVolume, error) {
	panic("not implemented")
}

func (r *RAIDController) LogicalVolume(metadata *logicalvolume.Metadata) (*logicalvolume.LogicalVolume, error) {
	panic("not implemented")
}

func (r *RAIDController) EnableJBOD(metadata *physicaldrive.Metadata) error {
	panic("not implemented")
}

func (r *RAIDController) DisableJBOD(metadata *physicaldrive.Metadata) error {
	panic("not implemented")
}

func (r *RAIDController) SetLVCacheOptions(metadata *logicalvolume.Metadata, cacheOpts *logicalvolume.CacheOptions) error {
	panic("not implemented")
}

func (r *RAIDController) CreateLV(lvRequest *logicalvolume.Request) (*logicalvolume.LogicalVolume, error) {
	panic("not implemented")
}

func (r *RAIDController) AddPVToLV(lvMetadata *logicalvolume.Metadata, pdMetadata *physicaldrive.Metadata) error {
	panic("not implemented")
}

func (r *RAIDController) DeleteLV(metadata *logicalvolume.Metadata) error {
	panic("not implemented")
}

func (r *RAIDController) DeletePVFromLV(lvMetadata *logicalvolume.Metadata, pdMetadata *physicaldrive.Metadata) error {
	panic("not implemented")
}

func (r *RAIDController) StartBlink(metadata *physicaldrive.Metadata) error {
	panic("not implemented")
}

func (r *RAIDController) StopBlink(metadata *physicaldrive.Metadata) error {
	panic("not implemented")
}
