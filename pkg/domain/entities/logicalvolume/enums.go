package logicalvolume

type (
	RAIDLevel   uint8
	ReadPolicy  string
	WritePolicy string
	IOPolicy    string
	LVStatus    uint8
)

const (
	RAIDLevelUnknown RAIDLevel = iota
	RAIDLevel0
	RAIDLevel1
	RAIDLevel10
)

const (
	ReadPolicyUnknown     ReadPolicy = "unknwon"
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
	LVStatusOffline
	LVStatusPartiallyDegraded
	LVStatusDegraded
	LVStatusFailed
)
