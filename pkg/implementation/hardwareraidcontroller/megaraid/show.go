package megaraid

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/utils"
)

// showAll returns all information for all controllers.
func (a *Adapter) showAll() (*CmdOutput, error) {
	output, err := a.runner.Run([]string{"show", "all"})
	if err != nil {
		return nil, errors.Wrap(err, ErrCommandFailed.Error())
	}

	fmt.Println(string(output.Controllers[0].ResponseData))

	return output, nil
}

// showAllController returns all information for a given controller.
func (a *Adapter) showAllController(selector string) (json.RawMessage, error) {
	output, err := a.runner.Run([]string{selector, "show", "all"})
	if err != nil {
		return nil, errors.Wrap(err, ErrCommandFailed.Error())
	}

	return output.Controllers[0].ResponseData, nil
}

// showAllPhysicalDrives returns all physical drives for a given controller.
func (a *Adapter) showAllPhysicalDrives(selector string) ([]PD, error) {
	responseData, err := a.showAllController(selector)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to controller show all %s", selector)
	}

	pds, err := utils.UnmarshalToSlice[PD](responseData, "PD LIST")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get PD list")
	}

	return pds, nil
}

// showAllPhysicalDrive returns all information for a given physical drive.
func (a *Adapter) showAllPhysicalDrive(selector string) (json.RawMessage, error) {
	output, err := a.runner.Run([]string{selector, "show", "all"})
	if err != nil {
		return nil, errors.Wrap(err, ErrCommandFailed.Error())
	}

	// Get the response data for the controller
	responseData := output.Controllers[0].ResponseData

	return responseData, nil
}

// showAllVirtualDrives returns all logical drives for a given controller.
func (a *Adapter) showAllVirtualDrives(selector string) ([]VD, error) {
	responseData, err := a.showAllController(selector)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to controller show all %s", selector)
	}

	vds, err := utils.UnmarshalToSlice[VD](responseData, "VD LIST")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get VD list")
	}

	return vds, nil
}

// showAllVirtualDrive returns all logical drives for a given controller.
func (a *Adapter) showAllVirtualDrive(selector string) (json.RawMessage, error) {
	output, err := a.runner.Run([]string{selector, "show", "all"})
	if err != nil {
		return nil, errors.Wrap(err, ErrCommandFailed.Error())
	}

	responseData := output.Controllers[0].ResponseData

	return responseData, nil
}
