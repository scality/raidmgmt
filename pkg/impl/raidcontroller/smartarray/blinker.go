package smartarray

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/scality/raidmgmt/domain/entities/physicaldrive"
)

// blink makes a physical drive blink.
func (s *SSACLI) blink(metadata *physicaldrive.Metadata, action string) error {
	slot := formatSlot(metadata.Slot)

	args := []string{
		"controller",
		"slot=" + strconv.Itoa(metadata.CtrlMetadata.ID),
		"physicaldrive",
		slot,
		"modify",
		"led=" + action,
	}

	_, err := s.CommandRunner.Run(args)
	if err != nil {
		return errors.Wrapf(err, "failed to blink physical drive %s", slot)
	}

	return nil
}

// StartBlink starts blinking a physical drive.
func (s *SSACLI) StartBlink(metadata *physicaldrive.Metadata) error {
	err := s.blink(metadata, "on")
	if err != nil {
		return errors.Wrap(err, "failed to start blinking physical drive")
	}

	return nil
}

// StopBlink stops blinking a physical drive.
func (s *SSACLI) StopBlink(metadata *physicaldrive.Metadata) error {
	err := s.blink(metadata, "off")
	if err != nil {
		return errors.Wrap(err, "failed to stop blinking physical drive")
	}

	return nil
}
