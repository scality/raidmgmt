package megaraid

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

const patternController string = "/c%s"

// controllers returns a list of RAID controllers.
func (a *Adapter) controllers() ([]*raidcontroller.RAIDController, error) {
	output, err := a.ShowAll()
	if err != nil {
		return nil, err
	}

	raidControllers := make([]*raidcontroller.RAIDController, 0)

	for _, controller := range output.Controllers {
		// Get the system overview for the controller
		// This is needed to get the controller ID
		systemOverview, err := unmarshalToSlice[SystemOverview](
			controller.ResponseData, "System Overview",
		)
		if err != nil {
			return nil, err
		}

		sysOvCtrlID := strconv.Itoa(systemOverview[0].Ctl)

		ctrl, err := a.ControllerByID(sysOvCtrlID)
		if err != nil {
			return nil, err
		}

		raidControllers = append(raidControllers, ctrl)
	}

	return raidControllers, nil
}

// ControllerByID returns a RAID controller by its ID.
func (a *Adapter) ControllerByID(ctrlID string) (*raidcontroller.RAIDController, error) {
	responseData, err := a.ShowAllController(ctrlID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrControllerNotFound, err)
	}

	// Get the basics for the controller
	// This is needed to get the controller model and serial number
	basics, err := unmarshalToPointer[Basics](responseData, "Basics")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrControllerNotFound, err)
	}

	return &raidcontroller.RAIDController{
		Metadata: &raidcontroller.Metadata{
			ID: ctrlID,
		},
		Name:   strings.Trim(basics.Model, " "),
		Serial: strings.Trim(basics.SerialNumber, " "),
	}, nil
}

func selectorCtrl(m *raidcontroller.Metadata) string {
	return fmt.Sprintf(patternController, m.ID)
}
