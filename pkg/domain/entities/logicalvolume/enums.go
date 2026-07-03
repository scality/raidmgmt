package logicalvolume

import "fmt"

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
	if r == RAIDLevelUnknown {
		return "Unknown"
	}

	return fmt.Sprintf("RAID%d", r.Level())
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
		return 255 //nolint:mnd // Default bad value
	}
}

// IsValid reports whether the RAID level is one modelled by the domain
// (RAID 0, 1 or 10). It rejects the zero value and any out-of-range level.
func (r RAIDLevel) IsValid() bool {
	switch r {
	case RAIDLevel0, RAIDLevel1, RAIDLevel10:
		return true
	case RAIDLevelUnknown:
		return false
	default:
		return false
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

// IsValid reports whether the read policy is a known settable value. The
// Unknown sentinel, the empty value and any unrecognized string are rejected.
func (r ReadPolicy) IsValid() bool {
	switch r { //nolint:exhaustive // unknown/empty handled by the default
	case ReadPolicyReadAhead, ReadPolicyNoReadAhead:
		return true
	default:
		return false
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

// IsValid reports whether the write policy is a known settable value. The
// Unknown sentinel, the empty value and any unrecognized string are rejected.
func (w WritePolicy) IsValid() bool {
	switch w { //nolint:exhaustive // unknown/empty handled by the default
	case WritePolicyWriteThrough, WritePolicyWriteBack, WritePolicyAlwaysWriteBack:
		return true
	default:
		return false
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

// IsValid reports whether the IO policy is a known settable value. The Unknown
// sentinel, the empty value and any unrecognized string are rejected, matching
// the read and write policies; the IO policy's optionality is handled by
// CacheOptions.Validate, not here.
func (i IOPolicy) IsValid() bool {
	switch i { //nolint:exhaustive // unknown/empty handled by the default
	case IOPolicyDirect, IOPolicyCached:
		return true
	default:
		return false
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
