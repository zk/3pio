# 3pio Future Roadmap

## Near-term Enhancements

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


### Past duration metrics

Prevent premature kills and to low timeouts by including past duration metrics (last run duration, avg, 99th) in a way that the agent can use to better run tests.

