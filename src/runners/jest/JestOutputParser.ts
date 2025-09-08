import { OutputParser } from '../base/OutputParser';

export class JestOutputParser implements OutputParser {
  parseOutputIntoTestLogs(outputContent: string): Map<string, string[]> {
    const lines = outputContent.split('\n');
    const fileOutputs = new Map<string, string[]>();
    let currentFile: string | null = null;
    let inHeader = true;

    for (const line of lines) {
      // Skip header lines (first 5 lines that start with #)
      if (inHeader) {
        if (line.startsWith('# ---')) {
          inHeader = false;
        }
        continue;
      }

      // Extract test file from Jest output patterns
      const testFile = this.extractTestFileFromLine(line);
      if (testFile) {
        // If we're switching files, add a blank line to separate sections
        if (currentFile && fileOutputs.has(currentFile) && fileOutputs.get(currentFile)!.length > 0) {
          const lastLine = fileOutputs.get(currentFile)![fileOutputs.get(currentFile)!.length - 1];
          if (lastLine.trim() !== '') {
            fileOutputs.get(currentFile)!.push('');
          }
        }
        
        currentFile = testFile;
        if (!fileOutputs.has(currentFile)) {
          fileOutputs.set(currentFile, []);
        }
        
        // Format as heading
        const heading = this.formatTestHeading(line);
        if (heading) {
          fileOutputs.get(currentFile)!.push(heading);
        }
      } else if (currentFile && line.trim()) {
        // Continue adding lines to current file until we see summary or end
        if (this.isEndOfTestOutput(line)) {
          // Add a blank line before ending the current section
          fileOutputs.get(currentFile)!.push('');
          currentFile = null; // Reset when we hit summary lines
        } else {
          fileOutputs.get(currentFile)!.push(line);
        }
      } else if (currentFile && !line.trim()) {
        // Handle empty lines within a section
        fileOutputs.get(currentFile)!.push(line);
      }
    }

    return fileOutputs;
  }

  extractTestFileFromLine(line: string): string | null {
    // Jest output patterns - these may vary depending on Jest version and configuration
    // Common patterns:
    // - PASS/FAIL prefix with file path
    // - Console log output with file indicators
    
    // Pattern for test file results: "PASS src/math.test.js"
    const resultMatch = line.match(/^(PASS|FAIL|SKIP)\s+([^\s]+\.(?:test|spec)\.[jt]sx?)(\s|$)/);
    if (resultMatch) {
      return resultMatch[2];
    }
    
    // Pattern for console output during test execution
    // Jest might use different patterns - this is a basic implementation
    const consoleMatch = line.match(/^\s*console\.[a-z]+\s+([^\s]+\.(?:test|spec)\.[jt]sx?)/);
    if (consoleMatch) {
      return consoleMatch[1];
    }
    
    return null;
  }

  isEndOfTestOutput(line: string): boolean {
    // Jest summary patterns
    return line.match(/^\s*(Test Suites|Tests:|Snapshots:|Time:|Ran all test suites|✓|×|✗)/i) !== null;
  }

  formatTestHeading(line: string): string | null {
    // Extract the meaningful part for heading
    const resultMatch = line.match(/^(PASS|FAIL|SKIP)\s+(.+)/);
    if (resultMatch) {
      return `# ${resultMatch[2]} (${resultMatch[1].toLowerCase()})`;
    }
    
    const consoleMatch = line.match(/^\s*console\.[a-z]+\s+(.+)/);
    if (consoleMatch) {
      return `# ${consoleMatch[1]} (console output)`;
    }
    
    return null;
  }
}