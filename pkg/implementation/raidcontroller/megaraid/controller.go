package megaraid

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/utils"
)

// patternController is the pattern for the controller selector.
const patternController string = "/c%d"

// controllers returns a list of RAID controllers.
func (a *Adapter) controllers() ([]*raidcontroller.RAIDController, error) {
	output, err := a.showAll()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get controllers information")
	}

	raidControllers := make([]*raidcontroller.RAIDController, 0)

	for _, controller := range output.Controllers {
		// Get the system overview for the controller
		// This is needed to get the controller ID
		systemOverview, err := utils.UnmarshalToSlice[SystemOverview](
			controller.ResponseData, "System Overview",
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"failed to get system overview for controller %d",
				controller.CommandStatus.Controller,
			)
		}

		metadata := raidcontroller.Metadata{
			// Take the index 0 because there is only one system overview
			// for a controller
			ID: systemOverview[0].Ctl,
		}

		// Get the controller
		ctrl, err := a.controller(&metadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get controller %d", metadata.ID)
		}

		// Append the controller to the list
		raidControllers = append(raidControllers, ctrl)
	}

	return raidControllers, nil
}

// controller returns a RAID controller for a given metadata.
func (a *Adapter) controller(metadata *raidcontroller.Metadata) (
	*raidcontroller.RAIDController,
	error,
) {
	selector := selectorCtrl(metadata)

	responseData, err := a.showAllController(selector)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get controller %d", metadata.ID)
	}

	// Get the basics for the controller
	// This is needed to get the controller model and serial number
	basics, err := utils.UnmarshalToPointer[Basics](responseData, "Basics")
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal basics")
	}

	// Get the supported adapter operations for the controller
	// This is needed to check if the controller supports JBOD
	supportedAdapterOperations, err := utils.UnmarshalToPointer[SupportedAdapterOperations](
		responseData, "Supported Adapter Operations",
	)
	if err != nil {
		return nil, errors.Wrap(err,
			"failed to unmarshal supported adapter operations")
	}

	isJBODSupported := strings.ToLower(supportedAdapterOperations.SupportJBOD) == "yes"

	isJBODEnabled := false

	if isJBODSupported {
		// Get the capabilities for the controller
		// This is needed to check if JBOD is enabled
		capabilities, err := utils.UnmarshalToPointer[Capabilities](responseData, "Capabilities")
		if err != nil {
			return nil, errors.Wrap(err,
				"failed to unmarshal capabilities")
		}

		isJBODEnabled = strings.ToLower(capabilities.EnableJBOD) == "yes"
	}

	return &raidcontroller.RAIDController{
		Metadata:        metadata,
		Name:            strings.TrimSpace(basics.Model),
		Serial:          strings.TrimSpace(basics.SerialNumber),
		IsJBODSupported: isJBODSupported,
		IsJBODEnabled:   isJBODEnabled,
	}, nil
}

// selectorCtrl returns the selector for a controller.
func selectorCtrl(m *raidcontroller.Metadata) string {
	return fmt.Sprintf(patternController, m.ID)
}
