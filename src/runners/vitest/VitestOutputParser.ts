import { OutputParser } from '../base/OutputParser';

export class VitestOutputParser implements OutputParser {
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

      // Check if this line indicates output from a specific test file
      // Vitest format: "stdout | math.test.js > ..." or "stderr | math.test.js > ..."
      const testFile = this.extractTestFileFromLine(line);
      if (testFile) {
        // If we're switching to a new section in the same file or different file,
        // add a blank line to separate sections
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
        // Continue adding lines to current file until we see a new file marker
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
    // Vitest format: "stdout | math.test.js > ..." or "stderr | math.test.js > ..."
    const vitestMatch = line.match(/^(stdout|stderr) \| ([^>]+\.(?:test|spec)\.[jt]sx?) > /);
    if (vitestMatch) {
      return vitestMatch[2];
    }
    
    return null;
  }

  isEndOfTestOutput(line: string): boolean {
    // Check if this is a summary line (starts with test runner symbols)
    return line.match(/^\s*(✓|✔|×|✗|↓|⚠|❯|\[PASS\]|\[FAIL\]|\[SKIP\])/) !== null;
  }

  formatTestHeading(line: string): string | null {
    const vitestMatch = line.match(/^(stdout|stderr) \| ([^>]+\.(?:test|spec)\.[jt]sx?) > (.+)/);
    if (vitestMatch) {
      const streamType = vitestMatch[1]; // 'stdout' or 'stderr'
      const content = vitestMatch[3];
      if (content.trim()) {
        return `# ${content} (${streamType})`;
      }
    }
    
    return null;
  }
}