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
