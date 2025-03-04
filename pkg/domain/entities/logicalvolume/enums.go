package logicalvolume

type (
	// RAIDLevel   string.
	ReadPolicy  string
	WritePolicy string
	IOPolicy    string
	LVStatus    uint8

	RAIDLevel uint8
)

const (
	RAIDLevelUnknown RAIDLevel = iota
	RAIDLevel0
	RAIDLevel1
	RAIDLevel10
)

const (
	ReadPolicyUnknown     ReadPolicy = "unknown"
	ReadPolicyReadAhead   ReadPolicy = "ra"
	ReadPolicyNoReadAhead ReadPolicy = "nora"

	WritePolicyUnknown         WritePolicy = "unknown"
	WritePolicyWriteBack       WritePolicy = "wb"
	WritePolicyWriteThrough    WritePolicy = "wt"
	WritePolicyAlwaysWriteBack WritePolicy = "awb"

	IOPolicyUnknown IOPolicy = "unknown"
	IOPolicyDirect  IOPolicy = "direct"
	IOPolicyCached  IOPolicy = "cached"
)

const (
	LVStatusUnknown LVStatus = iota
	LVStatusOptimal
	LVStatusDegraded
	LVStatusFailed
)

func (r RAIDLevel) String() string {
	switch r { //nolint:exhaustive // Not all cases are handled
	case RAIDLevel0:
		return "RAID0"
	case RAIDLevel1:
		return "RAID1"
	case RAIDLevel10:
		return "RAID10"
	default:
		return string(RAIDLevelUnknown)
	}
}

func (r RAIDLevel) Level() uint8 {
	switch r { //nolint:exhaustive // Not all cases are handled
	case RAIDLevel0:
		return 0
	case RAIDLevel1:
		return 1
	case RAIDLevel10:
		return 10 //nolint:mnd // This is a RAID level
	default:
		return 0
	}
}

func (r ReadPolicy) String() string {
	switch r { //nolint:exhaustive // Not all cases are handled
	case ReadPolicyReadAhead:
		return "ReadAhead"
	case ReadPolicyNoReadAhead:
		return "NoReadAhead"
	default:
		return string(ReadPolicyUnknown)
	}
}

func (w WritePolicy) String() string {
	switch w { //nolint:exhaustive // Not all cases are handled
	case WritePolicyWriteBack:
		return "WriteBack"
	case WritePolicyWriteThrough:
		return "WriteThrough"
	case WritePolicyAlwaysWriteBack:
		return "AlwaysWriteBack"
	default:
		return string(WritePolicyUnknown)
	}
}

func (i IOPolicy) String() string {
	switch i { //nolint:exhaustive // Not all cases are handled
	case IOPolicyDirect:
		return "Direct"
	case IOPolicyCached:
		return "Cached"
	default:
		return string(IOPolicyUnknown)
	}
}

func (l LVStatus) String() string {
	switch l { //nolint:exhaustive // Not all cases are handled
	case LVStatusOptimal:
		return "Optimal"
	case LVStatusDegraded:
		return "Degraded"
	case LVStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}
