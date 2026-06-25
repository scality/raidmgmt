package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/scality/raidmgmt/pkg/core"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumemanager"
	"github.com/scality/raidmgmt/pkg/implementation/physicaldrivegetter"
	"github.com/scality/raidmgmt/pkg/implementation/raidcontroller"
)

// Remove the full array.
func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).
		With(slog.String("test_type", "integration"))

	uDevADMCommandRunner := commandrunner.NewUDevADM(nil)
	lsblkCommandRunner := commandrunner.NewLSBLK(nil)
	smartCTLCommandRunner := commandrunner.NewSmartCTL(nil)
	mdadmCommandRunner := commandrunner.NewMDADM(nil)

	physicalDriveGetter := physicaldrivegetter.NewRHEL8(
		uDevADMCommandRunner,
		lsblkCommandRunner,
		smartCTLCommandRunner,
	)

	logicalVolumeGetter := logicalvolumegetter.NewMDADM(
		mdadmCommandRunner,
	)

	logicalVolumeManager := logicalvolumemanager.NewMDADM(
		mdadmCommandRunner,
		logicalVolumeGetter,
		physicalDriveGetter,
	)

	controller := core.NewRAIDController(
		raidcontroller.NewRHEL8(
			physicalDriveGetter,
			logicalVolumeGetter,
			logicalVolumeManager,
		),
	)

	tester := NewSoftwareRAIDControllerTester(*controller, logger)

	tester.runSoftwareControllerIntegrationTestSuite(ctx, logger)
}
