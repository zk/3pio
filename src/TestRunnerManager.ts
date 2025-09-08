import { TestRunnerDefinition } from './runners/base/TestRunnerDefinition';
import { OutputParser } from './runners/base/OutputParser';
import { JestDefinition } from './runners/jest/JestDefinition';
import { JestOutputParser } from './runners/jest/JestOutputParser';
import { VitestDefinition } from './runners/vitest/VitestDefinition';
import { VitestOutputParser } from './runners/vitest/VitestOutputParser';

/**
 * Static strategy pattern for test runner management
 * Explicit, compile-time known test runners
 */
export const TEST_RUNNERS = {
  jest: {
    definition: new JestDefinition(),
    parser: new JestOutputParser()
  },
  vitest: {
    definition: new VitestDefinition(),
    parser: new VitestOutputParser()
  }
} as const;

export type TestRunnerName = keyof typeof TEST_RUNNERS;

/**
 * Manager for test runner detection and access
 */
export class TestRunnerManager {
  /**
   * Detect which test runner should be used for the given command
   */
  static detect(args: string[], packageJsonContent?: string): TestRunnerName | null {
    // Try each test runner in priority order
    const runners: TestRunnerName[] = ['jest', 'vitest'];
    
    for (const runnerName of runners) {
      const runner = TEST_RUNNERS[runnerName];
      if (runner.definition.matches(args, packageJsonContent)) {
        return runnerName;
      }
    }
    
    return null;
  }
  
  /**
   * Get the definition for a specific test runner
   */
  static getDefinition(runner: TestRunnerName): TestRunnerDefinition {
    return TEST_RUNNERS[runner].definition;
  }
  
  /**
   * Get the output parser for a specific test runner
   */
  static getParser(runner: TestRunnerName): OutputParser {
    return TEST_RUNNERS[runner].parser;
  }
  
  /**
   * Get all available test runner names
   */
  static getAvailableRunners(): TestRunnerName[] {
    return Object.keys(TEST_RUNNERS) as TestRunnerName[];
  }
}