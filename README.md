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

- **Hardware RAID** -- MegaRAID, Dell PERC, and HPE Smart Array controllers.
- **Software RAID** -- `mdadm`-based RAID on RHEL8-family systems.
- **Unified interface** -- A single set of ports covers controller listing,
  physical drive and logical volume management, cache options, JBOD, and drive
  identification blinking.
- **Extensible** -- New controllers can be added by implementing the adapter
  interfaces.

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
> `pkg/implementation/raidcontroller/`.

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
│   ├── commandrunner/           # CLI tool wrappers (storcli, ssacli, mdadm, ...)
│   ├── controllergetter/        # Controller listing adapters
│   ├── logicalvolumegetter/     # Logical volume listing adapters
│   ├── logicalvolumemanager/    # Logical volume CRUD adapters
│   ├── physicaldrivegetter/     # Physical drive listing adapters
│   └── raidcontroller/          # Full RAIDController adapter compositions
│       └── megaraid/            # MegaRAID/PERC-specific implementation
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
