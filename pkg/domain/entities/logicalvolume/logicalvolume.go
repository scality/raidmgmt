package logicalvolume

import (
	"github.com/scality/raidmgmt/domain/entities/physicalvolume"
	"github.com/scality/raidmgmt/domain/entities/raidcontroller"
)

type RAIDLevel uint8

const (
	RAIDLevelUnknown RAIDLevel = iota
	RAIDLevel0
	RAIDLevel1
	RAIDLevel10
)

type WritePolicy uint8

const (
	WritePolicyUnknown WritePolicy = iota
	WritePolicyWriteBack
	WritePolicyWriteThrough
	WritePolicyAlwaysWriteBack
)

type ReadPolicy uint8

const (
	ReadPolicyUnknown ReadPolicy = iota
	ReadPolicyReadAhead
	ReadPolicyNoReadAhead
)

type IOPolicy uint8

const (
	IOPolicyUnknown IOPolicy = iota
	IOPolicyDirect
	IOPolicyCached
)

type CacheOptions struct {
	WritePolicy WritePolicy // Write policy of the cache (e.g.: WriteBack, WriteThrough)
	ReadPolicy  ReadPolicy  // Read policy of the cache (e.g.: ReadAhead, NoReadAhead)
	IOPolicy    IOPolicy    // IO policy of the cache (e.g.: Direct, Cached)
}

type LVStatus uint8

const (
	LVStatusUnknown LVStatus = iota
	LVStatusOptimal
	LVStatusDegraded
	LVStatusFailed
)

type LogicalVolume struct {
	Controller      *raidcontroller.RAIDController   // Controller of the array
	ID              string                           // ID of the array
	DevicePath      string                           // Device path of the array (e.g.: /dev/sda)
	RAIDLevel       RAIDLevel                        // RAID level of the array (e.g.: RAID 0, RAID 1, RAID 10, ...)
	PhysicalVolumes []*physicalvolume.PhysicalVolume // Physical volumes composing the array
	CacheOptions    *CacheOptions                    // Cache options of the array
	Status          LVStatus                         // State of the array (e.g.: Online, Offline, Degraded)
}
