/**
 * Standardized interface for test runner adapters
 */
export interface TestRunnerAdapter {
  /** Initialize the adapter */
  initialize(): void;
  
  /** Start capturing output */
  startCapture(): void;
  
  /** Stop capturing output */
  stopCapture(): void;
  
  /** Handle the start of a test file */
  handleTestFileStart(filePath: string): void;
  
  /** Handle the result of a test file */
  handleTestFileResult(filePath: string, status: 'PASS' | 'FAIL' | 'SKIP'): void;
  
  /** Cleanup when test run completes */
  cleanup(): void;
}