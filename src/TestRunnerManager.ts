import { TestRunnerDefinition } from './runners/base/TestRunnerDefinition';
import { OutputParser } from './runners/base/OutputParser';
import { JestDefinition } from './runners/jest/JestDefinition';
import { JestOutputParser } from './runners/jest/JestOutputParser';
import { VitestDefinition } from './runners/vitest/VitestDefinition';
import { VitestOutputParser } from './runners/vitest/VitestOutputParser';
import { PyTestDefinition } from './runners/pytest/PyTestDefinition';
import { PyTestOutputParser } from './runners/pytest/PyTestOutputParser';
import { Logger } from './utils/logger';

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
  },
  pytest: {
    definition: new PyTestDefinition(),
    parser: new PyTestOutputParser()
  }
} as const;

export type TestRunnerName = keyof typeof TEST_RUNNERS;

/**
 * Manager for test runner detection and access
 */
export class TestRunnerManager {
  private static logger = Logger.create('test-runner-manager');
  
  /**
   * Detect which test runner should be used for the given command
   */
  static detect(args: string[], packageJsonContent?: string): TestRunnerName | null {
    this.logger.info('Detecting test runner', { command: args.join(' ') });
    
    // Try each test runner in priority order
    const runners: TestRunnerName[] = ['jest', 'vitest', 'pytest'];
    
    for (const runnerName of runners) {
      this.logger.debug(`Checking if command matches ${runnerName}`);
      const runner = TEST_RUNNERS[runnerName];
      if (runner.definition.matches(args, packageJsonContent)) {
        this.logger.decision('Test runner detected', runnerName, `Command matched ${runnerName} patterns`);
        return runnerName;
      }
    }
    
    this.logger.warn('No test runner detected', { args });
    return null;
  }
  
  /**
   * Get the definition for a specific test runner
   */
  static getDefinition(runner: TestRunnerName): TestRunnerDefinition {
    this.logger.debug('Getting test runner definition', { runner });
    return TEST_RUNNERS[runner].definition;
  }
  
  /**
   * Get the output parser for a specific test runner
   */
  static getParser(runner: TestRunnerName): OutputParser {
    this.logger.debug('Getting output parser', { runner });
    return TEST_RUNNERS[runner].parser;
  }
  
  /**
   * Get all available test runner names
   */
  static getAvailableRunners(): TestRunnerName[] {
    const runners = Object.keys(TEST_RUNNERS) as TestRunnerName[];
    this.logger.debug('Available test runners', { runners });
    return runners;
  }
}