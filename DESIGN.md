# Design

## Scope

This library abstracts RAID configuration and provides a unified interface to
interact with various hardware RAID controllers and software RAID setups.

It does **not** contain any higher-level decision logic about _what_ RAID
operations should be performed in specific scenarios. It only provides the
building blocks to execute them.

## Architecture

The library follows [Hexagonal Architecture](https://en.wikipedia.org/wiki/Hexagonal_architecture_(software))
with three main concepts:

- **Entity** -- A representation of a physical or logical resource.
- **Port** -- An abstract interface describing operations on resources.
- **Adapter** -- A concrete implementation of a Port for a specific controller type.

### Entities

#### `RAIDController`

Represents a RAID controller card.

```go
type RAIDController struct {
    *Metadata

    Name            string // Name of the RAID controller card
    Serial          string // Serial number of the RAID controller card
    IsJBODSupported bool   // Can the controller be set in JBOD mode
    IsJBODEnabled   bool   // Is the controller currently in JBOD mode
}
```

#### `PhysicalDrive`

Represents a physical drive (disk).

```go
type DiskType uint8  // Unknown, HDD, SSD, NVMe
type PDStatus uint8  // Unknown, Used, UnassignedGood, UnassignedBad, Failed

type Slot struct {
    Port      string // Port number (if available)
    Enclosure string // Enclosure number (if available)
    Bay       string // Bay number (if available)
}

type PhysicalDrive struct {
    *Metadata

    Slot          *Slot
    Vendor        string
    Model         string
    Serial        string
    WWN           string   // World Wide Name
    Size          uint64   // Size in bytes
    Type          DiskType // HDD, SSD, NVMe
    JBOD          bool     // Is the disk in JBOD mode
    Status        PDStatus
    Reason        string   // Reason for the current status
    DevicePath    string   // e.g. /dev/sda
    PermanentPath string   // e.g. /dev/disk/by-id/...
}
```

#### `LogicalVolume`

Represents a logical volume (RAID array).

```go
type RAIDLevel uint8   // Unknown, RAID0, RAID1, RAID10
type LVStatus  uint8   // Unknown, Optimal, Degraded, Failed

type CacheOptions struct {
    ReadPolicy  ReadPolicy  // ReadAhead, NoReadAhead
    WritePolicy WritePolicy // WriteBack, WriteThrough, AlwaysWriteBack
    IOPolicy    IOPolicy    // Direct, Cached
}

type LogicalVolume struct {
    *Metadata

    PermanentPath   string
    DevicePath      string
    RAIDLevel       RAIDLevel
    PDrivesMetadata []*physicaldrive.Metadata
    CacheOptions    *CacheOptions
    Status          LVStatus
    Reason          string
    Size            uint64
}
```

### Ports

The main port is `RAIDController`, which composes several fine-grained interfaces:

| Interface | Responsibility |
|---|---|
| `ControllersGetter` | List and get RAID controllers |
| `PhysicalDrivesGetter` | List and get physical drives |
| `LogicalVolumesGetter` | List and get logical volumes |
| `LogicalVolumesManager` | Create, delete, and modify logical volumes |
| `LVCacheSetter` | Set cache options on logical volumes |
| `JBODSetter` | Enable/disable JBOD mode on physical drives |
| `Blinker` | Start/stop drive identification blinking |

Not all adapters support every operation. Unsupported operations return
`ErrFunctionNotSupportedByImplementation`.

### Adapters

#### MegaRAID / PERC

Adapter for Broadcom MegaRAID and Dell PERC controllers. Interacts with
hardware via `storcli`/`perccli`, which provides JSON output for easy parsing.

Supports all port operations: controller listing, physical drives, logical
volumes (CRUD), cache options, JBOD, and drive blinking.

> **Note:** The current MegaRAID implementation (`pkg/implementation/raidcontroller/megaraid/`)
> is a monolithic package that predates the decomposed architecture used by the
> Smart Array and RHEL8 adapters. It is planned for refactoring to follow the
> same pattern -- splitting into separate `commandrunner`, `controllergetter`,
> `physicaldrivegetter`, `logicalvolumegetter`, and `logicalvolumemanager`
> packages, and composing them in a top-level adapter.

#### Smart Array

Adapter for HPE Smart Array controllers. Interacts with hardware via `ssacli`.
Unlike `storcli`, this tool does not support JSON output, so responses are
parsed using regular expressions.

Supports all port operations.

#### Software RAID (RHEL8)

Adapter for `mdadm`-based software RAID on RHEL8-family systems.

Uses `mdadm`, `lsblk`, `udevadm`, and `smartctl` to gather information and
manage arrays.

This adapter has the following limitations compared to hardware RAID:

- No cache options.
- No drive blinking.
- No RAID 0 single-disk arrays.
- No JBOD mode.
- The `RAIDController` entity is not applicable (there is no hardware controller).
- Partitions may be used as members of an array. Each partition is represented
  as a `PhysicalDrive`, but this library does not manage partitions themselves.

## Core Service

The `core.RAIDController` wraps any adapter and adds input validation before
delegating to the underlying implementation. This is the recommended entry
point for consumers of the library.
