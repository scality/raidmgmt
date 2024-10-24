package physicalvolume

import "github.com/scality/raidmgmt/domain/entities/raidcontroller"

type DiskType uint8

const (
	DiskTypeUnknown DiskType = iota
	DiskTypeHDD
	DiskTypeSSD
	DiskTypeNVMe
)

type PVStatus uint8

const (
	PVStatusUnknown PVStatus = iota
	PVStatusUsed
	PVStatusUnassigned
	PVStatusFailed
)

type Slot struct {
	Port      string // Port number of the disk (if available)
	Enclosure string // Enclosure number of the disk (if available)
	Bay       string // Bay number of the disk (if available)
}

type PhysicalVolume struct {
	Controller *raidcontroller.RAIDController // Controller of the disk
	ID         string                         // ID of the disk
	Vendor     string                         // Vendor of the disk
	Model      string                         // Model of the disk
	Serial     string                         // Serial number of the disk
	Slot       Slot                           // Slot of the disk
	Size       int                            // Size of the disk in bytes
	Type       DiskType                       // Type of the disk (e.g.: HDD, SSD)
	JBOD       bool                           // Is the disk in JBOD mode
	Status     PVStatus                       // State of the disk (e.g.: Online, Offline, Failed)
}
