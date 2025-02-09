package rhel8

import (
	"github.com/scality/raidmgmt/pkg/domain/ports"
)

type (
	RHEL8 struct {
		ports.LogicalVolumesGetter
		ports.LogicalVolumesManager
	}
)

func NewRHEL8(
	logicalVolumesGetter ports.LogicalVolumesGetter,
	logicalVolumesManager ports.LogicalVolumesManager,
) *RHEL8 {
	return &RHEL8{
		LogicalVolumesGetter:  logicalVolumesGetter,
		LogicalVolumesManager: logicalVolumesManager,
	}
}
