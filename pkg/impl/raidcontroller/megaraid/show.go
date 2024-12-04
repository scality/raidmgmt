package megaraid

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// TODO
// const SHOWALL =

// ShowAll returns all information for all controllers.
func (a *Adapter) ShowAll() (*CmdOutput, error) {
	return a.cmd.Run([]string{"show", "all"})
}

// ShowAllController returns all information for a given controller.
func (a *Adapter) ShowAllController(controllerID string) (json.RawMessage, error) {
	output, err := a.cmd.Run([]string{fmt.Sprintf(patternController, controllerID), "show", "all"})
	if err != nil {
		return nil, err
	}

	ctrlIDInt, err := strconv.Atoi(controllerID)
	if err != nil {
		return nil, err
	}

	for i := range output.Controllers {
		if output.Controllers[i].CommandStatus.Controller == ctrlIDInt {
			return output.Controllers[i].ResponseData, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrControllerNotFound, controllerID)
}

// ShowAllPhysicalDrives returns all physical drives for a given controller.
func (m *Adapter) ShowAllPhysicalDrives(controllerID string) ([]PD, error) {
	responseData, err := m.ShowAllController(controllerID)
	if err != nil {
		return nil, err
	}

	return unmarshalToSlice[PD](responseData, "PD LIST")
}

// ShowDeviceAttributes returns the device attributes for a given physical drive.
func (m *Adapter) ShowDeviceAttributes(
	controllerID string, enclosureID, slotID int) (
	*DriveDeviceAttributes, error,
) {
	var (
		output   *CmdOutput
		err      error
		key      string
		selector string
	)

	isEnclosureIDInvalid := enclosureID < 0

	if isEnclosureIDInvalid {
		selector = fmt.Sprintf(patternNoEnclosure, controllerID, slotID)

		output, err = m.cmd.Run([]string{selector, "show", "all"})
		if err != nil {
			return nil, err
		}

		key = "Drive " + selector + " Device attributes"

	}

	ctrlIDInt, err := strconv.Atoi(controllerID)
	if err != nil {
		return nil, err
	}

	areAllIDsValid := ctrlIDInt >= 0 && enclosureID >= 0 && slotID >= 0
	if areAllIDsValid {
		selector = fmt.Sprintf(patternEnclosure, controllerID, enclosureID, slotID)
		output, err = m.cmd.Run([]string{selector, "show", "all"})

		key = "Drive " + selector + " Device attributes"
	}

	// Check the error after the if block to avoid shadowing the err variable
	// from the previous block
	// This allows us to test the error depending on the branch taken
	if err != nil {
		return nil, err
	}

	// Get the response data for the controller
	responseData, err := output.GetResponseDataByCtrlID(ctrlIDInt)
	if err != nil {
		return nil, err
	}

	return unmarshalToPointer[DriveDeviceAttributes](responseData, key)
}

// ShowAllVirtualDrives returns all logical drives for a given controller.
func (m *Adapter) ShowAllVirtualDrives(controllerID string) ([]VD, error) {
	responseData, err := m.ShowAllController(controllerID)
	if err != nil {
		return nil, err
	}

	return unmarshalToSlice[VD](responseData, "VD LIST")
}
