package raidcontroller

import (
	"github.com/scality/raidmgmt/pkg/domain/ports"
	"github.com/scality/raidmgmt/pkg/implementation/blinker"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/controllergetter"
	"github.com/scality/raidmgmt/pkg/implementation/jbodsetter"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumemanager"
	"github.com/scality/raidmgmt/pkg/implementation/lvcachesetter"
	"github.com/scality/raidmgmt/pkg/implementation/physicaldrivegetter"
)

// StorCLI2 is the top-level RAID controller adapter for MegaRAID/PERC
// controllers driven by storcli2/perccli2. It composes the storcli2 components
// into the full ports.RAIDController surface. A single composition serves both
// binaries since only the injected command runner differs. Unlike SmartArray,
// storcli2 supports every operation, so there are no
// ErrFunctionNotSupportedByImplementation stubs.
type StorCLI2 struct {
	ports.ControllersGetter
	ports.PhysicalDrivesGetter
	ports.LogicalVolumesGetter
	ports.LogicalVolumesManager
	ports.LVCacheSetter
	ports.JBODSetter
	ports.Blinker
}

var _ ports.RAIDController = &StorCLI2{}

// NewStorCLI2 wires the storcli2 components on top of the given command runner.
// One component package implements each port; the logical-volume manager and
// the cache setter both read current state through the logical-volume getter,
// which is therefore shared rather than constructed twice.
func NewStorCLI2(runner commandrunner.CommandRunner) *StorCLI2 {
	physicalDrivesGetter := physicaldrivegetter.NewStorCLI2(runner)
	logicalVolumesGetter := logicalvolumegetter.NewStorCLI2(runner)

	return &StorCLI2{
		ControllersGetter:    controllergetter.NewStorCLI2(runner),
		PhysicalDrivesGetter: physicalDrivesGetter,
		LogicalVolumesGetter: logicalVolumesGetter,
		LogicalVolumesManager: logicalvolumemanager.NewStorCLI2(
			runner, physicalDrivesGetter, logicalVolumesGetter,
		),
		LVCacheSetter: lvcachesetter.NewStorCLI2(runner, logicalVolumesGetter),
		JBODSetter:    jbodsetter.NewStorCLI2(runner),
		Blinker:       blinker.NewStorCLI2(runner),
	}
}
