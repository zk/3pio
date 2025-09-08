/**
 * Interface for test runner definition - defines how to detect, configure, and execute a test runner
 */
export interface TestRunnerDefinition {
  /** The name of the test runner */
  name: string;
  
  /** 
   * Check if the given command and environment matches this test runner 
   */
  matches(args: string[], packageJsonContent?: string): boolean;
  
  /** 
   * Get the list of test files that will be run by this command 
   */
  getTestFiles(args: string[]): Promise<string[]>;
  
  /** 
   * Build the main command with the adapter injected 
   * Returns array of command arguments for proper shell execution
   */
  buildMainCommand(args: string[], adapterPath: string): string[];
  
  /** 
   * Get the adapter filename for this test runner 
   */
  getAdapterFileName(): string;
  
  /** 
   * Interpret the exit code from the test runner 
   */
  interpretExitCode(exitCode: number): 'success' | 'test-failure' | 'system-error';
}