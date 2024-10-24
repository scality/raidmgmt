.DEFAULT_GOAL := all

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

.PHONY: all
all: lint tests
	@echo "All done"
