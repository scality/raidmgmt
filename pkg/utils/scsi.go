package utils //nolint:revive // This is a utility package.

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// SCSIHostRoot holds one sysfs directory per SCSI host adapter.
const SCSIHostRoot = "/sys/class/scsi_host"

// RescanSCSIHosts asks the kernel to probe the SCSI hosts under root driven
// by the given kernel driver (their "proc_name", such as megaraid_sas or
// mpi3mr) for new devices, such as a virtual drive just created on a RAID
// controller. Only those hosts are scanned, to avoid the multi-second
// link-down timeouts of empty SATA ports. It fails when no such host exists.
func RescanSCSIHosts(root, driver string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return errors.Wrap(err, "failed to list scsi hosts")
	}

	scanned := 0

	for _, entry := range entries {
		hostDir := filepath.Join(root, entry.Name())

		procName, err := os.ReadFile(filepath.Join(hostDir, "proc_name"))
		if err != nil || strings.TrimSpace(string(procName)) != driver {
			continue
		}

		// "- - -" means "scan every channel, target and LUN" on this host.
		if err := os.WriteFile(filepath.Join(hostDir, "scan"), []byte("- - -"), 0); err != nil {
			return errors.Wrapf(err, "failed to rescan scsi host %s", entry.Name())
		}

		scanned++
	}

	if scanned == 0 {
		return errors.Errorf("no %s scsi host found to rescan", driver)
	}

	return nil
}
