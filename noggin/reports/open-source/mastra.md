# 3pio Test Report: Mastra

**Project**: mastra
**Framework(s)**: Vitest
**Test Date**: 2025-09-15
**3pio Version**: (from build)

## Project Analysis
- Project type: TypeScript monorepo with pnpm workspaces
- Test framework(s): Vitest
- Test command(s): `npx vitest run`

## 3pio Test Results
### Command: `../../build/3pio npx vitest run`
- **Status**: PARTIAL
- **Exit Code**: 1
- **Detection**: Framework detected correctly: YES
- **Output**: 183 tests total, 154 passed, 29 failed

### Issues Encountered
- Multiple "group not found" errors in 3pio event handling
- Tests that failed appear to be related to missing AI model configurations (OpenAI client not configured)
- 3pio successfully captured test results despite internal event handling errors
- Report generation worked and created markdown reports for each test file

### Specific Failures
- AI/LLM-related tests failed due to missing model configuration
- Evaluation metrics tests failed (answer relevancy, bias detection, hallucination, toxicity, etc.)
- Document extraction tests failed (keywords, questions, summary, title extraction)

### Recommendations
1. The "group not found" errors suggest a race condition or issue with test suite discovery in Vitest adapter
2. Consider improving error handling for missing group references
3. Despite internal errors, 3pio successfully generated reports - this is good resilience
4. The test failures themselves are not 3pio-related but due to missing AI service configuration