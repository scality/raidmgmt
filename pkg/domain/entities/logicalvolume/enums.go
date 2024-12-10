package logicalvolume

type (
	RAIDLevel   uint8
	ReadPolicy  uint8
	WritePolicy uint8
	IOPolicy    uint8
	LVStatus    uint8
)

const (
	RAIDLevelUnknown RAIDLevel = iota
	RAIDLevel0
	RAIDLevel1
	RAIDLevel10
)

const (
	ReadPolicyUnknown ReadPolicy = iota
	ReadPolicyReadAhead
	ReadPolicyNoReadAhead

	ReadAhead   string = "ra"
	NoReadAhead string = "nra"
)

const (
	WritePolicyUnknown WritePolicy = iota
	WritePolicyWriteBack
	WritePolicyWriteThrough
	WritePolicyAlwaysWriteBack

	WriteBack       string = "wb"
	WriteThrough    string = "wt"
	AlwaysWriteBack string = "awb"
)

const (
	IOPolicyUnknown IOPolicy = iota
	IOPolicyDirect
	IOPolicyCached

	Direct string = "direct"
	Cached string = "cached"
)

const (
	LVStatusUnknown LVStatus = iota
	LVStatusOptimal
	LVStatusOffline
	LVStatusPartiallyDegraded
	LVStatusDegraded
	LVStatusFailed
)

func (r ReadPolicy) String() string {
	switch r {
	case ReadPolicyReadAhead:
		return ReadAhead
	case ReadPolicyNoReadAhead:
		return NoReadAhead
	default:
		return ""
	}
}

func (w WritePolicy) String() string {
	switch w {
	case WritePolicyWriteBack:
		return WriteBack
	case WritePolicyWriteThrough:
		return WriteThrough
	case WritePolicyAlwaysWriteBack:
		return AlwaysWriteBack
	default:
		return ""
	}
}

func (i IOPolicy) String() string {
	switch i {
	case IOPolicyDirect:
		return Direct
	case IOPolicyCached:
		return Cached
	default:
		return ""
	}
}
