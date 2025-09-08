# Documentation Inconsistencies Report

This document identifies inconsistencies between the project documentation and the actual implementation in the codebase.

## Summary

After systematically comparing each documentation file with the source code, I found several significant inconsistencies ranging from outdated API descriptions to missing implementation details. The documentation generally follows the high-level architecture but differs in many implementation specifics.

## Critical Inconsistencies

### 1. Report Manager (docs/report-manager.md)

**Major Inconsistency: Internal State Structure**
- **Documentation**: Shows `RunState` interface with `testFiles: Map<string, { status, logPath }>`
- **Implementation**: Uses `TestRunState` interface with `testFiles: Array<{ status, file, logFile }>`
- **Impact**: Complete mismatch of data structures

**Missing Implementation: Individual Log Files**
- **Documentation**: Claims individual `.log` files are created per test file
- **Implementation**: Uses unified `output.log` with post-processing to create individual logs in `parseOutputIntoTestLogs()`
- **Impact**: Different approach than documented

**API Method Mismatch**
- **Documentation**: `handleEvent(event: IPCEvent): void`
- **Implementation**: `async handleEvent(event: IPCEvent): Promise<void>`
- **Impact**: Different async contract

**Missing Method**
- **Documentation**: `initialize(runId: string, testFiles: string[], args: string): void`
- **Implementation**: `constructor(runId: string, testCommand: string)` + `async initialize(testFiles: string[]): Promise<void>`
- **Impact**: Different initialization pattern

### 2. IPC Manager (docs/ipc-manager.md)

**API Structure Mismatch**
- **Documentation**: Shows functions `createWriter()` and `createReader()`
- **Implementation**: Uses class-based approach with instance methods `writeEvent()` and `watchEvents()`
- **Impact**: Completely different API design

**Missing Static Method**
- **Documentation**: No mention of static helper methods
- **Implementation**: Includes `IPCManager.sendEvent()` static method for adapters
- **Impact**: Undocumented critical functionality

**Reader Implementation Differs**
- **Documentation**: Returns `{ close: () => void }`
- **Implementation**: Uses instance method `stopWatching()` and class-based lifecycle
- **Impact**: Different cleanup approach

### 3. CLI Orchestrator (docs/cli-orchestrator.md)

**Missing Method Details**
- **Documentation**: Mentions `reportManager.initialize(testFiles)` 
- **Implementation**: Uses `await this.reportManager.initialize(testFiles)` with different signature
- **Impact**: Async/sync mismatch

**Command Modification Logic**
- **Documentation**: Generic description of flag injection
- **Implementation**: Specific logic using `path.join(__dirname, 'jest.js')` for adapter paths
- **Impact**: Missing concrete implementation details

**Environment Variable Handling**
- **Documentation**: Shows `process.env.THREEPIO_IPC_PATH = "/path/..."`
- **Implementation**: Sets `process.env.THREEPIO_IPC_PATH = this.ipcPath` then passes explicitly in zx env
- **Impact**: More complex env var propagation than documented

### 4. Test Runner Adapters (docs/test-runner-adapter.md)

**Jest Adapter Implementation**
- **Documentation**: Shows `class JestAdapter extends DefaultReporter`
- **Implementation**: `class ThreePioJestReporter implements Reporter`
- **Impact**: Different inheritance pattern

**Vitest Adapter Hooks**
- **Documentation**: Lists specific hooks: `onRunStart`, `onTestFileStart`, `onTestFileResult`, `onRunComplete`
- **Implementation**: Uses `onInit`, `onTestFileStart`, `onTestFileResult`, `onFinished`
- **Impact**: Some hook names differ

**Stream Tapping Implementation**
- **Documentation**: Describes patching during `onTestFileStart/Result`
- **Implementation**: Vitest adapter starts capture in `onInit()` and maintains it throughout
- **Impact**: Different capture lifecycle

## Minor Inconsistencies

### 5. System Architecture (docs/system-architecture.md)

**File Paths**
- **Documentation**: Shows paths like `/.3pio/runs/RUN_ID/`
- **Implementation**: Uses `.3pio/runs/${runId}/` relative to `process.cwd()`
- **Impact**: Minor - leading slash vs relative path

**Report Structure**  
- **Documentation**: Shows individual log files created during run
- **Implementation**: Creates unified `output.log` then post-processes into individual files
- **Impact**: Different timing of file creation

### 6. Project Plan (docs/project-plan.md)

**Package Name**
- **Documentation**: Claims package will be `@heyzk/3pio`
- **Implementation**: Package.json shows different structure
- **Impact**: Publishing/distribution inconsistency

**Command Injection Examples**
- **Documentation**: Shows `@heyzk/3pio/jest` and `@heyzk/3pio/vitest`
- **Implementation**: Uses relative paths resolved from `__dirname`
- **Impact**: Different module resolution approach

## Recommendations

1. **Update Report Manager Documentation**: Align with actual `TestRunState` interface and async methods
2. **Rewrite IPC Manager Documentation**: Document the class-based API and static `sendEvent()` method  
3. **Clarify Adapter Documentation**: Update hook names and inheritance patterns to match implementation
4. **Add Implementation Details**: Document the unified output.log approach and post-processing strategy
5. **Update Examples**: Replace conceptual examples with actual code patterns used in implementation
6. **Synchronize Types**: Ensure all TypeScript interfaces mentioned in docs match those in `types/events.ts`

## Files Requiring Updates

- `docs/report-manager.md` - Complete rewrite of state management and API sections
- `docs/ipc-manager.md` - Complete rewrite of API design section  
- `docs/test-runner-adapter.md` - Update adapter class names and hook methods
- `docs/cli-orchestrator.md` - Add async/await patterns and environment handling details
- `docs/system-architecture.md` - Update data flow to reflect unified logging approach
- `docs/project-plan.md` - Update package structure and command injection examples