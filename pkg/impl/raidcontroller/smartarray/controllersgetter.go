package smartarray

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

const (
	// Capture leading whitespace.
	leadingWhitespacePattern = `^(\s*)`
	namePattern              = `HPE Smart Array (.*?) in Slot \d+`

	keyValueParts = 2
)

var (
	leadingWhitespaceRegexp = regexp.MustCompile(leadingWhitespacePattern)
	nameRegexp              = regexp.MustCompile(namePattern)
)

func parseControllers(output []byte) ([]*raidcontroller.RAIDController, error) {
	blocks := splitOutput(leadingWhitespaceRegexp, output)

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

	for _, line := range strings.Split(string(block), "\n") {
		if err := parseControllerLine(controller, line); err != nil {
			return nil, errors.Wrapf(err, "failed to parse controller line: %s", line)
		}
	}

	return controller, nil
}

// parseControllerLine parses a line of a controller block and updates the RAIDController entity.
func parseControllerLine(controller *raidcontroller.RAIDController, line string) error {
	if nameRegexp.FindStringSubmatch(line) != nil {
		controller.Name = nameRegexp.FindStringSubmatch(line)[1]

		return nil
	}

	key, value := parseLineDetail(line)

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
