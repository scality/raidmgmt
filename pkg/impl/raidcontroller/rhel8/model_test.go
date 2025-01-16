package rhel8_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scality/raidmgmt/rhel8"
)

const (
	mdadmSingleDiskExportOutput = `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.2
MD_UUID=0030d06e:fd0fa07d:0d04737a:dc97e22c
MD_NAME=0
MD_DEVICE_dev_nvme1n1_ROLE=1
MD_DEVICE_dev_nvme1n1_DEV=/dev/nvme1n1`
)

func TestParseMDADMExportOutput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		output  []byte
		want    *rhel8.MDADMExportDetails
		wantErr bool
	}{
		{
			name:   "Valid output",
			output: []byte(mdadmSingleDiskExportOutput),
			want: &rhel8.MDADMExportDetails{
				RaidLevel:    "raid1",
				DevicesCount: 2,
				Metadata:     "1.2",
				UUID:         "0030d06e:fd0fa07d:0d04737a:dc97e22c",
				Name:         "0",
				ArraySize:    "",
				DeviceName:   "",
				Devices: map[string]rhel8.MDADMDevices{
					"dev_nvme1n1": {
						Role:  "1",
						State: "",
						Path:  "/dev/nvme1n1",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Empty output",
			output:  []byte(""),
			want:    &rhel8.MDADMExportDetails{},
			wantErr: false,
		},
		{
			name:    "nil output",
			output:  nil,
			want:    &rhel8.MDADMExportDetails{},
			wantErr: false,
		},
	}

	for _, testCase := range testCases {
		details, err := rhel8.ParseMDADMExportOutput(testCase.output)
		if (err != nil) != testCase.wantErr {
			t.Errorf("TestParseMDADMExportOutput(%s) error = %v, wantErr %v", testCase.name, err, testCase.wantErr)
			t.FailNow()
		}

		assert.Equal(t, testCase.want, details)
		assert.Nil(t, err)
	}
}
