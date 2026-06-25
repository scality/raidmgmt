// Command storcli2-e2e is a manual on-hardware harness for the storcli2/perccli2
// adapter. By default it prints a read-only inventory; destructive commands
// (create, add, delete, scenario) run only when -confirm is given.
//
//	go run ./tests/integration/storcli2                       # inventory (read-only)
//	go run ./tests/integration/storcli2 scenario \
//	    -drives=252:0,252:1 -add-drives=252:2 -confirm         # full destructive cycle
//	go run ./tests/integration/storcli2 create -raid=1 -drives=252:0,252:1 -confirm
//	go run ./tests/integration/storcli2 add    -vd=0 -drives=252:2 -confirm
//	go run ./tests/integration/storcli2 delete -vd=0 -confirm
//
// Cross-compile for a target host (the binary shells out to storcli2/perccli2):
//
//	GOOS=linux GOARCH=amd64 go build -o storcli2-e2e ./tests/integration/storcli2
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/scality/raidmgmt/pkg/core"
	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	raidadapter "github.com/scality/raidmgmt/pkg/implementation/raidcontroller"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).
		With(slog.String("test_type", "e2e"), slog.String("adapter", "storcli2"))

	ctx := context.Background()

	// The command is the first non-flag argument; everything else are flags.
	// Defaulting to "inventory" keeps a bare invocation read-only.
	args := os.Args[1:]
	command := "inventory"

	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command, args = args[0], args[1:]
	}

	fs := flag.NewFlagSet(command, flag.ExitOnError)
	binary := fs.String("binary", commandrunner.StorCLI2Path, "path to the storcli2/perccli2 binary")
	controllerID := fs.Int("controller", 0, "controller index")
	raidLevel := fs.String("raid", "1", "RAID level for create/scenario: 0|1|10")
	drivesArg := fs.String("drives", "", "comma-separated EID:Slt drive ids (e.g. 252:0,252:1)")
	addDrivesArg := fs.String("add-drives", "", "comma-separated EID:Slt drive ids to expand with (scenario)")
	vdID := fs.String("vd", "", "virtual drive id for add/delete")
	confirm := fs.Bool("confirm", false, "required to run a destructive command")

	if err := fs.Parse(args); err != nil {
		logger.ErrorContext(ctx, "failed to parse arguments", slog.Any("error", err))
		os.Exit(1)
	}

	runner := commandrunner.NewStorCLI2(binary)
	controller := core.NewRAIDController(raidadapter.NewStorCLI2(runner))
	tester := NewHardwareRAIDControllerTester(*controller, *controllerID, logger)

	err := dispatch(ctx, tester, command, dispatchOptions{
		controllerID: *controllerID,
		raidLevel:    *raidLevel,
		drives:       *drivesArg,
		addDrives:    *addDrivesArg,
		vdID:         *vdID,
		confirm:      *confirm,
	})
	if err != nil {
		logger.ErrorContext(ctx, "command failed", slog.String("command", command), slog.Any("error", err))
		os.Exit(1)
	}
}

type dispatchOptions struct {
	controllerID int
	raidLevel    string
	drives       string
	addDrives    string
	vdID         string
	confirm      bool
}

// dispatch routes a command to the tester. inventory is read-only; every other
// command is destructive and requires confirmation.
func dispatch(
	ctx context.Context,
	tester *HardwareRAIDControllerTester,
	command string,
	opts dispatchOptions,
) error {
	if command == "inventory" {
		return tester.Inventory(ctx)
	}

	if !opts.confirm {
		return errors.Errorf("refusing to run destructive command %q without -confirm", command)
	}

	switch command {
	case "create":
		level, err := parseRAIDLevel(opts.raidLevel)
		if err != nil {
			return err
		}

		_, err = tester.Create(ctx, level, parseDrives(opts.drives, opts.controllerID))

		return err
	case "add":
		if opts.vdID == "" {
			return errors.New("add requires -vd")
		}

		return tester.Add(ctx, opts.vdID, parseDrives(opts.drives, opts.controllerID))
	case "delete":
		if opts.vdID == "" {
			return errors.New("delete requires -vd")
		}

		return tester.Delete(ctx, opts.vdID)
	case "scenario":
		level, err := parseRAIDLevel(opts.raidLevel)
		if err != nil {
			return err
		}

		return tester.Scenario(
			ctx,
			level,
			parseDrives(opts.drives, opts.controllerID),
			parseDrives(opts.addDrives, opts.controllerID),
		)
	default:
		return errors.Errorf("unknown command %q (want: inventory, create, add, delete, scenario)", command)
	}
}

// parseRAIDLevel maps a "0"/"1"/"10" string to a RAIDLevel.
func parseRAIDLevel(level string) (logicalvolume.RAIDLevel, error) {
	parsed := logicalvolume.RAIDLevelMap(level)
	if parsed == logicalvolume.RAIDLevelUnknown {
		return parsed, errors.Errorf("invalid RAID level %q (want: 0, 1 or 10)", level)
	}

	return parsed, nil
}

// parseDrives splits a comma-separated "EID:Slt" list into drive metadata for
// the given controller. Empty entries are skipped.
func parseDrives(arg string, controllerID int) []*physicaldrive.Metadata {
	ctrlMetadata := &raidcontroller.Metadata{ID: controllerID}

	var drives []*physicaldrive.Metadata

	for _, id := range strings.Split(arg, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		drives = append(drives, &physicaldrive.Metadata{
			CtrlMetadata: ctrlMetadata,
			ID:           id,
		})
	}

	return drives
}
