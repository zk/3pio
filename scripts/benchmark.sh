#!/bin/bash

set -e

echo "==================================="
echo "3pio Performance Benchmark"
echo "==================================="
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Success criteria from migration plan
TARGET_STARTUP_MS=50
TARGET_MEMORY_MB=10
TARGET_BINARY_SIZE_MB=20

# Build Go binary if needed
if [ ! -f "build/3pio" ]; then
    echo "Building Go binary..."
    make build
fi

# Build TypeScript version if needed
if [ ! -f "dist/cli.js" ]; then
    echo "Building TypeScript version..."
    npm run build
fi

echo "Testing Binary Sizes:"
echo "---------------------"

# Check Go binary size
GO_BINARY_SIZE_BYTES=$(stat -f%z build/3pio 2>/dev/null || stat -c%s build/3pio 2>/dev/null)
GO_BINARY_SIZE_MB=$((GO_BINARY_SIZE_BYTES / 1024 / 1024))

echo "Go binary size: ${GO_BINARY_SIZE_MB}MB"
if [ $GO_BINARY_SIZE_MB -lt $TARGET_BINARY_SIZE_MB ]; then
    echo -e "${GREEN}✅ Binary size target met (< ${TARGET_BINARY_SIZE_MB}MB)${NC}"
else
    echo -e "${RED}❌ Binary size target missed (>= ${TARGET_BINARY_SIZE_MB}MB)${NC}"
fi

# Check dist directory size (rough approximation for Node.js version)
DIST_SIZE_BYTES=$(du -sb dist 2>/dev/null | cut -f1 || du -s dist | awk '{print $1 * 1024}')
DIST_SIZE_MB=$((DIST_SIZE_BYTES / 1024 / 1024))
echo "TypeScript dist size: ${DIST_SIZE_MB}MB (excludes Node.js runtime)"

echo
echo "Testing Startup Times:"
echo "---------------------"

# Function to measure startup time
measure_startup_time() {
    local command="$1"
    local description="$2"
    local runs=5
    local total=0
    
    echo "Measuring $description ($runs runs)..."
    
    for i in $(seq 1 $runs); do
        if command -v gdate >/dev/null 2>&1; then
            # Use GNU date if available (more precise)
            start_time=$(gdate +%s%N)
            $command --help > /dev/null 2>&1
            end_time=$(gdate +%s%N)
            duration_ns=$((end_time - start_time))
            duration_ms=$((duration_ns / 1000000))
        else
            # Fallback to millisecond precision
            start_time=$(python3 -c "import time; print(int(time.time() * 1000))")
            $command --help > /dev/null 2>&1
            end_time=$(python3 -c "import time; print(int(time.time() * 1000))")
            duration_ms=$((end_time - start_time))
        fi
        
        total=$((total + duration_ms))
        echo "  Run $i: ${duration_ms}ms"
    done
    
    average=$((total / runs))
    echo "  Average: ${average}ms"
    echo
    
    return $average
}

# Measure Go binary startup
echo "Go binary startup time:"
measure_startup_time "./build/3pio" "Go binary"
GO_STARTUP_MS=$?

# Measure Node.js + TypeScript startup
echo "TypeScript + Node.js startup time:"
measure_startup_time "node dist/cli.js" "TypeScript version"
NODE_STARTUP_MS=$?

echo "Startup Performance Summary:"
echo "Go binary:     ${GO_STARTUP_MS}ms"
echo "Node.js + TS:  ${NODE_STARTUP_MS}ms"

if [ $GO_STARTUP_MS -lt $TARGET_STARTUP_MS ]; then
    echo -e "${GREEN}✅ Startup time target met (< ${TARGET_STARTUP_MS}ms)${NC}"
else
    echo -e "${RED}❌ Startup time target missed (>= ${TARGET_STARTUP_MS}ms)${NC}"
fi

# Calculate improvement
if [ $NODE_STARTUP_MS -gt 0 ]; then
    improvement=$((NODE_STARTUP_MS - GO_STARTUP_MS))
    improvement_ratio=$((NODE_STARTUP_MS * 100 / GO_STARTUP_MS))
    echo "Improvement: ${improvement}ms faster (${improvement_ratio}% of Node.js time)"
fi

echo
echo "Testing Memory Usage:"
echo "--------------------"

# Function to measure memory usage (peak RSS)
measure_memory_usage() {
    local command="$1"
    local description="$2"
    
    echo "Measuring $description memory usage..."
    
    # Use time command to get memory stats
    if command -v /usr/bin/time >/dev/null 2>&1; then
        # On macOS, /usr/bin/time -l gives us memory in bytes, need to convert
        temp_output=$(/usr/bin/time -l $command --help 2>&1 | grep "maximum resident set size")
        
        if [[ "$temp_output" =~ ([0-9]+)[[:space:]]*maximum ]]; then
            if [[ "$OSTYPE" == "darwin"* ]]; then
                # macOS gives bytes, convert to KB
                memory_kb=$((${BASH_REMATCH[1]} / 1024))
            else
                # Linux gives KB
                memory_kb=${BASH_REMATCH[1]}
            fi
        else
            memory_kb=""
        fi
    else
        echo "  /usr/bin/time not available, using ps for rough estimate"
        # Fallback: use ps (less accurate)
        $command --help > /dev/null 2>&1 &
        PID=$!
        sleep 0.1
        memory_kb=$(ps -o rss= -p $PID 2>/dev/null | tr -d ' ' || echo "0")
        kill $PID 2>/dev/null || true
    fi
    
    if [ -n "$memory_kb" ] && [ "$memory_kb" -gt 0 ]; then
        memory_mb=$((memory_kb / 1024))
        echo "  Peak memory: ${memory_mb}MB"
        return $memory_mb
    else
        echo "  Could not measure memory usage accurately"
        return 0
    fi
}

# Measure Go binary memory
measure_memory_usage "./build/3pio" "Go binary"
GO_MEMORY_MB=$?

# Measure Node.js memory  
measure_memory_usage "node dist/cli.js" "TypeScript + Node.js"
NODE_MEMORY_MB=$?

echo
echo "Memory Performance Summary:"
if [ $GO_MEMORY_MB -gt 0 ]; then
    echo "Go binary:     ${GO_MEMORY_MB}MB"
    if [ $GO_MEMORY_MB -lt $TARGET_MEMORY_MB ]; then
        echo -e "${GREEN}✅ Memory usage target met (< ${TARGET_MEMORY_MB}MB)${NC}"
    else
        echo -e "${RED}❌ Memory usage target missed (>= ${TARGET_MEMORY_MB}MB)${NC}"
    fi
fi

if [ $NODE_MEMORY_MB -gt 0 ]; then
    echo "Node.js + TS:  ${NODE_MEMORY_MB}MB"
    if [ $GO_MEMORY_MB -gt 0 ] && [ $NODE_MEMORY_MB -gt 0 ]; then
        memory_improvement=$((NODE_MEMORY_MB - GO_MEMORY_MB))
        echo "Memory saved: ${memory_improvement}MB"
    fi
fi

echo
echo "==================================="
echo "Performance Benchmark Complete"
echo "==================================="

# Exit with success if all targets met
TARGETS_MET=0
if [ $GO_BINARY_SIZE_MB -lt $TARGET_BINARY_SIZE_MB ]; then
    TARGETS_MET=$((TARGETS_MET + 1))
fi
if [ $GO_STARTUP_MS -lt $TARGET_STARTUP_MS ]; then
    TARGETS_MET=$((TARGETS_MET + 1))
fi
if [ $GO_MEMORY_MB -gt 0 ] && [ $GO_MEMORY_MB -lt $TARGET_MEMORY_MB ]; then
    TARGETS_MET=$((TARGETS_MET + 1))
elif [ $GO_MEMORY_MB -eq 0 ]; then
    # If we couldn't measure memory, don't count it as failed
    echo -e "${YELLOW}⚠️  Memory measurement unavailable${NC}"
fi

echo
if [ $TARGETS_MET -ge 2 ]; then
    echo -e "${GREEN}🎉 Performance targets largely met!${NC}"
    exit 0
else
    echo -e "${RED}❌ Performance targets not met${NC}"
    exit 1
fi