import { OutputParser } from '../base/OutputParser';

export class JestOutputParser implements OutputParser {
  parseOutputIntoTestLogs(outputContent: string): Map<string, string[]> {
    const lines = outputContent.split('\n');
    const fileOutputs = new Map<string, string[]>();
    let inHeader = true;

    let i = 0;
    while (i < lines.length) {
      const line = lines[i];
      
      // Skip header lines (first 5 lines that start with #)
      if (inHeader) {
        if (line.startsWith('# ---')) {
          inHeader = false;
        }
        i++;
        continue;
      }

      // Check for console.* output blocks
      if (line.match(/^\s*console\.(log|error|warn|info|debug)/)) {
        // Collect the console block
        const consoleBlock: string[] = [line];
        let fileName: string | null = null;
        
        // Look ahead for the stack trace to get the filename
        let j = i + 1;
        while (j < lines.length && j < i + 10) {
          consoleBlock.push(lines[j]);
          
          // Look for the stack trace line with filename
          const stackMatch = lines[j].match(/at (?:Object\.|Test\.|.*) \(([^:]+\.(test|spec)\.[jt]sx?):\d+:\d+\)/);
          if (stackMatch) {
            fileName = stackMatch[1];
            break;
          }
          
          // Also check for simpler format: "at filename:line:col"
          const simpleMatch = lines[j].match(/at ([^:]+\.(test|spec)\.[jt]sx?):\d+:\d+/);
          if (simpleMatch) {
            fileName = simpleMatch[1];
            break;
          }
          
          // Stop if we hit another console.* or a PASS/FAIL line
          if (lines[j].match(/^\s*console\./) || lines[j].match(/^(PASS|FAIL)/)) {
            j--;
            break;
          }
          
          j++;
        }
        
        // If we found a filename, add the block to that file's output
        if (fileName) {
          if (!fileOutputs.has(fileName)) {
            fileOutputs.set(fileName, []);
          }
          
          // Add the console block
          fileOutputs.get(fileName)!.push(...consoleBlock);
          fileOutputs.get(fileName)!.push(''); // Add blank line after block
        }
        
        // Move past the processed lines
        i = j + 1;
      }
      // Check for PASS/FAIL lines
      else if (line.match(/^(PASS|FAIL|SKIP)\s+\.\//)) {
        const match = line.match(/^(PASS|FAIL|SKIP)\s+\.\/([^\s]+)/);
        if (match) {
          const status = match[1];
          const fileName = match[2];
          
          if (!fileOutputs.has(fileName)) {
            fileOutputs.set(fileName, []);
          }
          
          // Add test result as a section
          fileOutputs.get(fileName)!.push(`# Test Result: ${status}`);
          fileOutputs.get(fileName)!.push(line);
          
          // Collect any following lines that belong to this test result (like test names)
          // BUT stop if we hit console output (which belongs to the next test)
          let j = i + 1;
          while (j < lines.length && 
                 !lines[j].match(/^(PASS|FAIL|SKIP|Test Suites|Tests:)/) &&
                 !lines[j].match(/^\s*console\./)) {
            if (lines[j].trim()) {
              fileOutputs.get(fileName)!.push(lines[j]);
            }
            j++;
          }
          fileOutputs.get(fileName)!.push(''); // Add blank line after
          
          i = j;
        } else {
          i++;
        }
      }
      else {
        i++;
      }
    }

    return fileOutputs;
  }

  extractTestFileFromLine(line: string): string | null {
    // Look for stack trace patterns with file names
    const stackMatch = line.match(/at (?:Object\.|Test\.|.*) \(([^:]+\.(test|spec)\.[jt]sx?):\d+:\d+\)/);
    if (stackMatch) {
      return stackMatch[1];
    }
    
    // Simpler stack trace format
    const simpleMatch = line.match(/at ([^:]+\.(test|spec)\.[jt]sx?):\d+:\d+/);
    if (simpleMatch) {
      return simpleMatch[1];
    }
    
    // PASS/FAIL lines with file paths
    const resultMatch = line.match(/^(PASS|FAIL|SKIP)\s+\.\/([^\s]+)/);
    if (resultMatch) {
      return resultMatch[2];
    }
    
    return null;
  }

  isEndOfTestOutput(line: string): boolean {
    // Jest summary patterns that indicate we've moved past individual test output
    return line.match(/^(Test Suites:|Tests:|Snapshots:|Time:|Ran all test suites)/i) !== null;
  }

  formatTestHeading(line: string): string | null {
    // Format console output blocks
    const consoleMatch = line.match(/^\s*console\.([a-z]+)/);
    if (consoleMatch) {
      return `# Console ${consoleMatch[1]}`;
    }
    
    // Format test results
    const resultMatch = line.match(/^(PASS|FAIL|SKIP)\s+\.\//);
    if (resultMatch) {
      return `# Test Result: ${resultMatch[1]}`;
    }
    
    return null;
  }
}