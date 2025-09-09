import { describe, it, expect } from 'vitest';
import { execSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';

describe('Test Case Reporting Integration', () => {
  const threePioCmd = path.join(__dirname, '../../dist/cli.js');
  const fixturesDir = path.join(__dirname, '../fixtures');
  
  // Helper function to get latest run directory
  const getLatestRunDir = (projectPath: string): string => {
    const runsDir = path.join(projectPath, '.3pio', 'runs');
    const runDirs = fs.readdirSync(runsDir);
    const latestRun = runDirs.sort().pop()!;
    return path.join(runsDir, latestRun);
  };
  
  // Helper function to clean .3pio directory
  const cleanProjectOutput = (projectPath: string): void => {
    const threePioDir = path.join(projectPath, '.3pio');
    if (fs.existsSync(threePioDir)) {
      fs.rmSync(threePioDir, { recursive: true, force: true });
    }
  };
  
  describe('Report File Generation', () => {
    const projectDir = path.join(fixturesDir, 'basic-vitest');
    
    beforeEach(() => {
      cleanProjectOutput(projectDir);
    });
    
    it('should create all expected report files', () => {
      // Run 3pio
      try {
        execSync(`${threePioCmd} npx vitest run math.test.js string.test.js`, {
          cwd: projectDir,
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected, ignore non-zero exit code
      }
      
      // Find the latest run directory
      const runDir = getLatestRunDir(projectDir);
      
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
    const projectDir = path.join(fixturesDir, 'basic-vitest');
    
    beforeEach(() => {
      cleanProjectOutput(projectDir);
    });
    
    it('should have correct structure and test case details', () => {
      // Run 3pio
      try {
        execSync(`${threePioCmd} npx vitest run math.test.js string.test.js`, {
          cwd: projectDir,
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected
      }
      
      // Find and read test-run.md
      const runDir = getLatestRunDir(projectDir);
      const testRunPath = path.join(runDir, 'test-run.md');
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
    const projectDir = path.join(fixturesDir, 'basic-vitest');
    
    beforeEach(() => {
      cleanProjectOutput(projectDir);
    });
    
    it('should have correct header and capture console output', () => {
      // Run 3pio
      try {
        execSync(`${threePioCmd} npx vitest run math.test.js string.test.js`, {
          cwd: projectDir,
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected
      }
      
      // Find and read output.log
      const runDir = getLatestRunDir(projectDir);
      const outputLogPath = path.join(runDir, 'output.log');
      const content = fs.readFileSync(outputLogPath, 'utf8');
      
      // Check header
      expect(content).toContain('# 3pio Test Output Log');
      expect(content).toContain('# Timestamp:');
      expect(content).toContain('# Command: npx vitest run math.test.js string.test.js');
      expect(content).toContain('# This file contains all stdout/stderr output from the test run.');
      expect(content).toContain('# ---');
      
      // Note: Vitest doesn't output console.log to stdout by default
      // So we check for Vitest's own output instead
      expect(content).toContain('RUN');
      
      // Should NOT contain output from skipped test
      expect(content).not.toContain('This should not run');
    });
  });
  
  describe('Individual Log Files Content', () => {
    const projectDir = path.join(fixturesDir, 'basic-vitest');
    
    beforeEach(() => {
      cleanProjectOutput(projectDir);
    });
    
    it('should have correct headers in math.test.js.log', () => {
      // Run 3pio
      try {
        execSync(`${threePioCmd} npx vitest run math.test.js string.test.js`, {
          cwd: projectDir,
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected
      }
      
      // Find and read math.test.js.log
      const runDir = getLatestRunDir(projectDir);
      const mathLogPath = path.join(runDir, 'logs', 'math.test.js.log');
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
      // Run 3pio
      try {
        execSync(`${threePioCmd} npx vitest run math.test.js string.test.js`, {
          cwd: projectDir,
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected
      }
      
      // Find and read string.test.js.log
      const runDir = getLatestRunDir(projectDir);
      const stringLogPath = path.join(runDir, 'logs', 'string.test.js.log');
      const content = fs.readFileSync(stringLogPath, 'utf8');
      
      // Check header
      expect(content).toContain('# File: string.test.js');
      expect(content).toContain('# Timestamp:');
      expect(content).toContain('# This file contains all stdout/stderr output from the test file execution.');
      expect(content).toContain('# ---');
    });
  });
  
  describe('Edge Cases', () => {
    it('should handle test files with no test cases gracefully', () => {
      const projectDir = path.join(fixturesDir, 'empty-vitest');
      cleanProjectOutput(projectDir);
      
      // Run 3pio
      try {
        execSync(`${threePioCmd} npx vitest run empty.test.js`, {
          cwd: projectDir,
          stdio: 'ignore'
        });
      } catch (e) {
        // Ignore errors
      }
      
      // Verify files are still created
      const runDir = getLatestRunDir(projectDir);
      const testRunPath = path.join(runDir, 'test-run.md');
      
      expect(fs.existsSync(testRunPath)).toBe(true);
      
      const content = fs.readFileSync(testRunPath, 'utf8');
      expect(content).toContain('empty.test.js');
    });
    
    it('should handle very long test names correctly', () => {
      const projectDir = path.join(fixturesDir, 'long-names-vitest');
      cleanProjectOutput(projectDir);
      
      // Run 3pio
      try {
        execSync(`${threePioCmd} npx vitest run longname.test.js`, {
          cwd: projectDir,
          stdio: 'ignore'
        });
      } catch (e) {
        // Ignore errors
      }
      
      // Check that long names don't break formatting
      const runDir = getLatestRunDir(projectDir);
      const testRunPath = path.join(runDir, 'test-run.md');
      const content = fs.readFileSync(testRunPath, 'utf8');
      
      expect(content).toContain('Suite with extremely long name');
      expect(content).toContain('incredibly long test name');
      
      // Markdown should still be valid
      expect(content.split('\n').filter(line => line.startsWith('#')).length).toBeGreaterThan(0);
    });
  });
  
  describe('Jest Integration', () => {
    const projectDir = path.join(fixturesDir, 'basic-jest');
    
    beforeEach(() => {
      cleanProjectOutput(projectDir);
    });
    
    it('should work with Jest test runner', () => {
      // Run 3pio with Jest
      try {
        execSync(`${threePioCmd} npx jest`, {
          cwd: projectDir,
          stdio: 'ignore'
        });
      } catch (e) {
        // Test failures are expected
      }
      
      // Verify report was created
      const runDir = getLatestRunDir(projectDir);
      const testRunPath = path.join(runDir, 'test-run.md');
      const content = fs.readFileSync(testRunPath, 'utf8');
      
      // Check for Jest-specific output
      expect(content).toContain('# 3pio Test Run');
      expect(content).toContain('math.test.js');
      expect(content).toContain('string.test.js');
      
      // Jest should also have test case details
      expect(content).toMatch(/✓|✕|○/); // Should have status symbols
    });
  });
});