import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { execSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';

describe('Test Case Reporting Integration', () => {
  const testProjectDir = path.join(__dirname, '../../scratch/test-reporting-project');
  const threePioCmd = path.join(__dirname, '../../dist/cli.js');
  
  beforeEach(() => {
    // Create a test project with sample test files
    fs.mkdirSync(testProjectDir, { recursive: true });
    
    // Create package.json
    const packageJson = {
      name: "test-reporting-project",
      version: "1.0.0",
      type: "module",
      scripts: {
        test: "vitest run"
      },
      devDependencies: {
        vitest: "^1.0.0"
      }
    };
    fs.writeFileSync(
      path.join(testProjectDir, 'package.json'),
      JSON.stringify(packageJson, null, 2)
    );
    
    // Create vitest.config.js
    const vitestConfig = `
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    reporters: [],
    passWithNoTests: false
  }
});
`;
    fs.writeFileSync(path.join(testProjectDir, 'vitest.config.js'), vitestConfig);
    
    // Create math.test.js with multiple test cases
    const mathTest = `
import { describe, it, expect } from 'vitest';

describe('Math operations', () => {
  it('should add numbers correctly', () => {
    console.log('Testing addition...');
    expect(2 + 2).toBe(4);
    console.log('Addition tests passed!');
  });
  
  it('should multiply numbers correctly', () => {
    console.log('Testing multiplication...');
    expect(3 * 4).toBe(12);
    console.log('Multiplication tests passed!');
  });
  
  it('should handle division', () => {
    console.log('Testing division...');
    expect(10 / 2).toBe(5);
    console.log('Division tests passed!');
  });
});
`;
    fs.writeFileSync(path.join(testProjectDir, 'math.test.js'), mathTest);
    
    // Create string.test.js with pass/fail/skip cases
    const stringTest = `
import { describe, it, expect } from 'vitest';

describe('String operations', () => {
  it('should concatenate strings', () => {
    console.log('Testing string concatenation...');
    expect('hello' + ' ' + 'world').toBe('hello world');
    console.log('String concatenation passed!');
  });
  
  it('should fail this test', () => {
    console.log('This test is expected to fail');
    console.error('Error: Intentional failure for testing');
    expect('foo').toBe('bar'); // This will fail
  });
  
  it.skip('should skip this test', () => {
    console.log('This should not run');
    expect(true).toBe(true);
  });
  
  it('should convert to uppercase', () => {
    console.log('Testing uppercase conversion...');
    expect('hello'.toUpperCase()).toBe('HELLO');
    console.log('Uppercase test passed!');
  });
});
`;
    fs.writeFileSync(path.join(testProjectDir, 'string.test.js'), stringTest);
  });
  
  afterEach(() => {
    // Clean up test project
    if (fs.existsSync(testProjectDir)) {
      fs.rmSync(testProjectDir, { recursive: true, force: true });
    }
  });
  
  describe('Report File Generation', () => {
    it('should create all expected report files', () => {
      // Install dependencies first
      execSync(`cd ${testProjectDir} && npm install --no-save vitest@latest`, {
        stdio: 'ignore'
      });
      
      // Run 3pio
      try {
        execSync(`cd ${testProjectDir} && ${threePioCmd} npx vitest run math.test.js string.test.js`, {
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected, ignore non-zero exit code
      }
      
      // Find the latest run directory
      const runsDir = path.join(testProjectDir, '.3pio', 'runs');
      const runDirs = fs.readdirSync(runsDir);
      expect(runDirs.length).toBeGreaterThan(0);
      
      const latestRun = runDirs.sort().pop()!;
      const runDir = path.join(runsDir, latestRun);
      
      // Check for test-run.md
      const testRunPath = path.join(runDir, 'test-run.md');
      expect(fs.existsSync(testRunPath)).toBe(true);
      
      // Check for output.log
      const outputLogPath = path.join(runDir, 'output.log');
      expect(fs.existsSync(outputLogPath)).toBe(true);
      
      // Check for individual log files
      const logsDir = path.join(runDir, 'logs');
      expect(fs.existsSync(logsDir)).toBe(true);
      
      const mathLogPath = path.join(logsDir, 'math.test.js.log');
      expect(fs.existsSync(mathLogPath)).toBe(true);
      
      const stringLogPath = path.join(logsDir, 'string.test.js.log');
      expect(fs.existsSync(stringLogPath)).toBe(true);
    });
  });
  
  describe('test-run.md Content', () => {
    it('should have correct structure and test case details', () => {
      // Install dependencies first
      execSync(`cd ${testProjectDir} && npm install --no-save vitest@latest`, {
        stdio: 'ignore'
      });
      
      // Run 3pio
      try {
        execSync(`cd ${testProjectDir} && ${threePioCmd} npx vitest run math.test.js string.test.js`, {
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected
      }
      
      // Find and read test-run.md
      const runsDir = path.join(testProjectDir, '.3pio', 'runs');
      const latestRun = fs.readdirSync(runsDir).sort().pop()!;
      const testRunPath = path.join(runsDir, latestRun, 'test-run.md');
      const content = fs.readFileSync(testRunPath, 'utf8');
      
      // Check main sections
      expect(content).toContain('# 3pio Test Run');
      expect(content).toContain('## Summary');
      expect(content).toContain('- Total Files: 2');
      expect(content).toContain('- Files Completed: 2');
      expect(content).toContain('- Files Passed: 1');
      expect(content).toContain('- Files Failed: 1');
      
      // Check math.test.js section
      expect(content).toContain('## math.test.js');
      expect(content).toContain('Status: **PASS**');
      expect(content).toContain('### Math operations');
      expect(content).toContain('✓ should add numbers correctly');
      expect(content).toContain('✓ should multiply numbers correctly');
      expect(content).toContain('✓ should handle division');
      
      // Check string.test.js section
      expect(content).toContain('## string.test.js');
      expect(content).toContain('Status: **FAIL**');
      expect(content).toContain('### String operations');
      expect(content).toContain('✓ should concatenate strings');
      expect(content).toContain('✕ should fail this test');
      expect(content).toContain('○ should skip this test');
      expect(content).toContain('✓ should convert to uppercase');
      
      // Check for error message
      expect(content).toMatch(/expected 'foo' to be 'bar'/);
      
      // Check for duration format (ms or s)
      expect(content).toMatch(/\(\d+(\.\d+)?\s*(ms|s)\)/);
      
      // Check for log file links
      expect(content).toContain('[Log](./logs/math.test.js.log)');
      expect(content).toContain('[Log](./logs/string.test.js.log)');
      expect(content).toContain('[output.log](./output.log)');
    });
  });
  
  describe('output.log Content', () => {
    it('should have correct header and capture console output', () => {
      // Install dependencies first
      execSync(`cd ${testProjectDir} && npm install --no-save vitest@latest`, {
        stdio: 'ignore'
      });
      
      // Run 3pio
      try {
        execSync(`cd ${testProjectDir} && ${threePioCmd} npx vitest run math.test.js string.test.js`, {
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected
      }
      
      // Find and read output.log
      const runsDir = path.join(testProjectDir, '.3pio', 'runs');
      const latestRun = fs.readdirSync(runsDir).sort().pop()!;
      const outputLogPath = path.join(runsDir, latestRun, 'output.log');
      const content = fs.readFileSync(outputLogPath, 'utf8');
      
      // Check header
      expect(content).toContain('# 3pio Test Output Log');
      expect(content).toContain('# Timestamp:');
      expect(content).toContain('# Command: npx vitest run math.test.js string.test.js');
      expect(content).toContain('# This file contains all stdout/stderr output from the test run.');
      expect(content).toContain('# ---');
      
      // Check for console output from tests
      expect(content).toContain('Testing addition...');
      expect(content).toContain('Addition tests passed!');
      expect(content).toContain('Testing multiplication...');
      expect(content).toContain('Testing string concatenation...');
      expect(content).toContain('This test is expected to fail');
      expect(content).toContain('Error: Intentional failure for testing');
      expect(content).toContain('Testing uppercase conversion...');
      
      // Should NOT contain output from skipped test
      expect(content).not.toContain('This should not run');
    });
  });
  
  describe('Individual Log Files Content', () => {
    it('should have correct headers in math.test.js.log', () => {
      // Install dependencies first
      execSync(`cd ${testProjectDir} && npm install --no-save vitest@latest`, {
        stdio: 'ignore'
      });
      
      // Run 3pio
      try {
        execSync(`cd ${testProjectDir} && ${threePioCmd} npx vitest run math.test.js string.test.js`, {
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected
      }
      
      // Find and read math.test.js.log
      const runsDir = path.join(testProjectDir, '.3pio', 'runs');
      const latestRun = fs.readdirSync(runsDir).sort().pop()!;
      const mathLogPath = path.join(runsDir, latestRun, 'logs', 'math.test.js.log');
      const content = fs.readFileSync(mathLogPath, 'utf8');
      
      // Check header
      expect(content).toContain('# File: math.test.js');
      expect(content).toContain('# Timestamp:');
      expect(content).toContain('# This file contains all stdout/stderr output from the test file execution.');
      expect(content).toContain('# ---');
      
      // Note: Individual test file logs may be empty for Vitest due to worker process isolation
      // The presence of the file with proper header is what we're validating
    });
    
    it('should have correct headers in string.test.js.log', () => {
      // Install dependencies first
      execSync(`cd ${testProjectDir} && npm install --no-save vitest@latest`, {
        stdio: 'ignore'
      });
      
      // Run 3pio
      try {
        execSync(`cd ${testProjectDir} && ${threePioCmd} npx vitest run math.test.js string.test.js`, {
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected
      }
      
      // Find and read string.test.js.log
      const runsDir = path.join(testProjectDir, '.3pio', 'runs');
      const latestRun = fs.readdirSync(runsDir).sort().pop()!;
      const stringLogPath = path.join(runsDir, latestRun, 'logs', 'string.test.js.log');
      const content = fs.readFileSync(stringLogPath, 'utf8');
      
      // Check header
      expect(content).toContain('# File: string.test.js');
      expect(content).toContain('# Timestamp:');
      expect(content).toContain('# This file contains all stdout/stderr output from the test file execution.');
      expect(content).toContain('# ---');
    });
    
    it('should organize output by test case when available', () => {
      // This test validates the test case demarcation feature
      // When console output is captured at the adapter level (not common in Vitest due to workers)
      // it should be organized under test case headers
      
      // Install dependencies first
      execSync(`cd ${testProjectDir} && npm install --no-save vitest@latest`, {
        stdio: 'ignore'
      });
      
      // Run 3pio with a configuration that might capture output
      try {
        execSync(`cd ${testProjectDir} && ${threePioCmd} npx vitest run math.test.js string.test.js`, {
          stdio: 'ignore',
          env: { ...process.env, THREEPIO_DEBUG: '1' }
        });
      } catch (e) {
        // Test failures are expected
      }
      
      const runsDir = path.join(testProjectDir, '.3pio', 'runs');
      const latestRun = fs.readdirSync(runsDir).sort().pop()!;
      const mathLogPath = path.join(runsDir, latestRun, 'logs', 'math.test.js.log');
      const content = fs.readFileSync(mathLogPath, 'utf8');
      
      // If output is captured, it should have test case sections
      // If not, it should indicate no output was captured
      if (content.includes('## Test Case Output')) {
        expect(content).toContain('### Math operations › should add numbers correctly');
        expect(content).toContain('### Math operations › should multiply numbers correctly');
        expect(content).toContain('### Math operations › should handle division');
      } else {
        expect(content).toMatch(/No output captured|no output/i);
      }
    });
  });
  
  describe('Edge Cases', () => {
    it('should handle test files with no test cases gracefully', () => {
      // Create an empty test file
      const emptyTest = `
import { describe } from 'vitest';

describe('Empty suite', () => {
  // No tests
});
`;
      fs.writeFileSync(path.join(testProjectDir, 'empty.test.js'), emptyTest);
      
      // Install dependencies first
      execSync(`cd ${testProjectDir} && npm install --no-save vitest@latest`, {
        stdio: 'ignore'
      });
      
      // Run 3pio
      try {
        execSync(`cd ${testProjectDir} && ${threePioCmd} npx vitest run empty.test.js`, {
          stdio: 'ignore'
        });
      } catch (e) {
        // Ignore errors
      }
      
      // Verify files are still created
      const runsDir = path.join(testProjectDir, '.3pio', 'runs');
      const latestRun = fs.readdirSync(runsDir).sort().pop()!;
      const testRunPath = path.join(runsDir, latestRun, 'test-run.md');
      
      expect(fs.existsSync(testRunPath)).toBe(true);
      
      const content = fs.readFileSync(testRunPath, 'utf8');
      expect(content).toContain('empty.test.js');
    });
    
    it('should handle very long test names correctly', () => {
      const longNameTest = `
import { describe, it, expect } from 'vitest';

describe('Suite with extremely long name that should be handled properly by the reporting system', () => {
  it('should handle this incredibly long test name that goes on and on and on without breaking the markdown formatting', () => {
    expect(true).toBe(true);
  });
});
`;
      fs.writeFileSync(path.join(testProjectDir, 'longname.test.js'), longNameTest);
      
      // Install dependencies first
      execSync(`cd ${testProjectDir} && npm install --no-save vitest@latest`, {
        stdio: 'ignore'
      });
      
      // Run 3pio
      try {
        execSync(`cd ${testProjectDir} && ${threePioCmd} npx vitest run longname.test.js`, {
          stdio: 'ignore'
        });
      } catch (e) {
        // Ignore errors
      }
      
      // Check that long names don't break formatting
      const runsDir = path.join(testProjectDir, '.3pio', 'runs');
      const latestRun = fs.readdirSync(runsDir).sort().pop()!;
      const testRunPath = path.join(runsDir, latestRun, 'test-run.md');
      const content = fs.readFileSync(testRunPath, 'utf8');
      
      expect(content).toContain('Suite with extremely long name');
      expect(content).toContain('incredibly long test name');
      
      // Markdown should still be valid
      expect(content.split('\n').filter(line => line.startsWith('#')).length).toBeGreaterThan(0);
    });
  });
});