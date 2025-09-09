# Test Runner Abstraction Strategy

This document outlines the architectural changes needed to support multiple test runners in a maintainable way by abstracting test runner-specific logic into pluggable components.

## Current Problem

The codebase has hardcoded logic for Jest and Vitest scattered throughout multiple files, making it difficult to add new test runners without modifying core components. Each new test runner requires changes in 6+ different places.

## Systems Requiring Abstraction

### 1. CLI Orchestrator - Test Runner Detection & Command Building

**Current Issues:**
- Hardcoded string matching in `detectTestRunner()` 
- Different dry run strategies in `performDryRun()`
- Different flag injection patterns in `executeMainCommand()`

**Needs Abstraction:**
```typescript
interface TestRunnerDefinition {
  name: string;
  matches: (args: string[], packageJsonContent?: string) => boolean;
  getTestFiles: (args: string[]) => Promise<string[]>;
  buildMainCommand: (args: string[], adapterPath: string) => string;
  getAdapterFileName: () => string;
  interpretExitCode: (exitCode: number) => 'success' | 'test-failure' | 'system-error';
}
```

### 2. Report Manager - Output Parsing System

**Current Issues:**
- Hardcoded Vitest pattern matching in `parseOutputIntoTestLogs()`:
  ```typescript
  const vitestMatch = line.match(/^(stdout|stderr) \| ([^>]+\.(?:test|spec)\.[jt]sx?) > /);
  ```
- Jest would have completely different output patterns
- No abstraction for different test runner output formats

**Needs Abstraction:**
```typescript
interface OutputParser {
  parseOutputIntoTestLogs(outputContent: string): Map<string, string[]>;
  extractTestFileFromLine(line: string): string | null;
  isEndOfTestOutput(line: string): boolean;
  formatTestHeading(line: string): string | null;
}
```

### 3. Test Runner Adapters - Interface Standardization

**Current Issues:**
- Different base interfaces (Jest's `Reporter` vs Vitest's `Reporter`)
- Different hook methods (`onTestStart` vs `onInit`, `onTestResult` vs `onFinished`)
- Different status determination logic
- Different lifecycle patterns

**Needs Abstraction:**
```typescript
interface TestRunnerAdapter {
  initialize(): void;
  startCapture(): void;
  stopCapture(): void;
  handleTestFileStart(filePath: string): void;
  handleTestFileResult(filePath: string, status: 'PASS' | 'FAIL' | 'SKIP'): void;
  cleanup(): void;
}

// Bridge pattern to adapt specific test runner interfaces
abstract class AdapterBridge<T> implements TestRunnerAdapter {
  constructor(protected nativeAdapter: T) {}
  abstract mapNativeHooks(): void;
  abstract extractTestStatus(result: any): 'PASS' | 'FAIL' | 'SKIP';
}
```

## Proposed Architecture Solution

### Static Strategy Pattern
```typescript
// Explicit, compile-time known test runners
const TEST_RUNNERS = {
  jest: {
    definition: new JestDefinition(),
    parser: new JestOutputParser()
  },
  vitest: {
    definition: new VitestDefinition(),
    parser: new VitestOutputParser()
  }
} as const;

class TestRunnerManager {
  static detect(args: string[], packageJsonContent?: string): keyof typeof TEST_RUNNERS | null;
  static getDefinition(runner: keyof typeof TEST_RUNNERS): TestRunnerDefinition;
  static getParser(runner: keyof typeof TEST_RUNNERS): OutputParser;
}
```

## Example Implementations

### Jest Definition
```typescript
class JestDefinition implements TestRunnerDefinition {
  name = 'jest';
  
  matches(args: string[], packageJsonContent?: string): boolean {
    const command = args[0];
    
    // Direct detection from command
    if (command === 'jest') return true;
    if ((command === 'npx' || command === 'yarn' || command === 'pnpm') && args[1] === 'jest') {
      return true;
    }
    
    // npm script detection
    if ((command === 'npm' && (args[1] === 'test' || args[1] === 'run')) && packageJsonContent) {
      try {
        const packageJson = JSON.parse(packageJsonContent);
        
        // Check test script
        const testScript = packageJson.scripts?.test;
        if (testScript?.includes('jest')) return true;
        
        // Check dependencies
        const deps = { ...packageJson.dependencies, ...packageJson.devDependencies };
        return !!deps.jest;
      } catch {
        return false;
      }
    }
    
    return false;
  }
  
  async getTestFiles(args: string[]): Promise<string[]> {
    // Check if specific test files provided
    const testFileExtensions = ['.test.js', '.test.ts', '.spec.js', '.spec.ts'];
    const providedFiles = args.filter(arg => 
      !arg.startsWith('-') && testFileExtensions.some(ext => arg.includes(ext))
    );
    
    if (providedFiles.length > 0) {
      return providedFiles;
    }
    
    try {
      // Jest dry run
      const dryRunCommand = args.join(' ') + ' --listTests';
      const result = await $`sh -c ${dryRunCommand}`;
      return JSON.parse(result.stdout);
    } catch {
      return [];
    }
  }
  
  buildMainCommand(args: string[], adapterPath: string): string {
    const hasReporters = args.some(arg => arg.includes('--reporters'));
    if (hasReporters) {
      return `${args.join(' ')} ${adapterPath}`;
    } else {
      return `${args.join(' ')} --reporters default --reporters ${adapterPath}`;
    }
  }
  
  getAdapterFileName(): string {
    return 'jest.js';
  }
  
  interpretExitCode(exitCode: number): 'success' | 'test-failure' | 'system-error' {
    switch (exitCode) {
      case 0: return 'success';
      case 1: return 'test-failure';
      default: return 'system-error';
    }
  }
}
```

### Vitest Definition
```typescript
class VitestDefinition implements TestRunnerDefinition {
  name = 'vitest';
  
  matches(args: string[], packageJsonContent?: string): boolean {
    const command = args[0];
    
    // Direct detection
    if (command === 'vitest') return true;
    if ((command === 'npx' || command === 'yarn' || command === 'pnpm') && args[1] === 'vitest') {
      return true;
    }
    
    // npm script detection
    if ((command === 'npm' && (args[1] === 'test' || args[1] === 'run')) && packageJsonContent) {
      try {
        const packageJson = JSON.parse(packageJsonContent);
        
        const testScript = packageJson.scripts?.test;
        if (testScript?.includes('vitest')) return true;
        
        const deps = { ...packageJson.dependencies, ...packageJson.devDependencies };
        return !!deps.vitest;
      } catch {
        return false;
      }
    }
    
    return false;
  }
  
  async getTestFiles(args: string[]): Promise<string[]> {
    // Vitest list doesn't work reliably, so extract from args or return empty
    const testFileExtensions = ['.test.js', '.test.ts', '.test.mjs', '.spec.js', '.spec.ts'];
    const providedFiles = args.filter(arg =>
      !arg.startsWith('-') && testFileExtensions.some(ext => arg.includes(ext))
    );
    
    return providedFiles;
  }
  
  buildMainCommand(args: string[], adapterPath: string): string {
    const hasReporter = args.some(arg => arg.includes('--reporter'));
    if (hasReporter) {
      return `${args.join(' ')} --reporter ${adapterPath}`;
    } else {
      return `${args.join(' ')} --reporter default --reporter ${adapterPath}`;
    }
  }
  
  getAdapterFileName(): string {
    return 'vitest.js';
  }
  
  interpretExitCode(exitCode: number): 'success' | 'test-failure' | 'system-error' {
    switch (exitCode) {
      case 0: return 'success';
      case 1: return 'test-failure';
      default: return 'system-error';
    }
  }
}
```

## Implementation Plan

### Files Requiring Major Changes

1. **`src/cli.ts`** - Replace hardcoded detection/command building with TestRunnerManager
2. **`src/ReportManager.ts`** - Replace hardcoded parsing with pluggable parsers
3. **`src/adapters/`** - Create base classes and bridge patterns
4. **`src/runners/`** - **New directory** for test runner definitions
5. **`build.js`** - Update build process for new structure

### New Directory Structure
```
src/
├── runners/
│   ├── jest/
│   │   ├── JestDefinition.ts
│   │   └── JestOutputParser.ts
│   ├── vitest/
│   │   ├── VitestDefinition.ts
│   │   └── VitestOutputParser.ts
│   └── base/
│       ├── TestRunnerDefinition.ts
│       └── OutputParser.ts
├── adapters/
│   ├── base/
│   │   ├── TestRunnerAdapter.ts
│   │   └── AdapterBridge.ts
│   ├── JestAdapterBridge.ts
│   └── VitestAdapterBridge.ts
└── TestRunnerManager.ts
```

## Usage in CLI Orchestrator

```typescript
class CLIOrchestrator {
  private async detectTestRunner(args: string[]): Promise<TestRunnerDefinition> {
    let packageJsonContent: string | undefined;
    
    // Only read package.json if we might need it (npm commands)
    if (args[0] === 'npm' && (args[1] === 'test' || args[1] === 'run')) {
      try {
        packageJsonContent = await fs.readFile('package.json', 'utf8');
      } catch {
        // package.json not found - continue without it
      }
    }
    
    // Try each test runner
    for (const runner of Object.values(TEST_RUNNERS)) {
      if (runner.definition.matches(args, packageJsonContent)) {
        return runner.definition;
      }
    }
    
    throw new Error('Could not detect test runner');
  }

  private async run(commandArgs: string[]): Promise<void> {
    const runner = await this.detectTestRunner(commandArgs);
    
    // Clean single method call
    const testFiles = await runner.getTestFiles(commandArgs);
    
    await this.initialize(testCommand, testFiles);
    
    // Clean command building
    const adapterPath = path.join(__dirname, runner.getAdapterFileName());
    const modifiedCommand = runner.buildMainCommand(commandArgs, adapterPath);
    
    const exitCode = await this.executeCommand(modifiedCommand);
    const exitType = runner.interpretExitCode(exitCode);
    // Handle exit type as needed
  }
}
```

## Benefits

### Maintainability
- **Single Responsibility**: Each component handles one specific aspect of test runner integration
- **Open/Closed Principle**: Adding new test runners doesn't require modifying existing code
- **Clear Interfaces**: Well-defined contracts between components

### Extensibility
- **Explicit Architecture**: New test runners are explicitly added to the TEST_RUNNERS constant
- **Minimal Integration**: Adding new test runners requires only implementing interfaces and adding to the constant
- **Clean Separation**: Test runner logic is isolated from core 3pio logic

### Testing
- **Isolated Testing**: Each parser/adapter can be unit tested independently  
- **Mock-Friendly**: Interfaces enable easy mocking for testing
- **Regression Prevention**: Changes to one test runner won't affect others

## Migration Strategy

1. **Phase 1**: Extract interfaces and create base classes
2. **Phase 2**: Implement Jest/Vitest using new architecture 
3. **Phase 3**: Replace hardcoded logic with static TEST_RUNNERS pattern
4. **Phase 4**: Clean up old hardcoded implementations
5. **Phase 5**: Add comprehensive tests for new architecture

This approach ensures a smooth transition while maintaining backward compatibility during the migration process.