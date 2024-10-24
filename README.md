# RAIDmgmt

RAIDmgmt is a Go library for managing RAID configurations on various RAID controllers. 
It provides an abstraction layer for interacting with different RAID controllers and software RAID setups, allowing users to perform RAID operations in a consistent way across different environments.

This library is based upon [this design on citadel](https://citadel.scality.net/design/platform/library/raid-management/). It follows the [Hexagonal architecture](https://en.wikipedia.org/wiki/Hexagonal_architecture_(software)) to easily integrate new RAID controllers.

## Features

- Supports MegaRAID, PERC, Smart Array, and software RAID on RHEL8 based OSes.
- Provides abstraction over RAID configuration and operations.
- Extensible with support for additional RAID controllers via adapters.

## Usage 

### Basic Example

TODO

## Architecture

This library is designed with Hexagonal Architecture, which separates the core business logic (domain layer) from the system-specific implementations (adapters).

- Core Domain: Contains core RAID logic and domain models.
- Ports: Defines the interfaces for interacting with RAID controllers.
- Adapters: Implementations of RAID interactions for specific controllers (e.g., MegaRAID, Smart Array).

The repo is structured as it follows:

├── go.work
├── Makefile
├── pkg
│   ├── core
│   │   ├── go.mod
│   │   └── raidcontrollerservice.go
│   ├── domain
│   │   ├── entities
│   │   │   ├── logicalvolume
│   │   │   │   └── logicalvolume.go
│   │   │   ├── physicalvolume
│   │   │   │   └── physicalvolume.go
│   │   │   └── raidcontroller
│   │   │       └── raidcontroller.go
│   │   ├── go.mod
│   │   └── ports
│   │       └── raidcontrollerservice
│   │           └── raidcontrollerservice.go
│   └── impl
│       ├── megaraid
│       │   ├── go.mod
│       │   └── storccli.go
│       ├── perc
│       │   ├── go.mod
│       │   └── perccli.go
│       ├── rhel8
│       │   ├── go.mod
│       │   └── mdadm.go
│       └── smartarray
│           ├── go.mod
│           └── ssacli.go
└── README.md

**Core** (`pkg/core/`): This is where the core business logic resides, such as the orchestration of RAID management tasks through the `raidcontrollerservice.go`. This part of the code should be agnostic to the specific RAID controller being used.

**Domain** (`pkg/domain/`): This contains both the **ports** (interfaces that define how the core interacts with the outside world) and **models** (representations of domain entities like `LogicalVolume`, `PhysicalVolume`, and `RaidController`). 

**Impl** (`pkg/impl/`): This holds the **adapters** for the different RAID controllers (MegaRAID, PERC, SmartArray, etc.). These adapters implement the ports defined in `pkg/domain/` and are responsible for the actual interaction with the respective CLI tools or system-level commands for each RAID controller.