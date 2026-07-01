.DEFAULT_GOAL := all

BIN_DIR := bin

# The e2e harnesses are deployed to RAID hosts running Rocky Linux 8 or 9 on
# x86-64. Building them as statically linked (CGO disabled) linux/amd64 binaries
# makes them depend on no glibc at all, so a single binary runs on both releases
# (Rocky 8 ships glibc 2.28, Rocky 9 ships glibc 2.34). Override on the command
# line for other targets, e.g. `make build-e2e GOARCH=arm64`.
E2E_GOOS ?= linux
E2E_GOARCH ?= amd64
E2E_BUILD_ENV := CGO_ENABLED=0 GOOS=$(E2E_GOOS) GOARCH=$(E2E_GOARCH)

# TODO
.PHONY: lint
lint:
	@echo "Linting..."
	golangci-lint run -c .golangci.yaml ./...
	@echo "Lint done"

# TODO
.PHONY: tests
tests:
	@echo "Running tests..."
	go test -v ./...
	@echo "Tests done"

# Build the on-hardware integration harnesses as static linux/amd64 binaries
# (Rocky Linux 8/9 compatible, see E2E_* above). These are manual, destructive
# tools (see tests/integration/README.md); building them just verifies they
# still compile.
.PHONY: build-e2e-mdadm
build-e2e-mdadm:
	@echo "Building mdadm e2e harness ($(E2E_GOOS)/$(E2E_GOARCH))..."
	$(E2E_BUILD_ENV) go build -o $(BIN_DIR)/mdadm-e2e ./tests/integration/mdadm

.PHONY: build-e2e-storcli2
build-e2e-storcli2:
	@echo "Building storcli2 e2e harness ($(E2E_GOOS)/$(E2E_GOARCH))..."
	$(E2E_BUILD_ENV) go build -o $(BIN_DIR)/storcli2-e2e ./tests/integration/storcli2

.PHONY: build-e2e
build-e2e: build-e2e-mdadm build-e2e-storcli2
	@echo "e2e harnesses built"

.PHONY: all
all: lint tests build-e2e
	@echo "All done"
