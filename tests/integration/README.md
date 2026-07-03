# Integration / e2e harnesses

Manual, on-hardware harnesses for the RAID controller adapters. Each adapter has
its own clearly-identifiable sibling directory, all built as standalone
`package main` programs:

| Directory | Adapter | Hardware |
|---|---|---|
| [`mdadm/`](mdadm/) | software RAID (RHEL8 / mdadm) | NVMe drives + `mdadm` |
| [`storcli2/`](storcli2/) | MegaRAID / PERC (storcli2 / perccli2) | controller + `storcli2`/`perccli2` binary |

These are **not** part of `go test` or CI: they shell out to real tools and
mutate real storage. Run them by hand on a host with the right hardware.

## mdadm

Runs a fixed destructive RAID0/RAID1/RAID10 suite (create, add/remove drives,
delete) against `/dev/nvme*` devices:

```sh
go run ./tests/integration/mdadm
```

## storcli2

Argument-driven. A bare invocation is **read-only** (inventory as markdown
tables); destructive commands run only with `-confirm`.

```sh
# read-only inventory (default)
go run ./tests/integration/storcli2

# full destructive cycle: create -> assert remove unsupported -> expand -> delete
go run ./tests/integration/storcli2 scenario -raid=1 -drives=252:0,252:1 -add-drives=252:2 -confirm

# individual destructive tasks
go run ./tests/integration/storcli2 create -raid=1 -drives=252:0,252:1 -confirm
go run ./tests/integration/storcli2 add    -vd=0 -drives=252:2 -confirm
go run ./tests/integration/storcli2 delete -vd=0 -confirm
```

Flags: `-binary` (default `/opt/MegaRAID/storcli2/storcli2`, set to the
`perccli2` path for PERC), `-controller` (default `0`), `-raid`, `-drives`,
`-add-drives`, `-vd`, `-confirm`. Drives are addressed by their `EID:Slt` id.

> storcli2 cannot remove drives from a volume, so the `scenario` exercises
> removal as a negative case (asserts `ErrFunctionNotSupportedByImplementation`)
> rather than mutating the array.

### Building for the RAID hosts

The harnesses only shell out to the vendor binaries, so they cross-compile
freely. The Makefile builds them as **statically linked `linux/amd64`** binaries
into `bin/`, which depend on no glibc and therefore run on both Rocky Linux 8
(glibc 2.28) and Rocky Linux 9 (glibc 2.34):

```sh
make build-e2e            # both harnesses -> bin/{mdadm-e2e,storcli2-e2e}
make build-e2e-storcli2   # storcli2 only
```

Override the target for other hosts via the `E2E_GOOS` / `E2E_GOARCH` variables,
e.g. `make build-e2e E2E_GOARCH=arm64`. Then copy the binary to the target host
and run it there.
