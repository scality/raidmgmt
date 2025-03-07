package controllergetter

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
)

const (
	// Capture leading whitespace.
	leadingWhitespaceRegexpPattern = `^(\s*)`
	nameRegexpPattern              = `HPE Smart Array (.*?) in Slot \d+`
	keyValueParts                  = 2
)

type SSACLI struct {
	commandrunner.CommandRunner
}

var (
	_                       ports.ControllersGetter = &SSACLI{}
	leadingWhitespaceRegexp                         = regexp.MustCompile(leadingWhitespaceRegexpPattern)
	nameRegexp                                      = regexp.MustCompile(nameRegexpPattern)
)

func NewSSACLI(commandRunner commandrunner.CommandRunner) *SSACLI {
	return &SSACLI{
		CommandRunner: commandRunner,
	}
}

// Controllers returns a list of RAID controllers.
func (s *SSACLI) Controllers() ([]*raidcontroller.RAIDController, error) {
	output, err := s.CommandRunner.Run([]string{
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

	output, err := s.CommandRunner.Run(args)
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

	for line := range strings.SplitSeq(string(block), "\n") {
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

// splitOutput splits the output into blocks based on the regular expression.
// TODO add tests.
func splitOutput(regularExpression *regexp.Regexp, output []byte) [][]byte {
	indices := regularExpression.FindAllIndex(output, -1)
	if indices == nil {
		return nil // No matches found
	}

	var blocks [][]byte

	start := 0

	for i, match := range indices {
		if i == 0 {
			continue // Skip the first match
		}

		block := output[start:match[0]] // everything before the match
		if len(block) > 0 {             // avoid empty blocks
			blocks = append(blocks, bytes.TrimSpace(block)) // trim space here
		}

		start = match[0] // Start of the next block is the current match
	}
	// Add the last block if any
	if start < len(output) {
		blocks = append(blocks, bytes.TrimSpace(output[start:]))
	}

	return blocks
}

// parseLineDetail parses a line of the show detail command and returns the key and value.
func parseLineDetail(line string) (key, value string) {
	if line == "" {
		return "", ""
	}

	splitParts := strings.Split(line, ":")

	if len(splitParts) != keyValueParts {
		return "", ""
	}

	key = strings.TrimSpace(splitParts[0])
	value = strings.TrimSpace(splitParts[1])

	return key, value
}
