package physicaldrive

type (
	DiskType uint8
	PDStatus uint8
)

const (
	DiskTypeUnknown DiskType = iota
	DiskTypeHDD
	DiskTypeSSD
	DiskTypeNVMe
)

const (
	PDStatusUnknown PDStatus = iota
	PDStatusUsed
	PDStatusUnassignedGood
	PDStatusUnassignedBad
	PDStatusFailed
)

func (d DiskType) String() string {
	switch d { //nolint:exhaustive // Not all cases are handled
	case DiskTypeHDD:
		return "HDD"
	case DiskTypeSSD:
		return "SSD"
	case DiskTypeNVMe:
		return "NVMe"
	default:
		return "Unknown"
	}
}

func (s PDStatus) String() string {
	switch s { //nolint:exhaustive // Not all cases are handled
	case PDStatusUsed:
		return "Used"
	case PDStatusUnassignedGood:
		return "UnassignedGood"
	case PDStatusUnassignedBad:
		return "UnassignedBad"
	case PDStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}
