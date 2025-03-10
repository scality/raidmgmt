package main

import (
	"os"

	"github.com/rs/zerolog"

	"github.com/scality/raidmgmt/pkg/core"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumemanager"
	"github.com/scality/raidmgmt/pkg/implementation/physicaldrivegetter"
	"github.com/scality/raidmgmt/pkg/implementation/raidcontroller"
)

// Remove the full array.
func main() {
	logger := zerolog.New(os.Stdout).With().Str("test_type", "integration").Logger()

	uDevADMCommandRunner := commandrunner.NewUDevADM()
	lsblkCommandRunner := commandrunner.NewLSBLK()
	smartCTLCommandRunner := commandrunner.NewSmartCTL()
	mdadmCommandRunner := commandrunner.NewMDADM()

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

	tester := NewSoftwareRAIDControllerTester(*controller, &logger)

	tester.runSoftwareControllerIntegrationTestSuite(&logger)
}
