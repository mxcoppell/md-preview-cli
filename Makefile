BINARY    := md-preview-cli
BUILD_DIR := bin
VERSION   := $(shell git describe --tags 2>/dev/null || git rev-parse --short HEAD)
LDFLAGS   := -X github.com/mxcoppell/md-preview-cli/internal/version.Version=$(VERSION)

.PHONY: build release clean test deps

## Development build — includes debug symbols, version set to "dev"
build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/md-preview-cli

## Release build — stripped symbols, version injected from git
release:
	go build -ldflags="-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) .

## Run all tests
test:
	go test ./...

## Download vendored JS dependencies
deps:
	./scripts/download-deps.sh

## Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
