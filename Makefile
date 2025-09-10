.PHONY: build clean test install dev adapters all

# Build variables
VERSION := 0.0.1-go
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
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/3pio cmd/3pio/main.go
	@echo "✅ Binary built: $(BUILD_DIR)/3pio"

# Build adapters and prepare for embedding
adapters:
	@echo "Preparing adapters for embedding..."
	@./scripts/prepare-adapters.sh

# Development build (with debug symbols)
dev:
	@echo "Building development binary..."
	@mkdir -p $(BUILD_DIR)
	go build -gcflags "all=-N -l" -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/3pio cmd/3pio/main.go
	@echo "✅ Development binary built: $(BUILD_DIR)/3pio"

# Run tests
test:
	@echo "Running Go tests..."
	go test -v ./...

# Install locally
install: build
	@echo "Installing 3pio..."
	go install -ldflags "$(LDFLAGS)" ./cmd/3pio
	@echo "✅ 3pio installed to $(GOPATH)/bin"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
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

# Help
help:
	@echo "3pio Makefile targets:"
	@echo "  make build      - Build the 3pio binary"
	@echo "  make adapters   - Build test runner adapters"
	@echo "  make dev        - Build with debug symbols"
	@echo "  make test       - Run tests"
	@echo "  make install    - Install 3pio locally"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make build-all  - Build for all platforms"
	@echo "  make help       - Show this help message"