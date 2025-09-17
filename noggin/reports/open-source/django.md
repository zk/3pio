# 3pio Test Report: Django

**Project**: django
**Framework(s)**: Python (Django test runner) - not currently supported by 3pio
**Test Date**: 2025-09-15
**3pio Version**: (from build)

## Project Analysis
- Project type: Python web framework
- Test framework(s): Django's built-in test runner (not pytest/unittest)
- Test command(s): `python tests/runtests.py` (custom test runner)
- Test suite size: Large comprehensive web framework test suite

## 3pio Test Results
### Command: `../../build/3pio python tests/runtests.py`
- **Status**: NOT SUPPORTED
- **Exit Code**: N/A
- **Detection**: Django uses custom test runner, not supported by 3pio
- **Output**: 3pio only supports Jest, Vitest, pytest, and cargo test

### Project Structure
- Uses Django's custom test runner system
- Tests located in tests/ directory with runtests.py entry point
- Complex test configuration with tox for multiple environments
- Does not use standard pytest/unittest runners

### Expected Compatibility
- 3pio does not support Django's test runner
- Would need Django test adapter to support this project
- Django has its own test discovery and execution system

### Recommendations
1. Consider adding Django test runner support to 3pio adapters
2. Django could potentially be configured to use pytest instead
3. Many Django projects do use pytest-django for testing
4. This would require significant adapter development for 3pio