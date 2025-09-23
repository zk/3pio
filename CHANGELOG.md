# Changelog

All notable changes to 3pio will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.7.3] - 2025-09-23

### Fixed
- Fixed pytest xdist deduplication to prevent duplicate test reporting in parallel mode
- Improved test tracking accuracy by establishing GroupManager as single source of truth
- Fixed nil pointer panic in orchestrator when commandErr is nil
- Resolved slice bounds panic in command parsing
- Fixed cross-env wrapper handling in npm scripts

### Added
- Comprehensive worker ID tracking for pytest xdist parallelization
- ERROR status support for tests that fail during setup/teardown
- Better handling of tests skipped during setup phase

### Improved
- Test status determination with consistent PASS/SKIP/FAIL rules applied recursively
- Setup failures now properly treated as FAIL and preserved in reports
- Vitest v2 pending test handling with end-of-run reconciliation
- Test path resolution to handle working directory changes correctly

### Documentation
- Added report status rules documentation
- Created verified libraries documentation with tested open source projects
- Streamlined technical documentation by removing outdated planning docs
- Updated CLAUDE.md with latest project guidelines

## [0.7.2] - Previous Release

### Features
- Vitest v2 and v3 support
- Yarn script extraction
- Improved test runner compatibility

## [0.7.1] - Previous Release

### Bug Fixes
- Fixed test tracking issues
- Improved report generation stability

## [0.7.0] - Previous Release

### Major Features
- Go implementation
- Multi-platform support
- Enhanced performance

[0.7.3]: https://github.com/zk/3pio/compare/v0.7.2...v0.7.3
[0.7.2]: https://github.com/zk/3pio/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/zk/3pio/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/zk/3pio/compare/v0.6.0...v0.7.0