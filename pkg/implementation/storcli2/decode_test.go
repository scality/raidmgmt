package storcli2_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scality/raidmgmt/pkg/implementation/storcli2"
	"github.com/stretchr/testify/require"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	return data
}

func TestDecodeSuccess(t *testing.T) {
	t.Parallel()

	out, err := storcli2.Decode(readFixture(t, "controllers/all.json"))
	require.NoError(t, err)
	require.Len(t, out.Controllers, 1)
	require.Equal(t, "Success", out.Controllers[0].CommandStatus.Status)
}

func TestDecodeSurfacesInJSONErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fixture string
		wantMsg string
	}{
		{name: "invalid controller (Description)", fixture: "controllers/c5_invalid.json", wantMsg: "Controller 5 not found"},
		{name: "invalid drive (Detailed Status)", fixture: "physicaldrives/e306s99_invalid.json", wantMsg: "Drive not found"},
		{name: "invalid vd (Detailed Status)", fixture: "virtualdrives/v999_invalid.json", wantMsg: "Invalid VD number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := storcli2.Decode(readFixture(t, tt.fixture))
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantMsg)
		})
	}
}

func TestDecodeNoControllers(t *testing.T) {
	t.Parallel()

	_, err := storcli2.Decode([]byte(`{"Controllers":[]}`))
	require.Error(t, err)
	require.ErrorContains(t, err, "no controllers")
}

func TestDecodeInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := storcli2.Decode([]byte(`not json`))
	require.Error(t, err)
}
