#!/bin/bash

# Prepare adapter files for embedding in Go binary
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ADAPTER_DIR="$PROJECT_ROOT/internal/adapters"

echo "Preparing adapters for embedding..."

# Ensure adapter directory exists
mkdir -p "$ADAPTER_DIR"

# Build TypeScript adapters if needed
if [ ! -f "$PROJECT_ROOT/dist/jest.js" ] || [ ! -f "$PROJECT_ROOT/dist/vitest.js" ]; then
    echo "Building TypeScript adapters..."
    cd "$PROJECT_ROOT"
    npm run build
fi

# Copy adapter files
echo "Copying adapter files..."
cp "$PROJECT_ROOT/dist/jest.js" "$ADAPTER_DIR/jest.js"
# Vitest is ESM, keep as .js but will extract as .mjs
cp "$PROJECT_ROOT/dist/vitest.js" "$ADAPTER_DIR/vitest.js"

# Copy Python adapter
if [ -f "$PROJECT_ROOT/dist/pytest_adapter.py" ]; then
    cp "$PROJECT_ROOT/dist/pytest_adapter.py" "$ADAPTER_DIR/pytest_adapter.py"
elif [ -f "$PROJECT_ROOT/src/adapters/pytest/pytest_adapter.py" ]; then
    cp "$PROJECT_ROOT/src/adapters/pytest/pytest_adapter.py" "$ADAPTER_DIR/pytest_adapter.py"
fi

echo "✅ Adapters prepared for embedding"
ls -la "$ADAPTER_DIR"/*.{js,py} 2>/dev/null || true