.DEFAULT_GOAL := all

BIN_DIR := bin

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

# Build the on-hardware integration harnesses. These are manual, destructive
# tools (see tests/integration/README.md); building them just verifies they
# still compile.
.PHONY: build-e2e-mdadm
build-e2e-mdadm:
	@echo "Building mdadm e2e harness..."
	go build -o $(BIN_DIR)/mdadm-e2e ./tests/integration/mdadm

.PHONY: build-e2e-storcli2
build-e2e-storcli2:
	@echo "Building storcli2 e2e harness..."
	go build -o $(BIN_DIR)/storcli2-e2e ./tests/integration/storcli2

.PHONY: build-e2e
build-e2e: build-e2e-mdadm build-e2e-storcli2
	@echo "e2e harnesses built"

.PHONY: all
all: lint tests build-e2e
	@echo "All done"
