package core_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/core"
	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/domain/ports"
)

// stubController satisfies ports.RAIDController by embedding the interface. Its
// methods are never reached in these tests: the core input guards return before
// delegating, so a call through the nil embedded interface would panic and fail
// the test loudly if a guard were missing.
type stubController struct {
	ports.RAIDController
}

func validLVMetadata() *logicalvolume.Metadata {
	return &logicalvolume.Metadata{
		CtrlMetadata: &raidcontroller.Metadata{ID: 0},
		ID:           "1",
	}
}

// TestAddPDsToLVRejectsEmptyDrives checks the core guard: growing a volume with
// no physical drive is a caller error caught before any adapter call.
func TestAddPDsToLVRejectsEmptyDrives(t *testing.T) {
	t.Parallel()

	controller := core.NewRAIDController(stubController{})

	err := controller.AddPDsToLV(validLVMetadata())
	require.ErrorIs(t, err, core.ErrNoPhysicalDrives)
}

// TestDeletePDsFromLVRejectsEmptyDrives checks the symmetric guard for removal.
func TestDeletePDsFromLVRejectsEmptyDrives(t *testing.T) {
	t.Parallel()

	controller := core.NewRAIDController(stubController{})

	err := controller.DeletePDsFromLV(validLVMetadata())
	require.ErrorIs(t, err, core.ErrNoPhysicalDrives)
}
