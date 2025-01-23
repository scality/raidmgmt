package logicalvolume

type (
	RAIDLevel   string
	ReadPolicy  string
	WritePolicy string
	IOPolicy    string
	LVStatus    uint8
)

const (
	RAIDLevelUnknown RAIDLevel = "unknown"
	RAIDLevel0       RAIDLevel = "0"
	RAIDLevel1       RAIDLevel = "1"
	RAIDLevel10      RAIDLevel = "10"
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
