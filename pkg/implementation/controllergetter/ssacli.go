package controllergetter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/utils"
)

const (
	// Capture leading whitespace.
	ssacliLeadingWhitespaceRegexpPattern = `^(\s*)`
	ssacliNameRegexpPattern              = `HPE Smart Array (.*?) in Slot \d+`
)

type SSACLI struct {
	SSACLI commandrunner.CommandRunner
}

var (
	_ ports.ControllersGetter = &SSACLI{}

	ssacliLeadingWhitespaceRegexp = regexp.MustCompile(ssacliLeadingWhitespaceRegexpPattern)
	ssacliNameRegexp              = regexp.MustCompile(ssacliNameRegexpPattern)
)

func NewSSACLI(ssacli *commandrunner.SSACLI) *SSACLI {
	return &SSACLI{
		SSACLI: ssacli,
	}
}

// Controllers returns a list of RAID controllers.
func (s *SSACLI) Controllers() ([]*raidcontroller.RAIDController, error) {
	output, err := s.SSACLI.Run([]string{
		"controller",
		"all",
		"show",
		"detail",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to show all controllers details")
	}

	controllers, err := parseControllers(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse controllers details")
	}

	return controllers, nil
}

// Controller returns a RAID controller for a given metadata.
func (s *SSACLI) Controller(metadata *raidcontroller.Metadata) (
	*raidcontroller.RAIDController,
	error,
) {
	args := []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.ID),
		"show",
		"detail",
	}

	output, err := s.SSACLI.Run(args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show details for controller %d", metadata.ID)
	}

	controller, err := parseController(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse controller")
	}

	return controller, nil
}

func parseControllers(output []byte) ([]*raidcontroller.RAIDController, error) {
	blocks := utils.SplitOutput(ssacliLeadingWhitespaceRegexp, output)

	controllers := make([]*raidcontroller.RAIDController, 0, len(blocks))

	for _, block := range blocks {
		controller, err := parseController(block)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse controller: %s", block)
		}

		controllers = append(controllers, controller)
	}

	return controllers, nil
}

// parseController parses a controller block and returns a RAIDController entity.
func parseController(block []byte) (*raidcontroller.RAIDController, error) {
	controller := &raidcontroller.RAIDController{
		Metadata: &raidcontroller.Metadata{},
	}

	for line := range strings.SplitSeq(string(block), "\n") {
		if err := parseControllerLine(controller, line); err != nil {
			return nil, errors.Wrapf(err, "failed to parse controller line: %s", line)
		}
	}

	return controller, nil
}

// parseControllerLine parses a line of a controller block and updates the RAIDController entity.
func parseControllerLine(controller *raidcontroller.RAIDController, line string) error {
	if ssacliNameRegexp.FindStringSubmatch(line) != nil {
		controller.Name = ssacliNameRegexp.FindStringSubmatch(line)[1]

		return nil
	}

	key, value := utils.ParseLineDetail(line)

	switch key {
	case "Serial Number":
		controller.Serial = value

	case "Slot":
		idInt, err := strconv.Atoi(value)
		if err != nil {
			return errors.Wrap(err, "failed to convert controller slot ID to int")
		}

		controller.ID = idInt
	}

	return nil
}
