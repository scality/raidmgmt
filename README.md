# RAIDmgmt

[![Pre-merge checks](https://github.com/scality/raidmgmt/actions/workflows/pre-merge.yaml/badge.svg)](https://github.com/scality/raidmgmt/actions/workflows/pre-merge.yaml)
![Status: Experimental](https://img.shields.io/badge/status-experimental-orange)
[![GitHub release](https://img.shields.io/github/release/scality/raidmgmt.svg)](https://github.com/scality/raidmgmt/releases/latest)

> **Warning:** This project is in an experimental phase. While it is used as
> part of a larger product, its API may change and it has not been extensively
> battle-tested in diverse environments. Use with caution in production.

RAIDmgmt is a Go library for managing RAID configurations across hardware and
software RAID controllers. It provides a unified abstraction layer so consumers
can perform RAID operations consistently, regardless of the underlying
controller.

Managing RAID across heterogeneous hardware is painful: each controller family
has its own CLI tool, output format, and quirks. RAIDmgmt solves this by
providing a single, well-typed Go interface that works identically whether
you're talking to a MegaRAID card, an HPE Smart Array, or a plain `mdadm`
setup.

## Features

- **Hardware RAID** -- MegaRAID and Dell PERC controllers, including the SAS4
  generation (MegaRAID 96xx / PERC 12 via `storcli2`/`perccli2`), and HPE
  Smart Array controllers.
- **Software RAID** -- `mdadm`-based RAID on RHEL8-family systems.
- **Unified interface** -- A single set of ports covers controller listing,
  physical drive and logical volume management, cache options, JBOD, and drive
  identification blinking.
- **Extensible** -- New controllers can be added by implementing the adapter
  interfaces.
- **Command logging** -- Every vendor CLI command the library executes can be
  logged to an `slog.Logger` by wrapping the command runner.

## Installation

```bash
go get github.com/scality/raidmgmt
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/scality/raidmgmt/pkg/core"
	"github.com/scality/raidmgmt/pkg/implementation/commandrunner"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumegetter"
	"github.com/scality/raidmgmt/pkg/implementation/logicalvolumemanager"
	"github.com/scality/raidmgmt/pkg/implementation/physicaldrivegetter"
	"github.com/scality/raidmgmt/pkg/implementation/raidcontroller"
)

func main() {
	// Create an RHEL8 software RAID controller
	runner := commandrunner.New()
	rc := raidcontroller.NewRHEL8(
		physicaldrivegetter.NewRHEL8(runner),
		logicalvolumegetter.NewMDADM(runner),
		logicalvolumemanager.NewMDADM(runner),
	)

	// Wrap it with the core service for input validation
	svc := core.NewRAIDController(rc)

	// List logical volumes (nil metadata for software RAID)
	volumes, err := svc.LogicalVolumes(nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, lv := range volumes {
		fmt.Printf("Volume %s: %s (%s)\n", lv.ID, lv.DevicePath, lv.RAIDLevel)
	}
}
```

> **Note:** The example above is for software RAID. For hardware RAID
> controllers (MegaRAID, Smart Array), see the adapter constructors in
> `pkg/implementation/raidcontroller/`. A `raidcontroller.StorCLI2`
> composition for the MegaRAID 96xx / PERC 12 generation will join them once
> the storcli2 write path lands; until then the storcli2 components are wired
> individually.

### Command logging

Everything this library does on a host goes through a
`commandrunner.CommandRunner`, so wrapping the runner in
`commandrunner.NewLogging` logs every command executed, whichever adapter
issued it:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

runner := commandrunner.NewLogging(commandrunner.NewStorCLI2(nil), logger)

drives, err := physicaldrivegetter.NewStorCLI2(runner).PhysicalDrives(metadata)
```

Each executed command produces one record with the binary invoked, its
arguments, how long it took and the outcome:

```json
{"time":"2026-09-22T10:12:31Z","level":"INFO","msg":"RAID command executed",
 "command":"/opt/MegaRAID/storcli2/storcli2","args":["/c0","show","all"],
 "duration":41000000,"output_bytes":9317}
```

Details worth knowing:

- Successful commands are logged at `slog.LevelInfo`; pass
  `commandrunner.WithLevel(slog.LevelDebug)` to lower that. Failures are always
  logged at `slog.LevelError`, with the error attached.
- The command **output is never logged** -- vendor payloads carry drive serials
  and other identifying data -- only its size.
- The arguments logged are the ones the adapter asked for: flags a runner adds
  itself (storcli2 and perccli2 append the JSON output flag) are not shown.
- The decorator is transparent, so it can wrap any runner, including the ones
  injected into a full `raidcontroller` composition. The legacy
  `megaraid.Runner` has its own equivalent, `megaraid.NewLoggingRunner`.

## Project Structure

```
pkg/
├── core/                        # Core service (validation + delegation)
├── domain/
│   ├── entities/
│   │   ├── logicalvolume/       # LogicalVolume entity, enums, methods
│   │   ├── physicaldrive/       # PhysicalDrive entity, enums, methods
│   │   └── raidcontroller/      # RAIDController entity
│   └── ports/                   # Port interfaces
├── implementation/
│   ├── blinker/                 # Drive blinking adapters
│   ├── commandrunner/           # CLI tool wrappers (storcli2, ssacli, mdadm, ...)
│   ├── controllergetter/        # Controller listing adapters
│   ├── logicalvolumegetter/     # Logical volume listing adapters
│   ├── logicalvolumemanager/    # Logical volume CRUD adapters
│   ├── physicaldrivegetter/     # Physical drive listing adapters
│   ├── raidcontroller/          # Full RAIDController adapter compositions
│   │   └── megaraid/            # MegaRAID/PERC (storcli, perccli) implementation
│   └── storcli2/                # storcli2/perccli2 shared JSON envelope + decoder
└── utils/                       # Shared utilities
```

See [DESIGN.md](DESIGN.md) for a detailed description of the architecture,
entities, ports, and adapters.

## Development

### Prerequisites

- Go 1.25+
- [golangci-lint](https://golangci-lint.run/)

### Commands

```bash
make lint    # Run linters
make tests   # Run unit tests
make all     # Run both
```

## Help

- **Issues & feature requests:** [GitHub Issues](https://github.com/scality/raidmgmt/issues)
- **Design documentation:** [DESIGN.md](DESIGN.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Maintainers

This project is maintained by the [MetalK8s](https://github.com/scality/metalk8s)
team at [Scality](https://github.com/scality).

## License

This project is licensed under the [Apache License 2.0](LICENSE).
