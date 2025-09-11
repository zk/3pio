#!/bin/bash
set -e

# Prepare pip package with binaries from GoReleaser build

echo "Preparing pip package..."

# Check if binaries exist from previous build
if [ ! -d "dist" ]; then
    echo "Building binaries with GoReleaser..."
    
    # Create a temporary goreleaser config for local builds
    cat > .goreleaser.pip.yml << 'EOF'
version: 2
project_name: 3pio
before:
  hooks:
    - make adapters
builds:
  - id: 3pio
    main: ./cmd/3pio/main.go
    binary: 3pio
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
    env:
      - CGO_ENABLED=0
snapshot:
  version_template: "local"
EOF

    goreleaser build --config .goreleaser.pip.yml --snapshot --clean
    rm .goreleaser.pip.yml
fi

# Create binaries directory in pip package
mkdir -p packaging/pip/threepio/binaries

# Copy binaries with proper naming
echo "Copying binaries to pip package..."

# Darwin AMD64
if [ -f "dist/3pio_darwin_amd64_v1/3pio" ]; then
    cp dist/3pio_darwin_amd64_v1/3pio packaging/pip/threepio/binaries/3pio-darwin-amd64
    echo "  - Copied darwin-amd64"
fi

# Darwin ARM64
if [ -f "dist/3pio_darwin_arm64_v8.0/3pio" ]; then
    cp dist/3pio_darwin_arm64_v8.0/3pio packaging/pip/threepio/binaries/3pio-darwin-arm64
    echo "  - Copied darwin-arm64"
fi

# Linux AMD64
if [ -f "dist/3pio_linux_amd64_v1/3pio" ]; then
    cp dist/3pio_linux_amd64_v1/3pio packaging/pip/threepio/binaries/3pio-linux-amd64
    echo "  - Copied linux-amd64"
fi

# Linux ARM64
if [ -f "dist/3pio_linux_arm64_v8.0/3pio" ]; then
    cp dist/3pio_linux_arm64_v8.0/3pio packaging/pip/threepio/binaries/3pio-linux-arm64
    echo "  - Copied linux-arm64"
fi

# Windows AMD64
if [ -f "dist/3pio_windows_amd64_v1/3pio.exe" ]; then
    cp dist/3pio_windows_amd64_v1/3pio.exe packaging/pip/threepio/binaries/3pio-windows-amd64.exe
    echo "  - Copied windows-amd64"
fi

# Show package size
echo ""
echo "Package contents:"
ls -lh packaging/pip/threepio/binaries/

# Calculate total size
TOTAL_SIZE=$(du -sh packaging/pip/threepio/binaries | cut -f1)
echo ""
echo "Total binaries size: $TOTAL_SIZE"

echo ""
echo "pip package is ready for building at packaging/pip/"
echo "To build and publish:"
echo "  cd packaging/pip"
echo "  python -m build"
echo "  python -m twine upload dist/*"