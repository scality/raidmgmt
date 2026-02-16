# Contributing

Thank you for your interest in contributing to RAIDmgmt! Here are some
guidelines to help you get started.

## Code of Conduct

Be respectful and constructive in all interactions. We expect contributors to
communicate professionally, provide helpful feedback, and work collaboratively
toward the shared goals of the project.

## Getting Started

1. Fork the repository and clone your fork.
2. Make sure you have Go 1.25+ and [golangci-lint](https://golangci-lint.run/)
   installed.
3. Run `make all` to verify that linting and tests pass before making changes.

## Making Changes

1. Create a branch for your work.
2. Keep commits focused -- one logical change per commit.
3. Follow the existing code style. The project uses `golangci-lint` with the
   configuration in `.golangci.yaml`.
4. Add or update tests for any changed behavior.
5. Run `make all` before submitting your changes.

## Testing

- All new code must have accompanying unit tests.
- Tests should go in `_test.go` files alongside the code they test.
- Use testdata fixtures for CLI output parsing tests (see
  `pkg/implementation/raidcontroller/megaraid/testdata/` for examples).
- Run the full test suite with `make tests` and ensure all tests pass.
- Run `make lint` to verify your code passes all linter checks.

## Documentation

- Update the README.md or DESIGN.md if your change affects the public API,
  architecture, or supported controllers.
- Document exported types and functions with Go doc comments.
- If adding a new adapter, document any limitations (unsupported operations)
  in DESIGN.md.

## Adding a New RAID Controller

The library is designed to be extended with new controllers. To add one:

1. Create a new package under `pkg/implementation/` for any low-level CLI
   interactions (command runners, getters, managers).
2. Implement the port interfaces defined in `pkg/domain/ports/raidcontroller.go`.
   If your controller does not support a given operation, return
   `ports.ErrFunctionNotSupportedByImplementation`.
3. Create a composite adapter under `pkg/implementation/raidcontroller/` that
   wires the individual implementations together (see `rhel8.go` or
   `smartarray.go` for examples).
4. Add unit tests with testdata fixtures.

## Pull Requests

1. Open a pull request against the `main` branch.
2. Describe what your change does and why.
3. Make sure CI checks pass (lint + tests).
4. At least one maintainer review is required before merging.

## Reporting Issues

Open a [GitHub issue](https://github.com/scality/raidmgmt/issues) with a clear
description of the problem or feature request. Include steps to reproduce if
applicable.
