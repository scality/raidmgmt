package logicalvolumegetter_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scality/raidmgmt/pkg/domain/entities/logicalvolume"
	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
)

var testDataPath = "./"

func TestLogicalVolumes(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	s := &logicalvolumegetter.SSACLI{
		SSACLI: mockRunner,
	}

	mapMockingStatusNominal := map[string][]byte{
		"1":  mockOutput("logicalvolumes/show/status/1"),
		"2":  mockOutput("logicalvolumes/show/status/2"),
		"3":  mockOutput("logicalvolumes/show/status/3"),
		"4":  mockOutput("logicalvolumes/show/status/4"),
		"5":  mockOutput("logicalvolumes/show/status/5"),
		"6":  mockOutput("logicalvolumes/show/status/6"),
		"7":  mockOutput("logicalvolumes/show/status/7"),
		"8":  mockOutput("logicalvolumes/show/status/8"),
		"9":  mockOutput("logicalvolumes/show/status/9"),
		"10": mockOutput("logicalvolumes/show/status/10"),
		"11": mockOutput("logicalvolumes/show/status/11"),
	}

	tests := []struct {
		name          string
		mockingDetail []byte
		mockingStatus map[string][]byte
		id            int
		expectedError bool
	}{
		{
			name:          "nominal case",
			mockingDetail: mockOutput("logicalvolumes/show/detail/all"),
			mockingStatus: mapMockingStatusNominal,
			id:            0,
			expectedError: false,
		},
		// TODO add more test cases
	}

	for _, tt := range tests {
		mockRunner.On("Run", []string{
			"controller",
			"slot=" + strconv.Itoa(tt.id),
			"logicaldrive",
			"all",
			"show",
			"detail",
		}).Return(tt.mockingDetail, nil)

		mockRunner.On("Run", []string{
			"controller",
			"slot=" + strconv.Itoa(tt.id),
			"show",
			"config",
		}).Return(mockOutput("controller/show/config"), nil)

		metadata := &raidcontroller.Metadata{
			ID: tt.id,
		}

		logicalVolumes, err := s.LogicalVolumes(metadata)

		seen := make(map[string]bool)

		if tt.expectedError {
			assert.Error(t, err)
			assert.Nil(t, logicalVolumes)
		} else {
			assert.NoError(t, err)
			assert.NotEmpty(t, logicalVolumes)
			assert.Len(t, logicalVolumes, 11)
			assert.Equal(t, logicalvolume.RAIDLevel1, logicalVolumes[0].RAIDLevel)
			assert.Equal(t, logicalvolume.RAIDLevel0, logicalVolumes[1].RAIDLevel)

			for _, lv := range logicalVolumes {
				if seen[lv.ID] {
					t.Errorf("Duplicate logical drive: %s", lv.ID)
				} else {
					seen[lv.ID] = true
				}

				t.Logf("Logical Drive %s: %+v", lv.ID, lv)
			}
		}
	}
}

func TestLogicalVolume(t *testing.T) {
	mockRunner := new(MockCommandRunner)

	s := logicalvolumegetter.SSACLI{
		SSACLI: mockRunner,
	}

	tests := []struct {
		name          string
		mockingDetail []byte
		metadata      *logicalvolume.Metadata
		expected      *logicalvolume.LogicalVolume
		expectedError bool
	}{
		{
			name:          "nominal case",
			mockingDetail: mockOutput("logicalvolumes/show/detail/1"),
			metadata: &logicalvolume.Metadata{
				CtrlMetadata: &raidcontroller.Metadata{
					ID: 0,
				},
				ID: "1",
			},
			expected: &logicalvolume.LogicalVolume{
				Metadata: &logicalvolume.Metadata{
					CtrlMetadata: &raidcontroller.Metadata{
						ID: 0,
					},
					ID: "1",
				},
			},
			expectedError: false,
		},
		// TODO add more test cases
	}

	for _, tt := range tests {
		mockRunner.On("Run", []string{
			"controller",
			"slot=" + strconv.Itoa(tt.metadata.CtrlMetadata.ID),
			"show",
			"config",
		}).Return(mockOutput("controller/show/config"), nil)

		mockRunner.On("Run", []string{
			"controller",
			"slot=" + strconv.Itoa(tt.metadata.CtrlMetadata.ID),
			"logicaldrive",
			tt.metadata.ID,
			"show",
			"detail",
		}).Return(tt.mockingDetail, nil)

		logicalVolume, err := s.LogicalVolume(tt.metadata)

		if tt.expectedError {
			assert.Error(t, err)
			assert.Nil(t, logicalVolume)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, logicalVolume)

			assert.Equal(t, tt.metadata.ID, logicalVolume.ID)
			assert.Equal(t, tt.metadata.CtrlMetadata.ID, logicalVolume.CtrlMetadata.ID)
		}
	}
}

func mockOutput(filename string) []byte {
	output, err := os.ReadFile(testDataPath + filename + ".txt")
	if err != nil {
		panic(err)
	}

	return output
}
