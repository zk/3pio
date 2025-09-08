/**
 * Interface for parsing test runner output into structured logs
 */
export interface OutputParser {
  /**
   * Parse the output content and return a map of test file paths to their output lines
   */
  parseOutputIntoTestLogs(outputContent: string): Map<string, string[]>;
  
  /**
   * Extract the test file path from a single line of output
   * Returns null if the line doesn't contain test file information
   */
  extractTestFileFromLine(line: string): string | null;
  
  /**
   * Check if this line indicates the end of test output (start of summary)
   */
  isEndOfTestOutput(line: string): boolean;
  
  /**
   * Format a test heading from output line, returns null if not a heading
   */
  formatTestHeading(line: string): string | null;
}