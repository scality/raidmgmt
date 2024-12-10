package physicaldrive

import "github.com/scality/raidmgmt/domain/entities/raidcontroller"

type (
	// Slot identifies the slot of a disk.
	Slot struct {
		Port      int // Port number of the disk (if available)
		Enclosure int // Enclosure number of the disk (if available)
		Bay       int // Bay number of the disk (if available)
	}

	// PhysicalDrive represents a physical drive.
	PhysicalDrive struct {
		Controller *raidcontroller.RAIDController // Controller of the disk
		ID         string                         // ID of the disk
		Vendor     string                         // Vendor of the disk
		Model      string                         // Model of the disk
		Serial     string                         // Serial number of the disk
		Slot       *Slot                          // Slot of the disk
		Size       uint64                         // Size of the disk in bytes
		Type       DiskType                       // Type of the disk (e.g.: HDD, SSD)
		JBOD       bool                           // Is the disk in JBOD mode
		Status     PDStatus                       // State of the disk (e.g.: Online, Offline, Failed)
	}

	// Metadata represents the metadata of a physical drive.
	Metadata struct {
		CtrlMetadata *raidcontroller.Metadata // Controller metadata of the disk
		Slot         *Slot                    // Slot of the disk
	}
)
