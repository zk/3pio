.PHONY: build clean test install dev adapters all

# Build variables
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-go")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Directories
BUILD_DIR := build
DIST_DIR := dist
ADAPTER_DIR := adapters

all: clean adapters build

# Build the Go binary
build: adapters
	@echo "Building 3pio binary..."
	@mkdir -p $(BUILD_DIR)
ifeq ($(OS),Windows_NT)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/3pio.exe cmd/3pio/main.go
	@echo "Binary built: $(BUILD_DIR)/3pio.exe"
else
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/3pio cmd/3pio/main.go
	@echo "✅ Binary built: $(BUILD_DIR)/3pio"
endif

# Build adapters and prepare for embedding
adapters:
	@echo "Preparing adapters for embedding..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/prepare-adapters.ps1
else
	@./scripts/prepare-adapters.sh
endif

# Development build (with debug symbols)
dev:
	@echo "Building development binary..."
	@mkdir -p $(BUILD_DIR)
	go build -gcflags "all=-N -l" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/3pio cmd/3pio/main.go
	@echo "✅ Development binary built: $(BUILD_DIR)/3pio"

# Run tests
test:
	@echo "Running Go tests..."
	go test -v -race ./internal/...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v ./tests/integration_go/...

# Run all tests
test-all: test test-integration

# Generate test coverage
coverage:
	@echo "Generating test coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Install locally
install: build
	@echo "Installing 3pio..."
	go install -ldflags "$(LDFLAGS)" ./cmd/3pio
	@echo "✅ 3pio installed to $(GOPATH)/bin"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	go mod tidy

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		go vet ./...; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html *.test
	@find . -name ".3pio" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "✅ Clean complete"

# Cross-platform builds
build-all: clean adapters
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/3pio-darwin-amd64 cmd/3pio/main.go
	
	# macOS ARM64 (M1)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/3pio-darwin-arm64 cmd/3pio/main.go
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/3pio-linux-amd64 cmd/3pio/main.go
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/3pio-linux-arm64 cmd/3pio/main.go
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/3pio-windows-amd64.exe cmd/3pio/main.go
	
	@echo "✅ All platforms built in $(BUILD_DIR)/"

# Run with example commands
run-jest: build
	./$(BUILD_DIR)/3pio npx jest

run-vitest: build
	./$(BUILD_DIR)/3pio npx vitest run

run-npm-test: build
	./$(BUILD_DIR)/3pio npm test

# Package for distribution
package: clean adapters
	@echo "Building and packaging for distribution..."
	@# Build binaries using goreleaser
	@if [ -f .goreleaser.local.yml ]; then \
		goreleaser build --config .goreleaser.local.yml --clean --snapshot; \
	else \
		@echo "Creating local goreleaser config..."; \
		@cat > .goreleaser.local.yml <<EOF; \
version: 2\n\
project_name: 3pio\n\
before:\n\
  hooks:\n\
    - make adapters\n\
builds:\n\
  - id: 3pio\n\
    main: ./cmd/3pio/main.go\n\
    binary: 3pio\n\
    goos:\n\
      - linux\n\
      - darwin\n\
      - windows\n\
    goarch:\n\
      - amd64\n\
      - arm64\n\
    ignore:\n\
      - goos: windows\n\
        goarch: arm64\n\
    ldflags:\n\
      - -s -w\n\
      - -X main.version={{.Version}}\n\
      - -X main.commit={{.Commit}}\n\
      - -X main.date={{.Date}}\n\
    env:\n\
      - CGO_ENABLED=0\n\
archives:\n\
  - id: default\n\
    name_template: '{{ .ProjectName }}-{{ .Os }}-{{ .Arch }}'\n\
    format: tar.gz\n\
    format_overrides:\n\
      - goos: windows\n\
        format: zip\n\
checksum:\n\
  name_template: '{{ .ProjectName }}-{{ .Version }}-checksums.txt'\n\
snapshot:\n\
  version_template: "{{ .Tag }}-next"\n\
release:\n\
  disable: true\n\
EOF; \
		goreleaser build --config .goreleaser.local.yml --clean --snapshot; \
	fi
	
	@echo "Copying binaries to npm package..."
	@mkdir -p packaging/npm/binaries
	@cp dist/3pio_darwin_amd64*/3pio packaging/npm/binaries/3pio-darwin-amd64
	@cp dist/3pio_darwin_arm64*/3pio packaging/npm/binaries/3pio-darwin-arm64
	@cp dist/3pio_linux_amd64*/3pio packaging/npm/binaries/3pio-linux-amd64
	@cp dist/3pio_linux_arm64*/3pio packaging/npm/binaries/3pio-linux-arm64
	@cp dist/3pio_windows_amd64*/3pio.exe packaging/npm/binaries/3pio-windows-amd64.exe
	
	@echo "Copying binaries to pip package..."
	@mkdir -p packaging/pip/threepio/binaries
	@cp dist/3pio_darwin_amd64*/3pio packaging/pip/threepio/binaries/3pio-darwin-amd64
	@cp dist/3pio_darwin_arm64*/3pio packaging/pip/threepio/binaries/3pio-darwin-arm64
	@cp dist/3pio_linux_amd64*/3pio packaging/pip/threepio/binaries/3pio-linux-amd64
	@cp dist/3pio_linux_arm64*/3pio packaging/pip/threepio/binaries/3pio-linux-arm64
	@cp dist/3pio_windows_amd64*/3pio.exe packaging/pip/threepio/binaries/3pio-windows-amd64.exe
	
	@echo "✅ Packaging complete!"
	@echo ""
	@echo "To publish npm package:"
	@echo "  cd packaging/npm && npm publish"
	@echo ""
	@echo "To publish pip package:"
	@echo "  cd packaging/pip && python -m build && twine upload dist/*"

# Publish npm package
publish-npm: package
	@echo "Publishing npm package..."
	cd packaging/npm && npm publish
	@echo "✅ npm package published"

# Publish pip package
publish-pip: package
	@echo "Building and publishing pip package..."
	cd packaging/pip && \
		rm -rf dist build *.egg-info && \
		python -m build && \
		twine upload dist/*
	@echo "✅ pip package published"

# Publish both packages
publish: package
	@echo "Publishing to npm and pip..."
	@$(MAKE) publish-npm
	@$(MAKE) publish-pip
	@echo "✅ All packages published"

# Help
help:
	@echo "3pio Makefile targets:"
	@echo "  make build           - Build the 3pio binary"
	@echo "  make adapters        - Build test runner adapters"
	@echo "  make dev             - Build with debug symbols"
	@echo "  make test            - Run unit tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-all        - Run all tests"
	@echo "  make coverage        - Generate test coverage report"
	@echo "  make fmt             - Format code"
	@echo "  make lint            - Run linter"
	@echo "  make install         - Install 3pio locally"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make build-all       - Build for all platforms"
	@echo "  make package         - Build and prepare packages for npm and pip"
	@echo "  make publish-npm     - Build and publish npm package"
	@echo "  make publish-pip     - Build and publish pip package"
	@echo "  make publish         - Build and publish both npm and pip packages"
	@echo "  make help            - Show this help message"