package blinker

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/storcli2"
)

const (
	// storcli2CmdStart and storcli2CmdStop begin and end a drive locate.
	storcli2CmdStart = "start"
	storcli2CmdStop  = "stop"
	// storcli2CmdLocate is the locate (blink) operation token.
	storcli2CmdLocate = "locate"
	// storcli2EnclosureSelector and storcli2NoEnclosureSelector address a single
	// drive, with or without an enclosure component.
	storcli2EnclosureSelector   = "/c%d/e%s/s%s"
	storcli2NoEnclosureSelector = "/c%d/s%s"
)

// StorCLI2 blinks physical drives through a storcli2/perccli2 command runner. A
// single implementation serves both binaries; the concrete runner is injected
// at construction time.
type StorCLI2 struct {
	runner commandrunner.CommandRunner
}

var _ ports.Blinker = &StorCLI2{}

// NewStorCLI2 returns a blinker backed by the given storcli2 / perccli2 command
// runner.
func NewStorCLI2(runner commandrunner.CommandRunner) *StorCLI2 {
	return &StorCLI2{
		runner: runner,
	}
}

// StartBlink starts locating (blinking) a physical drive.
func (s *StorCLI2) StartBlink(metadata *physicaldrive.Metadata) error {
	if err := s.locate(metadata, storcli2CmdStart); err != nil {
		return errors.Wrap(err, "failed to start blinking physical drive")
	}

	return nil
}

// StopBlink stops locating (blinking) a physical drive.
func (s *StorCLI2) StopBlink(metadata *physicaldrive.Metadata) error {
	if err := s.locate(metadata, storcli2CmdStop); err != nil {
		return errors.Wrap(err, "failed to stop blinking physical drive")
	}

	return nil
}

// locate runs "<start|stop> locate" on a drive selector and surfaces the
// in-JSON failure that storcli2 may report regardless of its exit code.
func (s *StorCLI2) locate(metadata *physicaldrive.Metadata, action string) error {
	selector, err := storcli2SelectorPD(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to build drive selector")
	}

	output, err := s.runner.Run([]string{selector, action, storcli2CmdLocate})
	if err != nil {
		return errors.Wrapf(err, "failed to run %s locate command", action)
	}

	if _, err := storcli2.Decode(output); err != nil {
		return errors.Wrapf(err, "%s locate command failed", action)
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
