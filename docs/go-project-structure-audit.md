# Go Project Structure Audit

## Current Structure Analysis

### ✅ Well-Structured Components

1. **`cmd/3pio/`** - Correct location for the main application entry point
   - Follows Go convention for executable commands
   - Clean separation of main.go

2. **`internal/`** - Properly uses internal packages for private code
   - `adapters/` - Embedded JavaScript/Python adapters
   - `ipc/` - Inter-process communication
   - `orchestrator/` - Main orchestration logic
   - `report/` - Report generation
   - `runner/` - Test runner detection and management
   - `utils/` - Shared utilities

3. **`tests/`** - Good test organization
   - `integration_go/` - Integration tests
   - `fixtures/` - Test fixtures with sample projects

4. **`build/`** - Appropriate for build artifacts

5. **`scripts/`** - Shell scripts for automation

### 🔧 Areas for Improvement

## Recommended Structure Changes

### 1. Move Test Fixtures to Follow Go Convention

**Current:** `tests/fixtures/`
**Recommended:** `testdata/fixtures/`

Go convention uses `testdata` directory which is ignored by the Go toolchain.

```bash
# Migration commands
mkdir -p testdata
mv tests/fixtures testdata/
# Update integration tests to reference new path
```

### 2. Reorganize Tests

**Current Structure:**
```
tests/
├── fixtures/        # Should move to testdata/
└── integration_go/  # Integration tests
```

**Recommended Structure:**
```
# Unit tests stay with their packages
internal/
├── orchestrator/
│   └── orchestrator_test.go
├── report/
│   └── manager_test.go
└── runner/
    └── *_test.go

# Integration tests in root
integration_test.go     # Main integration tests
testdata/              # Test fixtures
└── fixtures/
    ├── basic-jest/
    ├── basic-vitest/
    └── ...
```

### 3. Add Missing Standard Directories

Create these directories for better organization:

```bash
# For any future public APIs
mkdir -p pkg

# For auto-generated code (if needed)
mkdir -p gen

# For compiled binaries by platform
mkdir -p dist
```

### 4. Documentation Structure

Enhance documentation organization:

```bash
docs/
├── api/              # API documentation
├── architecture/     # Already exists - good!
├── development/      # Development guides
├── deployment/       # Deployment instructions
└── migration/        # Migration guides (TypeScript to Go)
```

### 5. Clean Up Temporary/Development Files

Remove or relocate:
- `scratch/` - Consider moving to `/tmp` or `.gitignore`
- `packaging/` - Good structure, but consider moving pip/npm artifacts since we're Go-only now
- `claude-plans/` - Consider `.claude/plans/` to keep AI-related files together

### 6. Configuration and Build Files

Add standard Go project files:

```bash
# Makefile for common tasks
touch Makefile

# GitHub Actions workflow (if not exists)
mkdir -p .github/workflows

# Goreleaser config for releases
touch .goreleaser.yml
```

## Recommended Project Structure

```
3pio/
├── .github/
│   └── workflows/         # CI/CD pipelines
├── cmd/
│   └── 3pio/             # Main application
│       └── main.go
├── internal/             # Private packages
│   ├── adapters/         # Embedded adapters
│   ├── ipc/             # IPC communication
│   ├── orchestrator/    # Core orchestration
│   ├── report/          # Report generation
│   ├── runner/          # Test runner management
│   └── utils/           # Shared utilities
├── pkg/                  # Public packages (future)
├── testdata/            # Test fixtures
│   └── fixtures/
├── docs/                # Documentation
│   ├── api/
│   ├── architecture/
│   ├── development/
│   └── migration/
├── scripts/             # Build and utility scripts
├── dist/               # Distribution binaries
├── .gitignore
├── README.md
├── LICENSE
├── Makefile            # Build automation
├── go.mod
├── go.sum
└── .goreleaser.yml     # Release configuration
```

## Makefile Template

```makefile
.PHONY: build test clean install

VERSION := $(shell git describe --tags --always --dirty)
GOFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(GOFLAGS) -o dist/3pio ./cmd/3pio

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

clean:
	rm -rf dist/ build/

install:
	go install $(GOFLAGS) ./cmd/3pio

release:
	goreleaser release --clean
```

## Priority Actions

1. **High Priority:**
   - Move `tests/fixtures/` to `testdata/fixtures/`
   - Create Makefile for standardized builds
   - Clean up scratch directory

2. **Medium Priority:**
   - Reorganize documentation structure
   - Set up .goreleaser.yml for releases
   - Remove obsolete packaging directories (npm, pip)

3. **Low Priority:**
   - Add pkg/ directory when public APIs are needed
   - Consider moving integration tests to root level

## Benefits of Restructuring

1. **Standards Compliance:** Follows Go community conventions
2. **Tool Compatibility:** Works better with Go toolchain and IDEs
3. **Clarity:** Clear separation of concerns
4. **Maintainability:** Easier for new contributors to understand
5. **Build Efficiency:** Proper structure for CI/CD pipelines