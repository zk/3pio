# 3pio Future Roadmap

## Near-term Enhancements

### Make Support (Future Work)
- Parse Makefiles to extract test commands
- Support common test target patterns
- Transform commands to include 3pio adapters
- Graceful fallback for complex Makefiles

### Multiple Test Runner Support
Run multiple test commands in a single 3pio session:
```bash
# Future capability
3pio multi "npm test" "pytest" "go test ./..."
```
- Concurrent execution of different test runners
- Separate report sections for each runner
- Unified summary across all test suites
- Use case: Projects with multiple languages/test frameworks

### Enhanced Variable Resolution
- Expand Makefile variables beyond simple cases
- Support environment variable substitution
- Handle complex variable references

### Performance Optimizations
- **✅ COMPLETED: Removed `go list` dependency**: Go test no longer uses `go list` for package metadata
  - Eliminated ~200-500ms startup latency
  - Tests are discovered dynamically from JSON output
  - Package information is derived from test events themselves
  - Go test startup is now faster and consistent with other runners
  - Module/workspace support maintained through dynamic discovery

## Medium-term Goals

### Additional Test Runners
- **Ruby**: RSpec, Minitest
- **Java**: JUnit, TestNG
- **Rust**: cargo test integration
- **.NET**: dotnet test integration
- **PHP**: PHPUnit

### CI/CD Integration
- GitHub Actions reporter
- GitLab CI integration
- Jenkins plugin
- CircleCI orb

### Performance Analytics
- Test timing trends over time
- Identify slow tests
- Flaky test detection
- Performance regression alerts

## Long-term Vision

### Universal Test Protocol
- Language-agnostic test reporting standard
- Plugin architecture for any test runner
- Community-contributed adapters

### Distributed Testing
- Coordinate tests across multiple machines
- Aggregate results from parallel runs
- Smart test distribution based on history

### AI-Enhanced Features
- Intelligent test failure analysis
- Suggest fixes based on error patterns
- Predict test failures from code changes
- Auto-generate test documentation

### Developer Experience
- VS Code extension
- IntelliJ plugin
- Real-time test status in editor
- Interactive test debugging

## Experimental Ideas

### Test Impact Analysis
- Map tests to code coverage
- Run only affected tests on changes
- Dependency graph visualization

### Test Quality Metrics
- Mutation testing integration
- Test effectiveness scoring
- Coverage quality analysis

### Collaborative Testing
- Share test results across team
- Crowd-sourced test failure solutions
- Test result annotations and comments

## Community Requested Features

*To be populated based on user feedback*

- Custom report formats
- Webhook notifications
- Test result database storage
- Historical comparison views
- Test categorization and tagging

## Contributing

Have an idea for 3pio? Open an issue or discussion on GitHub to share your suggestions!