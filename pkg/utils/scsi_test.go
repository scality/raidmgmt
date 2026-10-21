package utils_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scality/raidmgmt/pkg/utils"
)

// fakeSCSIHosts builds a sysfs-like directory with one host per driver, each
// with its proc_name and an empty scan file.
func fakeSCSIHosts(t *testing.T, drivers ...string) string {
	t.Helper()

	root := t.TempDir()

	for i, driver := range drivers {
		host := filepath.Join(root, "host"+string(rune('0'+i)))
		require.NoError(t, os.Mkdir(host, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(host, "proc_name"), []byte(driver+"\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(host, "scan"), nil, 0o600))
	}

	return root
}

func scanned(t *testing.T, root, host string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, host, "scan"))
	require.NoError(t, err)

	return string(data)
}

// TestRescanSCSIHosts checks that only the hosts of the given driver are
// rescanned, and that finding none is an error.
func TestRescanSCSIHosts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		driver      string
		wantScanned []string
		wantErr     string
	}{
		{name: "mpi3mr", driver: "mpi3mr", wantScanned: []string{"host1"}},
		{name: "megaraid_sas", driver: "megaraid_sas", wantScanned: []string{"host0", "host3"}},
		{name: "no host of the driver", driver: "smartpqi", wantErr: "no smartpqi scsi host found to rescan"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := fakeSCSIHosts(t, "megaraid_sas", "mpi3mr", "ahci", "megaraid_sas")

			err := utils.RescanSCSIHosts(root, tt.driver)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)

			for _, host := range []string{"host0", "host1", "host2", "host3"} {
				want := ""
				if slices.Contains(tt.wantScanned, host) {
					want = "- - -"
				}

				assert.Equal(t, want, scanned(t, root, host), host)
			}
		})
	}
}

func TestRescanSCSIHostsNoSysfs(t *testing.T) {
	t.Parallel()

	err := utils.RescanSCSIHosts(filepath.Join(t.TempDir(), "missing"), "mpi3mr")
	require.ErrorContains(t, err, "failed to list scsi hosts")
}
