# 3pio Future Vision

## Universal Test Reporting Across All Languages

3pio aims to become the universal test reporting tool for AI-assisted development across all major programming languages and test frameworks.

## Distribution Strategy

### Multi-Ecosystem Package Distribution

Each language ecosystem will get native package manager support:

- **JavaScript**: `npm install -g 3pio` (current)
- **Python**: `pip install 3pio` (planned)
- **Ruby**: `gem install 3pio`
- **Go**: `go install 3pio`
- **Rust**: `cargo install 3pio`
- **Java**: `mvn install 3pio` or `gradle install 3pio`
- **.NET**: `dotnet tool install 3pio`

Each distribution includes:
1. The same core 3pio CLI
2. All test runner adapters bundled
3. Native package manager conventions
4. No cross-language dependencies required

### Ultimate Goal: Static Binary Distribution

Future work includes packaging 3pio as a **statically linked binary** with zero runtime dependencies.

**Benefits:**
- No Node.js, Python, or any language runtime required
- Single binary includes all functionality  
- Works on any machine out of the box

**Distribution Channels:**
- Direct download from GitHub releases
- OS package managers: `brew install 3pio`, `apt install 3pio`, `choco install 3pio`
- Language package managers remain as convenience options

## Implementation Approaches for Static Binary

### Option 1: Bun Compilation (Recommended for fast iteration)
- Use `bun build --compile` to create standalone executable
- Advantages: Fast execution, excellent npm compatibility, ~50MB binaries
- Command: `bun build src/cli.ts --compile --outfile 3pio`
- Challenge: Python/Ruby/other language adapters need special handling

### Option 2: Deno Compilation (Mature option)
- Use `deno compile` for single-binary distribution
- Advantages: Mature compilation, built-in security model, web standards
- Command: `deno compile --allow-read --allow-write --allow-run src/cli.ts`
- Challenge: May need code adjustments for Deno compatibility

### Option 3: Native Rewrite (Long-term option)
- Complete rewrite in Rust or Go with embedded adapters
- Advantages: Maximum performance, smallest binaries, true zero-dependency
- Challenge: Significant development effort

**Note**: For JavaScript test runners (Jest/Vitest), Bun/Deno compilation would provide true zero-dependency binaries. For Python/Ruby/other languages, the binary could either:
- Embed adapters as strings and write them to temp files at runtime
- Require the target language runtime (Python for pytest, Ruby for RSpec, etc.)

## Supported Test Frameworks Roadmap

### Currently Supported
- **JavaScript**: Jest, Vitest
- **Python**: pytest (in development)

### Planned Support
- **Ruby**: RSpec, Minitest
- **Go**: Go test, Ginkgo
- **Rust**: cargo test, nextest
- **Java**: JUnit, TestNG, Spock
- **.NET**: NUnit, xUnit, MSTest
- **PHP**: PHPUnit, Pest
- **Swift**: XCTest
- **Kotlin**: Kotest

## Design Principles

1. **One Tool, Many Distributions**: Same functionality regardless of installation method
2. **Zero Configuration**: Works out of the box with sensible defaults
3. **Native Ecosystem Integration**: Respects each language's conventions
4. **AI-First Output**: Structured, parseable output optimized for LLM consumption
5. **Progressive Enhancement**: Basic functionality with no dependencies, enhanced features with runtimes

This approach scales naturally as 3pio becomes the standard for test reporting in AI-assisted development workflows.