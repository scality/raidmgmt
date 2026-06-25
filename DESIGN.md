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

The same applies at the value level: an enum (e.g. `ReadPolicy`, `WritePolicy`)
is the **union** of what all controllers support, so each adapter handles a
**subset**. Adapters translate domain enum values to vendor CLI tokens through a
small mapping function that returns an "unsettable" signal (e.g. `(token string,
ok bool)`) for values it cannot express, failing closed via an exhaustive
`switch` default. This is distinct from entity `Validate()` (which guards
domain-level coherence): the mapping guards the adapter boundary, keeps vendor
vocabulary out of the domain, and rejects values a controller cannot realize —
including ones added later for other controllers.

Relatedly, setters read current state before writing and emit only the changed
flags. This is not for idempotency but to minimize real mutations and to skip
fields the (lossy) getter reports as `Unknown` when the caller did not change
them — avoiding a spurious "unsettable" rejection on an untouched field.

### Adapters

#### MegaRAID / PERC (storcli, perccli)

Adapter for Broadcom MegaRAID and Dell PERC controllers up to the SAS3.5
generation (MegaRAID 94xx/95xx). Interacts with hardware via
`storcli`/`perccli`, which provides JSON output for easy parsing.

Supports all port operations: controller listing, physical drives, logical
volumes (CRUD), cache options, JBOD, and drive blinking.

> **Note:** The current MegaRAID implementation (`pkg/implementation/raidcontroller/megaraid/`)
> is a monolithic package that predates the decomposed architecture used by the
> Smart Array and RHEL8 adapters. It is planned for refactoring to follow the
> same pattern -- splitting into separate `commandrunner`, `controllergetter`,
> `physicaldrivegetter`, `logicalvolumegetter`, and `logicalvolumemanager`
> packages, and composing them in a top-level adapter.

#### MegaRAID 96xx / PERC 12 (storcli2, perccli2)

Adapter for the SAS4 ("MegaRAID 8" / tri-mode) generation: Broadcom MegaRAID
96xx and Dell PERC 12 controllers, driven through `storcli2`/`perccli2`.

It is built directly on the decomposed pattern: one component package per
port (`controllergetter`, `physicaldrivegetter`, `logicalvolumegetter`, ...),
each named `StorCLI2` and taking a `commandrunner.CommandRunner`. Both
binaries emit the same JSON schema, so a single set of components serves
both; the concrete runner (`commandrunner.StorCLI2` or
`commandrunner.PercCLI2`) is injected at construction time.

Design notes, verified against the StorCLI2 User Guide and a live MegaRAID
9660-16i:

- All outputs share a JSON envelope decoded by `pkg/implementation/storcli2.Decode`.
  The process exit code is **not** a reliable success signal (some failures
  exit 0, others exit non-zero while still writing the JSON payload), so
  errors are detected from each controller's `Command Status` instead, and
  the runners return the payload as-is on a non-zero exit.
- Showing a nonexistent object may report success with an absent section
  (treated as an empty inventory) or an explicit failure payload: the User
  Guide documents the former, while the binaries tested so far do the
  latter. Both are handled.
- storcli2 has no controller-level JBOD enable and no JBOD personality.
  `IsJBODSupported` is derived from a usable `JBOD` Advanced Software Option
  (`show aso`) and `IsJBODEnabled` from the primary auto-configure behavior
  (`show autoconfig`); JBOD itself is a per-drive **state**, orthogonal to
  the per-drive **status** (each `set` command changes only one of the two).
- Drive states come in suffixed variants (`Shld`, `Sntz`, `Dgrd`), so they
  are matched by family; a `Failed`/`Offline`/`Missing` status takes
  precedence over any state.
- storcli2 dropped some storcli operations: there is no RAID-level migration,
  so a volume's member set can no longer be *shrunk* (storcli's `start migrate
  option=remove`); growing a volume moves to `/cx/vx expand drives=`. This is
  about reshaping the array's member **count** only -- replacing a *failed*
  drive (rebuild onto a hot spare or a hot-swapped replacement, copyback) keeps
  the member count unchanged, is a separate command family that storcli2 still
  exposes, and is out of scope for raidmgmt regardless of controller. There is
  also no IO policy (`Cached`/`Direct`) cache option -- the IO policy of parsed
  volumes is always `Unknown`.

The read path (controller, physical drive and logical volume getters), the
cache and JBOD setters (`lvcachesetter`, `jbodsetter`), the shared
envelope/decoder and both command runners are implemented -- each in its own
package named after its port. The remaining ports follow the same component
pattern; the table below maps each port operation to its storcli2 command
(verified against the StorCLI2 User
Guide, the official storcli-to-storcli2 command map of the MegaRAID 8
software guide, and the binary's own help -- the grammar differs from
storcli in several places).

| Port operation | storcli2 command | Notes |
|---|---|---|
| `CreateLV` | `/cx add vd r<level> [Size=<sz>] drives=e:s,... [WT\|WB\|AWB] [nora\|ra]` | Cache policies are bare tokens at creation time; storcli's `type=` / `wrcache=` / `rdpolicy=` forms are gone. |
| `DeleteLV` | `/cx/vx delete [discardcache] [force]` | A nonexistent VD yields a failure payload surfaced by `Decode`. |
| `AddPDsToLV` | `/cx/vx expand drives=e:s,...` | Online capacity expansion. Documented and present in the binary help, but not exercised on hardware yet; progress is visible through `/cx/vx show expansion` and `show ocedriveinfo`. |
| `DeletePDsFromLV` | -- | Not supported: removing a member (shrinking a volume) used storcli's `start migrate option=remove`, which the storcli2 command map drops with no replacement. This is array *reshaping*, not failed-drive replacement (rebuild / hot-spare / copyback), which keeps the member count and is a separate, still-supported command family out of scope here. Returns `ErrFunctionNotSupportedByImplementation`. |
| `SetLVCacheOptions` | `/cx/vx set rdcache=RA\|NoRA` and `/cx/vx set wrcache=WT\|WB\|AWB` | Two separate commands: storcli's combined syntax is rejected. The IO policy cannot be set (see above); beware that `CacheOptions.Validate()` rejects an unknown IO policy, so a request cannot be round-tripped from getter output as-is. |
| `EnableJBOD` | `/cx/ex/sx set jbod [force]` | Converts the drive **state**; the drive status is unchanged. |
| `DisableJBOD` | `/cx/ex/sx set uconf [force]` | storcli's `delete jbod` no longer parses; `set good` would only change the status. |
| `StartBlink` / `StopBlink` | `/cx/ex/sx start locate` / `stop locate` | Same grammar as storcli. |

The top-level `raidcontroller.StorCLI2` composition embeds the storcli2
components across the full `ports.RAIDController` surface (no stubs: every
operation except `DeletePDsFromLV` is supported, and that unsupported error is
returned by the logical-volume manager itself, not by a composition-level
override). A single composition serves both binaries since only the injected
runner differs.

> **Note:** Part of the pre-staged write-path fixtures under
> `pkg/implementation/lvcachesetter/testdata/storcli2/cacheoptions/` and
> `pkg/implementation/jbodsetter/testdata/storcli2/jbod/` were
> captured with the storcli grammar and are plain-text syntax errors; they
> must be regenerated with the commands above using
> `tests/testdata-tools/collect_storcli2_testdata.sh` (DESTRUCTIVE mode).

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
