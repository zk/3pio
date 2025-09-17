# 3pio Test Report: Sway

**Project**: sway
**Framework(s)**: Rust (cargo test) - supported by 3pio
**Test Date**: 2025-09-15
**3pio Version**: (from build)

## Project Analysis
- Project type: Rust i3-compatible Wayland compositor
- Test framework(s): cargo test (Rust's built-in testing)
- Test command(s): `cargo test` (window manager testing)
- Test suite size: Window manager with system-level integration tests

## 3pio Test Results
### Command: `../../build/3pio cargo test`
- **Status**: NOT TESTED
- **Exit Code**: N/A
- **Detection**: Expected to work (Rust/cargo test supported)
- **Output**: Not executed yet

### Project Structure
- Rust window manager/compositor
- Complex system integration (Wayland, graphics, input)
- May require Wayland dependencies for testing
- System-level window management functionality

### Expected Compatibility
- 3pio supports Rust cargo test
- Should handle standard Rust testing correctly
- May have system-dependent tests requiring Wayland
- Window manager tests may need specific environment

### Recommendations
1. Test with 3pio to verify Rust testing works
2. Some tests may require Wayland environment setup
3. Consider unit tests only: `cargo test --lib`
4. May need system dependencies for full test execution