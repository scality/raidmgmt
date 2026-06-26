package controllergetter_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/domain/entities/raidcontroller"
	"github.com/scality/raidmgmt/pkg/implementation/controllergetter"
)

// storcli2Fixture reads a storcli2 JSON fixture from the package testdata.
func storcli2Fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile("testdata/storcli2/" + name)
	require.NoError(t, err)

	return data
}

// mockController registers the three Run calls Controller() issues for a
// controller: "show all", "show aso" and "show autoconfig".
func mockController(mockRunner *MockCommandRunner, id int, showAll, aso, autoConfig []byte) {
	selector := fmt.Sprintf("/c%d", id)
	mockRunner.On("Run", []string{selector, "show", "all"}).Return(showAll, nil)
	mockRunner.On("Run", []string{selector, "show", "aso"}).Return(aso, nil)
	mockRunner.On("Run", []string{selector, "show", "autoconfig"}).Return(autoConfig, nil)
}

func TestStorCLI2Controllers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		showAll     []byte
		mockDetail  bool
		expectedLen int
		expectError bool
	}{
		{
			name:        "nominal case",
			showAll:     storcli2Fixture(t, "all.json"),
			mockDetail:  true,
			expectedLen: 1,
		},
		{
			name:        "no controllers",
			showAll:     []byte(`{"Controllers":[]}`),
			expectError: true,
		},
		{
			name:        "no managed controllers",
			showAll:     storcli2Fixture(t, "all_no_controllers.json"),
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockRunner.On("Run", []string{"show", "all"}).Return(tt.showAll, nil)

			if tt.mockDetail {
				mockController(mockRunner, 0,
					storcli2Fixture(t, "c0.json"),
					storcli2Fixture(t, "c0_aso.json"),
					storcli2Fixture(t, "c0_autoconfig.json"),
				)
			}

			s := controllergetter.NewStorCLI2(mockRunner)

			controllers, err := s.Controllers()
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, controllers)

				return
			}

			require.NoError(t, err)
			require.Len(t, controllers, tt.expectedLen)

			if tt.expectedLen > 0 {
				assert.Equal(t, 0, controllers[0].ID)
			}
		})
	}
}

func TestStorCLI2Controller(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	mockController(mockRunner, 0,
		storcli2Fixture(t, "c0.json"),
		storcli2Fixture(t, "c0_aso.json"),
		storcli2Fixture(t, "c0_autoconfig.json"),
	)

	s := controllergetter.NewStorCLI2(mockRunner)

	controller, err := s.Controller(&raidcontroller.Metadata{ID: 0})
	require.NoError(t, err)
	require.NotNil(t, controller)
	assert.Equal(t, 0, controller.ID)
	assert.Equal(t, "MegaRAID 9660-16i Tri-Mode Storage Adapter", controller.Name)
	assert.Equal(t, "SPE4912106", controller.Serial)
	// JBOD is a factory-installed (Unlimited) license, and the primary
	// auto-configure behavior is UGood, so JBOD is supported but not active.
	assert.True(t, controller.IsJBODSupported)
	assert.False(t, controller.IsJBODEnabled)
}

func TestStorCLI2ControllerNotFound(t *testing.T) {
	t.Parallel()

	mockRunner := new(MockCommandRunner)
	// "/c5 show all" reports a failure payload that Decode rejects, so the getter
	// errors before reaching the aso / autoconfig calls.
	mockRunner.On("Run", []string{"/c5", "show", "all"}).
		Return(storcli2Fixture(t, "c5_invalid.json"), nil)

	s := controllergetter.NewStorCLI2(mockRunner)

	controller, err := s.Controller(&raidcontroller.Metadata{ID: 5})
	require.Error(t, err)
	assert.Nil(t, controller)
}

// TestStorCLI2ControllerJBOD exercises the JBOD capability / state mapping over
// synthetic "show aso" and "show autoconfig" payloads, keeping the real "show
// all" fixture for the controller identity.
func TestStorCLI2ControllerJBOD(t *testing.T) {
	t.Parallel()

	asoPayload := func(option, timeRemaining string) []byte {
		return fmt.Appendf(nil,
			`{"Controllers":[{"Command Status":{"Status":"Success"},`+
				`"Response Data":{"Advanced Software options":`+
				`[{"Software option":%q,"Time Remaining":%q}]}}]}`,
			option, timeRemaining)
	}

	autoConfigPayload := func(value string) []byte {
		return fmt.Appendf(nil,
			`{"Controllers":[{"Command Status":{"Status":"Success"},`+
				`"Response Data":{"Auto-config Information":`+
				`[{"Auto-config property":"Primary Auto-configure behavior","Value":%q}]}}]}`,
			value)
	}

	tests := []struct {
		name            string
		aso             []byte
		autoConfig      []byte
		expectedJBODSup bool
		expectedJBODEn  bool
	}{
		{
			name:            "licensed and inactive",
			aso:             asoPayload("JBOD", "Unlimited"),
			autoConfig:      autoConfigPayload("UGood"),
			expectedJBODSup: true,
			expectedJBODEn:  false,
		},
		{
			name:            "expired license, JBOD auto-config",
			aso:             asoPayload("JBOD", "Expired"),
			autoConfig:      autoConfigPayload("JBOD"),
			expectedJBODSup: false,
			expectedJBODEn:  true,
		},
		{
			name:            "licensed and secure JBOD active",
			aso:             asoPayload("JBOD", "Unlimited"),
			autoConfig:      autoConfigPayload("SecureJBOD"),
			expectedJBODSup: true,
			expectedJBODEn:  true,
		},
		{
			name:            "no JBOD license",
			aso:             asoPayload("RAID 0", "Unlimited"),
			autoConfig:      autoConfigPayload("UGood"),
			expectedJBODSup: false,
			expectedJBODEn:  false,
		},
		{
			// Per the User Guide, "Time Remaining" may be suffixed with
			// "(unsupported)" when the controller cannot use the option.
			name:            "JBOD license listed but unsupported",
			aso:             asoPayload("JBOD", "Unlimited (unsupported)"),
			autoConfig:      autoConfigPayload("UGood"),
			expectedJBODSup: false,
			expectedJBODEn:  false,
		},
		{
			name:            "JBOD trial license counting down",
			aso:             asoPayload("JBOD", "30 Days 4 Hours"),
			autoConfig:      autoConfigPayload("UGood"),
			expectedJBODSup: true,
			expectedJBODEn:  false,
		},
		// Both fields are informational: firmware that rejects the probe
		// subcommands or omits their sections (possible on perccli2 / Dell
		// PERC) degrades them to false instead of failing the inventory.
		{
			name: "probe subcommands rejected by firmware",
			aso: []byte(`{"Controllers":[{"Command Status":` +
				`{"Status":"Failure","Description":"Un-supported command"}}]}`),
			autoConfig: []byte(`{"Controllers":[{"Command Status":` +
				`{"Status":"Failure","Description":"Un-supported command"}}]}`),
			expectedJBODSup: false,
			expectedJBODEn:  false,
		},
		{
			name: "probe sections missing from response data",
			aso: []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
				`"Response Data":{}}]}`),
			autoConfig: []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
				`"Response Data":{}}]}`),
			expectedJBODSup: false,
			expectedJBODEn:  false,
		},
		{
			name: "primary auto-configure property absent",
			aso:  asoPayload("JBOD", "Unlimited"),
			autoConfig: []byte(`{"Controllers":[{"Command Status":{"Status":"Success"},` +
				`"Response Data":{"Auto-config Information":[]}}]}`),
			expectedJBODSup: true,
			expectedJBODEn:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRunner := new(MockCommandRunner)
			mockController(mockRunner, 0, storcli2Fixture(t, "c0.json"), tt.aso, tt.autoConfig)

			s := controllergetter.NewStorCLI2(mockRunner)

			controller, err := s.Controller(&raidcontroller.Metadata{ID: 0})
			require.NoError(t, err)
			require.NotNil(t, controller)
			assert.Equal(t, tt.expectedJBODSup, controller.IsJBODSupported)
			assert.Equal(t, tt.expectedJBODEn, controller.IsJBODEnabled)
		})
	}
}
