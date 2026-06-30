package jbodsetter

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/storcli2"
)

const (
	// storcli2CmdSet is the storcli2 "set" command token.
	storcli2CmdSet = "set"
	// storcli2JBODOption converts a drive to the JBOD state.
	storcli2JBODOption = "jbod"
	// storcli2UConfOption converts a drive back to the unconfigured state;
	// storcli2 dropped storcli's "delete jbod".
	storcli2UConfOption = "uconf"

	// storcli2EnclosureSelector and storcli2NoEnclosureSelector address a single
	// drive, with or without an enclosure component.
	storcli2EnclosureSelector   = "/c%d/e%s/s%s"
	storcli2NoEnclosureSelector = "/c%d/s%s"
)

// StorCLI2 sets the JBOD state of a physical drive through a storcli2 /
// perccli2 command runner. A single implementation serves both binaries; the
// concrete runner is injected at construction time.
type StorCLI2 struct {
	runner commandrunner.CommandRunner
}

var _ ports.JBODSetter = &StorCLI2{}

// NewStorCLI2 returns a JBOD setter backed by the given storcli2 / perccli2
// command runner.
func NewStorCLI2(runner commandrunner.CommandRunner) *StorCLI2 {
	return &StorCLI2{
		runner: runner,
	}
}

// EnableJBOD converts a drive to the JBOD state ("set jbod"). It changes only
// the drive state, not its status.
func (s *StorCLI2) EnableJBOD(metadata *physicaldrive.Metadata) error {
	if err := s.setDriveState(metadata, storcli2JBODOption); err != nil {
		return errors.Wrap(err, "failed to enable JBOD")
	}

	return nil
}

// DisableJBOD converts a JBOD drive back to the unconfigured state
// ("set uconf"); storcli's "delete jbod" no longer parses.
func (s *StorCLI2) DisableJBOD(metadata *physicaldrive.Metadata) error {
	if err := s.setDriveState(metadata, storcli2UConfOption); err != nil {
		return errors.Wrap(err, "failed to disable JBOD")
	}

	return nil
}

// setDriveState runs "set <state>" on a drive selector and surfaces the in-JSON
// failure that storcli2 may report regardless of its exit code.
func (s *StorCLI2) setDriveState(metadata *physicaldrive.Metadata, state string) error {
	selector, err := storcli2SelectorPD(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to build drive selector")
	}

	output, err := s.runner.Run([]string{selector, storcli2CmdSet, state})
	if err != nil {
		return errors.Wrapf(err, "failed to run set %s command", state)
	}

	if _, err := storcli2.Decode(output); err != nil {
		return errors.Wrapf(err, "set %s command failed", state)
	}

	return nil
}

// storcli2SelectorPD builds the storcli2 selector for a drive, choosing the
// enclosure or no-enclosure form from its parsed slot.
func storcli2SelectorPD(metadata *physicaldrive.Metadata) (string, error) {
	slot, err := physicaldrive.ParseSlot(metadata.ID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse slot %s", metadata.ID)
	}

	if slot.Enclosure != "" {
		return fmt.Sprintf(
			storcli2EnclosureSelector, metadata.CtrlMetadata.ID, slot.Enclosure, slot.Bay,
		), nil
	}

	return fmt.Sprintf(storcli2NoEnclosureSelector, metadata.CtrlMetadata.ID, slot.Bay), nil
}
