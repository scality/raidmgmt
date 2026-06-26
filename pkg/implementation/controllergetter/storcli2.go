package controllergetter

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/storcli2"
	"github.com/scality/raidmgmt/pkg/utils"
)

const (
	// storcli2ControllerSelector is the storcli2 selector for a single
	// controller, formatted with the controller index.
	storcli2ControllerSelector = "/c%d"

	// storcli2CmdShow, storcli2CmdAll, storcli2CmdAdvancedSoftwareOptions and
	// storcli2CmdAutoConfig are storcli2 command tokens.
	storcli2CmdShow                    = "show"
	storcli2CmdAll                     = "all"
	storcli2CmdAdvancedSoftwareOptions = "aso"
	storcli2CmdAutoConfig              = "autoconfig"

	// storcli2AdvancedSoftwareOptionsKey is the "Response Data" key listing the
	// licensed Advanced Software Options, and storcli2AutoConfigKey the one
	// describing the auto-configure behavior.
	storcli2AdvancedSoftwareOptionsKey = "Advanced Software options"
	storcli2AutoConfigKey              = "Auto-config Information"

	// storcli2JBODOption is the "Software option" name of the JBOD license and
	// also the auto-configure behavior that exposes drives as JBOD.
	storcli2JBODOption = "JBOD"
	// storcli2SecureJBODBehavior is the secure variant of the JBOD auto-configure
	// behavior.
	storcli2SecureJBODBehavior = "SecureJBOD"
	// storcli2OptionUnsupported is the "Time Remaining" marker of an Advanced
	// Software Option the controller cannot use: per the StorCLI2 User Guide
	// v1.1 the value is "Unlimited" or a days-and-hours countdown, optionally
	// suffixed with "(unsupported)".
	storcli2OptionUnsupported = "unsupported"
	// storcli2OptionExpired is the "Time Remaining" value of an expired Advanced
	// Software Option license. It is not in the User Guide v1.1 vocabulary
	// (storcli1 had it) but is kept as a defensive guard.
	storcli2OptionExpired = "Expired"
	// storcli2PrimaryAutoConfigProp is the auto-configure property that holds the
	// behavior applied to new drives.
	storcli2PrimaryAutoConfigProp = "Primary Auto-configure behavior"
)

type (
	// StorCLI2 reads RAID controller information through a storcli2 / perccli2
	// command runner. A single implementation serves both binaries since they
	// share the same JSON schema; the concrete runner is injected at
	// construction time.
	StorCLI2 struct {
		runner commandrunner.CommandRunner
	}

	// storcli2SystemOverview is one entry of the global "show all" "System
	// Overview" section. Only the controller index is needed here to address
	// each controller individually.
	storcli2SystemOverview struct {
		Ctrl int `json:"Ctrl"`
	}

	// storcli2Basics holds the controller identity fields from "/cN show all".
	storcli2Basics struct {
		ProductName  string `json:"Product Name"`
		SerialNumber string `json:"Serial Number"`
	}

	// storcli2AdvancedSoftwareOption is one entry of the "Advanced Software
	// options" section from "/cN show aso". A non-expired "JBOD" entry means the
	// controller is licensed for JBOD.
	storcli2AdvancedSoftwareOption struct {
		SoftwareOption string `json:"Software option"`
		TimeRemaining  string `json:"Time Remaining"`
	}

	// storcli2AutoConfigEntry is one entry of the "Auto-config Information"
	// section from "/cN show autoconfig".
	storcli2AutoConfigEntry struct {
		Property string `json:"Auto-config property"`
		Value    string `json:"Value"`
	}
)

var _ ports.ControllersGetter = &StorCLI2{}

// NewStorCLI2 returns a controller getter backed by the given storcli2 /
// perccli2 command runner.
func NewStorCLI2(runner commandrunner.CommandRunner) *StorCLI2 {
	return &StorCLI2{
		runner: runner,
	}
}

// Controllers returns every RAID controller managed by the adapter.
func (s *StorCLI2) Controllers() ([]*raidcontroller.RAIDController, error) {
	output, err := s.runner.Run([]string{storcli2CmdShow, storcli2CmdAll})
	if err != nil {
		return nil, errors.Wrap(err, "failed to show all controllers")
	}

	cmd, err := storcli2.Decode(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode controllers output")
	}

	controllers := make([]*raidcontroller.RAIDController, 0)

	for i := range cmd.Controllers {
		found, err := s.controllersFromOverview(cmd.Controllers[i])
		if err != nil {
			return nil, err
		}

		controllers = append(controllers, found...)
	}

	return controllers, nil
}

// controllersFromOverview resolves every controller listed in one "show all"
// envelope's "System Overview" section. storcli2 omits that section when the
// host has no controllers ("Number of Controllers": 0); that is an empty
// inventory, not an error, mirroring the logical-volume and physical-drive
// getters.
func (s *StorCLI2) controllersFromOverview(controller storcli2.Controller) (
	[]*raidcontroller.RAIDController,
	error,
) {
	overview, err := utils.UnmarshalToSlice[storcli2SystemOverview](
		controller.ResponseData, "System Overview",
	)
	if err != nil {
		if errors.Is(err, utils.ErrKeyNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to unmarshal system overview")
	}

	controllers := make([]*raidcontroller.RAIDController, 0, len(overview))

	for j := range overview {
		metadata := &raidcontroller.Metadata{ID: overview[j].Ctrl}

		found, err := s.Controller(metadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get controller %d", metadata.ID)
		}

		controllers = append(controllers, found)
	}

	return controllers, nil
}

// Controller returns the RAID controller addressed by the given metadata. JBOD
// capability and state are read from dedicated commands ("show aso" and "show
// autoconfig") rather than from "show all": storcli2 dropped storcli1's
// controller-level "Support JBOD"/"Enable JBOD" fields, exposing JBOD as a
// licensed Advanced Software Option applied per drive instead. Both fields are
// informational, so firmware that does not expose this data degrades them to
// false instead of failing the inventory.
func (s *StorCLI2) Controller(metadata *raidcontroller.Metadata) (
	*raidcontroller.RAIDController,
	error,
) {
	output, err := s.runner.Run([]string{
		fmt.Sprintf(storcli2ControllerSelector, metadata.ID),
		storcli2CmdShow,
		storcli2CmdAll,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show details for controller %d", metadata.ID)
	}

	cmd, err := storcli2.Decode(output)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode controller %d output", metadata.ID)
	}

	basics, err := utils.UnmarshalToPointer[storcli2Basics](cmd.Controllers[0].ResponseData, "Basics")
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal basics")
	}

	jbodSupported, err := s.jbodSupported(metadata.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to determine JBOD support for controller %d", metadata.ID)
	}

	jbodEnabled, err := s.jbodEnabled(metadata.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to determine JBOD state for controller %d", metadata.ID)
	}

	return &raidcontroller.RAIDController{
		Metadata:        metadata,
		Name:            strings.TrimSpace(basics.ProductName),
		Serial:          strings.TrimSpace(basics.SerialNumber),
		IsJBODSupported: jbodSupported,
		IsJBODEnabled:   jbodEnabled,
	}, nil
}

// jbodSupported reports whether the controller is JBOD-capable, read from the
// licensed Advanced Software Options. storcli2 has no controller-level JBOD
// enable flag; JBOD is a licensed feature applied per drive, so capability is
// the presence of a usable (neither unsupported nor expired) "JBOD" license.
func (s *StorCLI2) jbodSupported(id int) (bool, error) {
	output, err := s.runner.Run([]string{
		fmt.Sprintf(storcli2ControllerSelector, id),
		storcli2CmdShow,
		storcli2CmdAdvancedSoftwareOptions,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to show advanced software options")
	}

	// IsJBODSupported is informational only: firmware that rejects the "show
	// aso" subcommand or omits the Advanced Software Options section (possible
	// on perccli2 / Dell PERC) must not fail the whole controller inventory.
	cmd, err := storcli2.Decode(output)
	if err != nil {
		return false, nil //nolint:nilerr // informational, see above.
	}

	options, err := utils.UnmarshalToSlice[storcli2AdvancedSoftwareOption](
		cmd.Controllers[0].ResponseData, storcli2AdvancedSoftwareOptionsKey,
	)
	if err != nil {
		return false, nil //nolint:nilerr // informational, see above.
	}

	for _, option := range options {
		if strings.EqualFold(option.SoftwareOption, storcli2JBODOption) {
			return isOptionUsable(option.TimeRemaining), nil
		}
	}

	return false, nil
}

// isOptionUsable reports whether an Advanced Software Option is usable from
// its "Time Remaining" value: an "(unsupported)"-marked option is listed but
// cannot be used by the controller, and an expired one no longer can.
func isOptionUsable(timeRemaining string) bool {
	trimmed := strings.TrimSpace(timeRemaining)

	return !strings.Contains(strings.ToLower(trimmed), storcli2OptionUnsupported) &&
		!strings.EqualFold(trimmed, storcli2OptionExpired)
}

// jbodEnabled reports whether the controller currently operates in JBOD mode.
// storcli2 has no JBOD personality and no controller-level JBOD enable, so the
// closest controller-level signal is whether new drives are auto-configured as
// JBOD.
func (s *StorCLI2) jbodEnabled(id int) (bool, error) {
	output, err := s.runner.Run([]string{
		fmt.Sprintf(storcli2ControllerSelector, id),
		storcli2CmdShow,
		storcli2CmdAutoConfig,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to show auto-configure behavior")
	}

	// IsJBODEnabled is informational only: firmware that rejects the "show
	// autoconfig" subcommand or omits the Auto-config Information section
	// (possible on perccli2 / Dell PERC) must not fail the whole controller
	// inventory.
	cmd, err := storcli2.Decode(output)
	if err != nil {
		return false, nil //nolint:nilerr // informational, see above.
	}

	entries, err := utils.UnmarshalToSlice[storcli2AutoConfigEntry](
		cmd.Controllers[0].ResponseData, storcli2AutoConfigKey,
	)
	if err != nil {
		return false, nil //nolint:nilerr // informational, see above.
	}

	for _, entry := range entries {
		if strings.EqualFold(entry.Property, storcli2PrimaryAutoConfigProp) {
			return isJBODBehavior(entry.Value), nil
		}
	}

	return false, nil
}

// isJBODBehavior reports whether an auto-configure behavior exposes drives as
// JBOD (plain or secure).
func isJBODBehavior(behavior string) bool {
	return strings.EqualFold(behavior, storcli2JBODOption) ||
		strings.EqualFold(behavior, storcli2SecureJBODBehavior)
}
