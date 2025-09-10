import { OutputParser } from '../base/OutputParser';

export class PyTestOutputParser implements OutputParser {
  parseOutputIntoTestLogs(outputContent: string): Map<string, string[]> {
    const lines = outputContent.split('\n');
    const fileOutputs = new Map<string, string[]>();
    let currentFile: string | null = null;
    let inFailureDetails = false;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      
      // Skip empty lines in the beginning
      if (!line.trim() && !currentFile) continue;
      
      // Check if we've reached the summary section
      if (this.isEndOfTestOutput(line)) {
        break;
      }
      
      // Check for test file markers (e.g., "test_math.py::TestMath::test_addition")
      const fileMatch = line.match(/^([^\s]+\.py)::/);
      if (fileMatch) {
        currentFile = fileMatch[1];
        if (!fileOutputs.has(currentFile)) {
          fileOutputs.set(currentFile, []);
        }
      }
      
      // Check for test result markers
      const resultMatch = line.match(/^([^\s]+\.py)\s+(PASSED|FAILED|SKIPPED|ERROR|XFAIL|XPASS)/);
      if (resultMatch) {
        const fileName = resultMatch[1];
        const status = resultMatch[2];
        
        if (!fileOutputs.has(fileName)) {
          fileOutputs.set(fileName, []);
        }
        
        currentFile = fileName;
        
        // Add test result line
        fileOutputs.get(fileName)!.push(`# Test Result: ${status}`);
        fileOutputs.get(fileName)!.push(line);
        fileOutputs.get(fileName)!.push('');
        continue;
      }
      
      // Check for print output during test execution
      // pytest shows stdout/stderr after test names
      if (line.includes('-- Captured stdout') || line.includes('-- Captured stderr') || 
          line.includes('-- Captured log')) {
        if (currentFile) {
          fileOutputs.get(currentFile)!.push(line);
          
          // Capture the following output lines
          let j = i + 1;
          while (j < lines.length && !lines[j].startsWith('--') && 
                 !this.isEndOfTestOutput(lines[j]) &&
                 !lines[j].match(/\.py::/)) {
            fileOutputs.get(currentFile)!.push(lines[j]);
            j++;
          }
          fileOutputs.get(currentFile)!.push('');
          i = j - 1;
        }
        continue;
      }
      
      // Check for failure details section
      if (line.startsWith('=== FAILURES ===') || line.startsWith('FAILED')) {
        inFailureDetails = true;
        continue;
      }
      
      if (inFailureDetails) {
        // Extract file from failure header (e.g., "___ test_function ___")
        const failureHeaderMatch = line.match(/^___.*___$/);
        if (failureHeaderMatch) {
          // Look for the file path in the next few lines
          let j = i + 1;
          while (j < lines.length && j < i + 5) {
            const pathMatch = lines[j].match(/^([^\s]+\.py):\d+:/);
            if (pathMatch) {
              currentFile = pathMatch[1];
              if (!fileOutputs.has(currentFile)) {
                fileOutputs.set(currentFile, []);
              }
              fileOutputs.get(currentFile)!.push('# Failure Details');
              fileOutputs.get(currentFile)!.push(line);
              break;
            }
            j++;
          }
        }
        
        // Add failure details to current file
        if (currentFile && !this.isEndOfTestOutput(line)) {
          fileOutputs.get(currentFile)!.push(line);
        }
      }
      
      // Handle print statements that appear during test execution
      if (currentFile && line.trim() && !line.startsWith('=')) {
        // Check if this looks like output from a print statement
        const isPrintOutput = !line.match(/^[^\s]+\.py/) && 
                             !line.startsWith('collected') &&
                             !line.includes('passed') &&
                             !line.includes('failed');
        
        if (isPrintOutput) {
          fileOutputs.get(currentFile)!.push(line);
        }
      }
    }
    
    return fileOutputs;
  }
  
  extractTestFileFromLine(line: string): string | null {
    // Test execution lines: "test_math.py::TestClass::test_method PASSED"
    const execMatch = line.match(/^([^\s]+\.py)::/);
    if (execMatch) {
      return execMatch[1];
    }
    
    // Result lines: "test_math.py PASSED [100%]"
    const resultMatch = line.match(/^([^\s]+\.py)\s+(PASSED|FAILED|SKIPPED|ERROR)/);
    if (resultMatch) {
      return resultMatch[1];
    }
    
    // Failure location lines: "test_math.py:42: AssertionError"
    const errorMatch = line.match(/^([^\s]+\.py):\d+:/);
    if (errorMatch) {
      return errorMatch[1];
    }
    
    // File paths in traceback
    const tracebackMatch = line.match(/File "([^"]+\.py)", line \d+/);
    if (tracebackMatch) {
      // Extract just the filename from the full path
      const fullPath = tracebackMatch[1];
      const fileName = fullPath.split('/').pop();
      return fileName || null;
    }
    
    return null;
  }
  
  isEndOfTestOutput(line: string): boolean {
    // pytest summary sections that indicate end of test output
    return line.startsWith('=') && (
      line.includes('passed') ||
      line.includes('failed') ||
      line.includes('error') ||
      line.includes('warning') ||
      line.includes('summary') ||
      line.includes('short test summary')
    );
  }
  
  formatTestHeading(line: string): string | null {
    // Format captured output sections
    if (line.includes('-- Captured stdout')) {
      return '# Captured stdout';
    }
    if (line.includes('-- Captured stderr')) {
      return '# Captured stderr';
    }
    if (line.includes('-- Captured log')) {
      return '# Captured log';
    }
    
    // Format test results
    const resultMatch = line.match(/\s+(PASSED|FAILED|SKIPPED|ERROR|XFAIL|XPASS)/);
    if (resultMatch) {
      return `# Test Result: ${resultMatch[1]}`;
    }
    
    // Format failure sections
    if (line.match(/^___.*___$/)) {
      return '# Test Failure';
    }
    
    if (line.startsWith('=== FAILURES ===')) {
      return '# Failures';
    }
    
    return null;
  }
}