package raidcontrollerservice

import (
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicalvolume"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

type RAIDControllerService interface {
	// GetControllers returns a list of RAID controllers
	GetControllers() ([]raidcontroller.RAIDController, error)
	// GetPhysicalVolumes returns a list of physical volumes for a given RAID controller
	GetPhysicalVolumes(controller raidcontroller.RAIDController) ([]physicalvolume.PhysicalVolume, error)
	// GetLogicalVolumes returns a list of logical volumes for a given RAID controller
	GetLogicalVolumes(controller raidcontroller.RAIDController) ([]logicalvolume.LogicalVolume, error)
	// UpdatePV updates a physical volume
	UpdatePV(pv physicalvolume.PhysicalVolume) error
	// UpdateLV updates a logical volume
	UpdateLV(lv logicalvolume.LogicalVolume) error
	// CreateLV creates a logical volume
	CreateLV(lv logicalvolume.LogicalVolume) (logicalvolume.LogicalVolume, error)
	// AddPVToLV adds a physical volume to a logical volume
	AddPVToLV(lv logicalvolume.LogicalVolume, pv physicalvolume.PhysicalVolume) error
	// DeleteLV deletes a logical volume
	DeleteLV(lv logicalvolume.LogicalVolume) error
	// DeletePVFromLV deletes a physical volume from a logical volume
	DeletePVFromLV(lv logicalvolume.LogicalVolume, pv physicalvolume.PhysicalVolume) error
	// StartBlink starts blinking a physical volume
	StartBlink(pv physicalvolume.PhysicalVolume) error
	// StopBlink stops blinking a physical volume
	StopBlink(pv physicalvolume.PhysicalVolume) error
}
