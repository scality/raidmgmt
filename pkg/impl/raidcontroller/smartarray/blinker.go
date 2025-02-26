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

	_, err := s.Run(args)
	if err != nil {
		return errors.Wrapf(err, "failed to blink physical drive %s", slot)
	}

	return nil
}
