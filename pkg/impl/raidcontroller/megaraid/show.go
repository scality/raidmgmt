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
