package core

import (
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicalvolume"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/domain/ports/raidcontrollerservice"
)

type RAIDControllerService struct {
	// TODO: Add the necessary fields
}

var _ raidcontrollerservice.RAIDControllerService = &RAIDControllerService{}

// NewRAIDControllerService creates a new RAIDControllerService
func NewRAIDControllerService() *RAIDControllerService {
	// TODO: Implement the NewRAIDControllerService function
	return &RAIDControllerService{}
}

func (r *RAIDControllerService) GetControllers() ([]raidcontroller.RAIDController, error) {
	// TODO: Implement the GetControllers method
	return nil, nil
}

func (r *RAIDControllerService) GetPhysicalVolumes(controller raidcontroller.RAIDController) ([]physicalvolume.PhysicalVolume, error) {
	// TODO: Implement the GetPhysicalVolumes method
	return nil, nil
}

func (r *RAIDControllerService) GetLogicalVolumes(controller raidcontroller.RAIDController) ([]logicalvolume.LogicalVolume, error) {
	// TODO: Implement the GetLogicalVolumes method
	return nil, nil
}

func (r *RAIDControllerService) UpdatePV(pv physicalvolume.PhysicalVolume) error {
	// TODO: Implement the UpdatePV method
	return nil
}

func (r *RAIDControllerService) UpdateLV(lv logicalvolume.LogicalVolume) error {
	// TODO: Implement the UpdateLV method
	return nil
}

func (r *RAIDControllerService) CreateLV(lv logicalvolume.LogicalVolume) (logicalvolume.LogicalVolume, error) {
	// TODO: Implement the CreateLV method
	return logicalvolume.LogicalVolume{}, nil
}

func (r *RAIDControllerService) AddPVToLV(lv logicalvolume.LogicalVolume, pv physicalvolume.PhysicalVolume) error {
	// TODO: Implement the AddPVToLV method
	return nil
}

func (r *RAIDControllerService) DeleteLV(lv logicalvolume.LogicalVolume) error {
	// TODO: Implement the DeleteLV method
	return nil
}

func (r *RAIDControllerService) DeletePVFromLV(lv logicalvolume.LogicalVolume, pv physicalvolume.PhysicalVolume) error {
	// TODO: Implement the DeletePVFromLV method
	return nil
}

func (r *RAIDControllerService) StartBlink(pv physicalvolume.PhysicalVolume) error {
	// TODO: Implement the StartBlink method
	return nil
}

func (r *RAIDControllerService) StopBlink(pv physicalvolume.PhysicalVolume) error {
	// TODO: Implement the StopBlink method
	return nil
}
