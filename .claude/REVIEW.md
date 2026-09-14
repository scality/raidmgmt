# Review criteria

Read by the `/review-pr` skill (Scality agent hub) and by anyone reviewing by hand.
Flag problems only — see "What not to flag" at the end.

## What this repo is

`raidmgmt` is a Go library (`github.com/scality/raidmgmt`) that abstracts hardware
and software RAID controllers behind one API, following hexagonal architecture:
domain entities and ports in the core, vendor CLIs wrapped in adapters under
`pkg/implementation/`. Architecture and boundaries: `DESIGN.md`.

It is imported by other projects, so exported entities, port signatures and
sentinel errors are a public contract.

## Criteria

| Area | What to check |
|------|---------------|
| Error handling | Wrap errors with `github.com/pkg/errors` (`errors.Wrap`/`Wrapf`) to preserve stack context — matching the existing code; do not silently drop errors or return bare `err` where a wrap adds useful context. Reuse package-level sentinels (`core.Err*`, `ports.ErrFunctionNotSupportedByImplementation`) instead of inventing duplicate error strings. |
| Hexagonal boundaries | Respect the entity/port/adapter separation (`DESIGN.md`). Domain entities and ports must not import adapter or vendor-CLI packages. New controller support belongs in `pkg/implementation/` behind the existing port interfaces. |
| Port interface compliance | When an adapter changes, verify it still satisfies its port interface and that unsupported operations return `ErrFunctionNotSupportedByImplementation` rather than `nil` or a panic. Interface signature changes ripple to every adapter — check they were all updated. |
| CLI output parsing | Adapters parse external tool output (storcli2, perccli2, ssacli, mdadm, lsblk, smartctl, udevadm). Check for unguarded map/slice/pointer access, missing fields, nil dereferences, and tool-version/format drift. New or changed parsing must have a corresponding `testdata/` fixture. |
| Command execution | Validate how external commands are built and run (`commandrunner`). Watch for unsanitized inputs interpolated into command arguments, missing non-zero exit-code handling, and ignored stderr. |
| Units & conversions | Sizes are in bytes (`uint64`); enums (DiskType, PDStatus, RAID levels) map vendor-specific strings. Check for integer overflow/truncation, wrong unit conversions, and unhandled enum values that should map to an `Unknown` sentinel. |
| Tests | New behavior needs table-driven tests with `testify`. If a mocked interface changed, mockery-generated mocks must be regenerated and committed. Confirm `testdata/` fixtures match the code paths they exercise. |
| Concurrency | If goroutines are introduced, ensure they have clear exit conditions, shared state is guarded, and errors propagate rather than being lost. |
| Public API & compatibility | This is an imported library (`github.com/scality/raidmgmt`). Flag breaking changes to exported entities, port signatures, or sentinel errors, since they affect downstream consumers. |
| Security | No credentials/serials/secrets logged or hardcoded; no command injection via untrusted device paths or identifiers. |

## What not to flag

- Anything the linters already own: `golangci-lint` (`.golangci.yaml`), `gofmt`,
  `goimports` — formatting, import order, unused variables, naming.
- Generated files (mockery mocks) except when they are stale with respect to the
  interfaces changed in the same PR.
- Markdown or comment wording preferences.
- Refactors unrelated to the PR's purpose.
