package megaraid

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/scality/raidmgmt/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/utils"
)

// showAll returns all information for all controllers.
func (a *Adapter) showAll() (*CmdOutput, error) {
	output, err := a.runner.Run([]string{"show", "all"})
	if err != nil {
		return nil, errors.Wrap(err, ErrCommandFailed.Error())
	}

	return output, nil
}

// showAllController returns all information for a given controller.
func (a *Adapter) showAllController(controllerID int) (json.RawMessage, error) {
	selector := selectorCtrl(&raidcontroller.Metadata{ID: controllerID})

	output, err := a.runner.Run([]string{selector, "show", "all"})
	if err != nil {
		return nil, errors.Wrap(err, ErrCommandFailed.Error())
	}

	return output.Controllers[0].ResponseData, nil
}

// showAllPhysicalDrives returns all physical drives for a given controller.
func (a *Adapter) showAllPhysicalDrives(controllerID int) ([]PD, error) {
	responseData, err := a.showAllController(controllerID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to controller show all %d", controllerID)
	}

	pds, err := utils.UnmarshalToSlice[PD](responseData, "PD LIST")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get PD list")
	}

	return pds, nil
}

// showAllPhysicalDrive returns all information for a given physical drive.
func (a *Adapter) showAllPhysicalDrive(metadata *physicaldrive.Metadata) (json.RawMessage, error) {
	if err := validateID(metadata.Slot); err != nil {
		return nil, errors.Wrap(err, "failed to validate slot")
	}

	selector, err := selectorPD(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get selector")
	}

	output, err := a.runner.Run([]string{selector, "show", "all"})
	if err != nil {
		return nil, errors.Wrap(err, ErrCommandFailed.Error())
	}

	// Get the response data for the controller
	responseData := output.Controllers[0].ResponseData

	return responseData, nil
}

// showAllVirtualDrives returns all logical drives for a given controller.
func (a *Adapter) showAllVirtualDrives(controllerID int) ([]VD, error) {
	responseData, err := a.showAllController(controllerID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to controller show all %d", controllerID)
	}

	vds, err := utils.UnmarshalToSlice[VD](responseData, "VD LIST")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get VD list")
	}

	return vds, nil
}

// showAllVirtualDrive returns all logical drives for a given controller.
func (a *Adapter) showAllVirtualDrive(metadata *logicalvolume.Metadata) (json.RawMessage, error) {
	selector, err := selectorLV(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get selector")
	}

	output, err := a.runner.Run([]string{selector, "show", "all"})
	if err != nil {
		return nil, errors.Wrap(err, ErrCommandFailed.Error())
	}

	responseData := output.Controllers[0].ResponseData

	return responseData, nil
}
