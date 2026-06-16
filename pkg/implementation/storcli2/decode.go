package storcli2

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// statusSuccess is the value of "Command Status".Status on success.
const statusSuccess = "Success"

// Decode unmarshals the storcli2/perccli2 JSON envelope and surfaces in-JSON
// errors. The process exit code is not a reliable success signal (some
// failures exit 0, others exit non-zero while still writing the JSON
// payload), so errors are detected from each controller's
// "Command Status".Status instead.
func Decode(data []byte) (*CmdOutput, error) {
	var out CmdOutput

	if err := json.Unmarshal(data, &out); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal JSON")
	}

	if len(out.Controllers) == 0 {
		return nil, errors.New("no controllers found")
	}

	for i := range out.Controllers {
		status := out.Controllers[i].CommandStatus
		if status.Status != statusSuccess {
			return nil, errors.Wrap(parseError(status), "error running command")
		}
	}

	return &out, nil
}

// parseError builds an error from a failed command status, preferring the
// per-target "Detailed Status" message and falling back to the top-level
// description.
func parseError(status CommandStatus) error {
	for _, detail := range status.DetailedStatus {
		if detail.Status != statusSuccess && detail.ErrMsg != "" {
			return errors.New(detail.ErrMsg)
		}
	}

	return errors.New(status.Description)
}
