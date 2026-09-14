package logicalvolumemanager_test

import (
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/physicaldrive"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumemanager"
)

// ssacliCreateRequest builds a create request on controller 0 with as many
// available drives as the RAID level needs.
func ssacliCreateRequest(
	level logicalvolume.RAIDLevel,
	drives int,
) *logicalvolume.Request {
	ctrl := &raidcontroller.Metadata{ID: 0}

	pdsMetadata := make([]*physicaldrive.Metadata, 0, drives)
	for bay := 1; bay <= drives; bay++ {
		pdsMetadata = append(pdsMetadata, &physicaldrive.Metadata{
			CtrlMetadata: ctrl,
			ID:           "1I:1:" + strconv.Itoa(bay),
		})
	}

	return &logicalvolume.Request{
		CtrlMetadata:    ctrl,
		RAIDLevel:       level,
		PDrivesMetadata: pdsMetadata,
	}
}

// ssacli takes the RAID level as a number, and 1+0 for RAID 10. Rendering the
// enum instead of its level sent the code point of its ordinal, so ssacli got
// "raid=\x01" for a RAID 0 and exited 1 without creating anything.
func TestSSACLICreateLVBuildsTheCommand(t *testing.T) {
	tests := []struct {
		name     string
		level    logicalvolume.RAIDLevel
		drives   int
		expected []string
	}{
		{
			name:   "RAID 0 on a single drive",
			level:  logicalvolume.RAIDLevel0,
			drives: 1,
			expected: []string{
				"controller", "slot=0", "create", "type=ld",
				"drives=1I:1:1", "raid=0", "forced",
			},
		},
		{
			name:   "RAID 1 on a mirror",
			level:  logicalvolume.RAIDLevel1,
			drives: 2,
			expected: []string{
				"controller", "slot=0", "create", "type=ld",
				"drives=1I:1:1,1I:1:2", "raid=1", "forced",
			},
		},
		{
			name:   "RAID 10 spells 1+0",
			level:  logicalvolume.RAIDLevel10,
			drives: 4,
			expected: []string{
				"controller", "slot=0", "create", "type=ld",
				"drives=1I:1:1,1I:1:2,1I:1:3,1I:1:4", "raid=1+0", "forced",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := ssacliCreateRequest(tt.level, tt.drives)

			mockRunner := new(MockCommandRunner)
			mockPDGetter := new(MockPhysicalDrivesGetter)

			for _, pdMetadata := range request.PDrivesMetadata {
				mockPDGetter.On("PhysicalDrive", pdMetadata).Return(&physicaldrive.PhysicalDrive{
					Metadata: pdMetadata,
					Size:     1000,
					Status:   physicaldrive.PDStatusUnassignedGood,
				}, nil)
			}

			// The command is the assertion, so the run stops right after it.
			var got []string

			mockRunner.On("Run", mock.AnythingOfType("[]string")).
				Run(func(args mock.Arguments) {
					got = args.Get(0).([]string)
				}).
				Return([]byte(nil), errors.New("stop after the create command")).Once()

			manager := &logicalvolumemanager.SSACLI{
				PhysicalDrivesGetter: mockPDGetter,
				SSACLI:               mockRunner,
			}

			_, err := manager.CreateLV(request)

			require.Error(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}
