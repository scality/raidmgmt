package logicalvolumemanager

// SetRescanSCSIHosts replaces, in tests, the SCSI rescan StorCLI2 runs while
// waiting for a new volume, so that unit tests never write to the real /sys.
func (s *StorCLI2) SetRescanSCSIHosts(rescan func() error) { s.rescan = rescan }

// SetFileExists replaces, in tests, the check that a new volume's path exists
// on the host.
func (s *StorCLI2) SetFileExists(fileExists func(path string) bool) { s.fileExists = fileExists }
