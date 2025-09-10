#!/bin/bash

# Verify adapter files exist for embedding in Go binary
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ADAPTER_DIR="$PROJECT_ROOT/internal/adapters"

echo "Verifying adapters for embedding..."

# Check that adapter files exist
MISSING_FILES=()

if [ ! -f "$ADAPTER_DIR/jest.js" ]; then
    MISSING_FILES+=("jest.js")
fi

if [ ! -f "$ADAPTER_DIR/vitest.js" ]; then
    MISSING_FILES+=("vitest.js")
fi

if [ ! -f "$ADAPTER_DIR/pytest_adapter.py" ]; then
    MISSING_FILES+=("pytest_adapter.py")
fi

# Report missing files
if [ ${#MISSING_FILES[@]} -gt 0 ]; then
    echo "❌ Missing adapter files:"
    for file in "${MISSING_FILES[@]}"; do
        echo "  - $ADAPTER_DIR/$file"
    done
    echo ""
    echo "These files should be committed to the repository."
    exit 1
fi

echo "✅ All adapters present for embedding"
ls -la "$ADAPTER_DIR"/*.{js,py} 2>/dev/null || true