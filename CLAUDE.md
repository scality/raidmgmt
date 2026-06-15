# raidmgmt

This is a **Go library for managing RAID configurations** across hardware and
software RAID controllers. It provides a unified, well-typed interface so
consumers can perform RAID operations regardless of the underlying controller.

It follows **Hexagonal Architecture** (see `DESIGN.md`):

- **Entities** (`pkg/domain/entities/`) — typed representations of resources
  (RAIDController, PhysicalDrive, LogicalVolume).
- **Ports** (`pkg/domain/ports/`) — abstract interfaces describing operations.
- **Adapters** (`pkg/implementation/`) — concrete implementations per controller
  family (MegaRAID/storcli2, Dell PERC/perccli2, HPE Smart Array/ssacli, mdadm
  software RAID on RHEL8).

Key characteristics:

- Adapters **shell out to vendor CLI tools** and parse their JSON/text output
  (`commandrunner`, `storcli2`, `controllergetter`, etc.). Parsing fixtures live
  in `testdata/` directories.
- Error handling uses `github.com/pkg/errors` (`errors.Wrap`/`Wrapf`/`New`) and
  package-level sentinel errors (e.g. `core.ErrInvalidRAIDControllerMetadata`,
  `ports.ErrFunctionNotSupportedByImplementation`).
- Tests use `testify` with `mockery`-generated mocks; expect table-driven tests
  and `testdata/` fixtures.
- Linting is enforced via `.golangci.yaml`; CI is in `.github/workflows/`.
- No Scality internal git dependencies — all modules are public.

Implementation in GoLang should follow standards:

- Style: https://github.com/uber-go/guide/blob/master/style.md
- Follow as much as possible design patterns : https://refactoring.guru/design-patterns/go
- Avoid common mistakes https://github.com/golang/go/wiki/CommonMistakes
- Learn from Code review comments https://go.dev/wiki/CodeReviewComments

More complex projects may also:

- Follow clean architecture https://github.com/scality/golang-clean-architecture-boilerplate
- Handle errors with scality go-errors library: https://github.com/scality/go-errors
