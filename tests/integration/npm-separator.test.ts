import { describe, it, expect, beforeEach } from 'vitest';
import { execSync } from 'child_process';
import path from 'path';
import fs from 'fs';

describe('npm test with -- separator integration', () => {
  const cliPath = path.join(__dirname, '../../dist/cli.js');
  const fixturesDir = path.join(__dirname, '../fixtures');
  const projectDir = path.join(fixturesDir, 'npm-separator-jest');

  // Helper function to clean .3pio directory
  const cleanProjectOutput = (projectPath: string): void => {
    const threePioDir = path.join(projectPath, '.3pio');
    if (fs.existsSync(threePioDir)) {
      fs.rmSync(threePioDir, { recursive: true, force: true });
    }
  };

  // Helper function to get latest run directory
  const getLatestRunDir = (projectPath: string): string => {
    const runsDir = path.join(projectPath, '.3pio', 'runs');
    const runDirs = fs.readdirSync(runsDir);
    const latestRun = runDirs.sort().pop()!;
    return path.join(runsDir, latestRun);
  };

  beforeEach(() => {
    cleanProjectOutput(projectDir);
  });

  it('should handle npm test -- file.test.js command format with full verification', () => {
    // This test verifies that the command format is preserved correctly:
    // 3pio npm test -- example.test.js
    
    try {
      const output = execSync(
        `node ${cliPath} npm test -- example.test.js`,
        { 
          cwd: projectDir,
          encoding: 'utf-8',
          env: { ...process.env, NO_COLOR: '1' }
        }
      );
      
      // Check that the command is displayed correctly with the -- separator
      expect(output).toContain('npm test -- example.test.js');
      expect(output).toContain('example.test.js');
      
      // Get the run directory for file verifications
      const runDir = getLatestRunDir(projectDir);
      
      // 1. Verify all required files exist
      const testRunPath = path.join(runDir, 'test-run.md');
      const outputLogPath = path.join(runDir, 'output.log');
      const logsDir = path.join(runDir, 'logs');
      
      expect(fs.existsSync(testRunPath)).toBe(true);
      expect(fs.existsSync(outputLogPath)).toBe(true);
      expect(fs.existsSync(logsDir)).toBe(true);
      
      // 2. Verify test-run.md content
      const testRunContent = fs.readFileSync(testRunPath, 'utf8');
      expect(testRunContent).toContain('# 3pio Test Run');
      expect(testRunContent).toContain('- Timestamp:');
      expect(testRunContent).toContain('- Arguments: `npm test -- example.test.js`');
      expect(testRunContent).toContain('## Summary');
      expect(testRunContent).toContain('- Total Files:');
      expect(testRunContent).toContain('example.test.js');
      expect(testRunContent).toContain('[Log](./logs/example.test.js.log)');
      expect(testRunContent).toContain('[output.log](./output.log)');
      
      // 3. Verify output.log content
      const outputLogContent = fs.readFileSync(outputLogPath, 'utf8');
      expect(outputLogContent).toContain('# 3pio Test Output Log');
      expect(outputLogContent).toContain('# Timestamp:');
      expect(outputLogContent).toContain('# Command: npm test -- example.test.js');
      expect(outputLogContent).toContain('# This file contains all stdout/stderr output from the test run.');
      expect(outputLogContent).toContain('# ---');
      
      // 4. Verify individual log file
      const exampleLogPath = path.join(logsDir, 'example.test.js.log');
      expect(fs.existsSync(exampleLogPath)).toBe(true);
      
      const exampleLogContent = fs.readFileSync(exampleLogPath, 'utf8');
      expect(exampleLogContent).toContain('# File: example.test.js');
      expect(exampleLogContent).toContain('# Timestamp:');
      expect(exampleLogContent).toContain('# This file contains all stdout/stderr output from the test file execution.');
      expect(exampleLogContent).toContain('# ---');
      
    } catch (error: any) {
      // If there's an error, still perform verifications on generated files
      if (error.stdout) {
        expect(error.stdout).toContain('npm test -- example.test.js');
        expect(error.stdout).toContain('example.test.js');
        
        // Even with errors, verify files were created
        const runDir = getLatestRunDir(projectDir);
        const testRunPath = path.join(runDir, 'test-run.md');
        const outputLogPath = path.join(runDir, 'output.log');
        
        expect(fs.existsSync(testRunPath)).toBe(true);
        expect(fs.existsSync(outputLogPath)).toBe(true);
      } else {
        throw error;
      }
    }
  });

  it('should pass all arguments after -- to the test runner with full verification', () => {
    try {
      // Test with multiple arguments after --
      const output = execSync(
        `node ${cliPath} npm test -- example.test.js --coverage`,
        { 
          cwd: projectDir,
          encoding: 'utf-8',
          env: { ...process.env, NO_COLOR: '1' }
        }
      );
      
      // Check that the command preserves all arguments
      expect(output).toContain('npm test -- example.test.js --coverage');
      expect(output).toContain('example.test.js');
      
      // Get the run directory for file verifications
      const runDir = getLatestRunDir(projectDir);
      
      // 1. Verify all required files exist
      const testRunPath = path.join(runDir, 'test-run.md');
      const outputLogPath = path.join(runDir, 'output.log');
      const logsDir = path.join(runDir, 'logs');
      
      expect(fs.existsSync(testRunPath)).toBe(true);
      expect(fs.existsSync(outputLogPath)).toBe(true);
      expect(fs.existsSync(logsDir)).toBe(true);
      
      // 2. Verify test-run.md shows the full command with coverage flag
      const testRunContent = fs.readFileSync(testRunPath, 'utf8');
      expect(testRunContent).toContain('# 3pio Test Run');
      expect(testRunContent).toContain('- Arguments: `npm test -- example.test.js --coverage`');
      expect(testRunContent).toContain('## Summary');
      
      // 3. Verify output.log shows the full command
      const outputLogContent = fs.readFileSync(outputLogPath, 'utf8');
      expect(outputLogContent).toContain('# 3pio Test Output Log');
      expect(outputLogContent).toContain('# Command: npm test -- example.test.js --coverage');
      expect(outputLogContent).toContain('# ---');
      
      // 4. Verify individual log file exists
      const exampleLogPath = path.join(logsDir, 'example.test.js.log');
      expect(fs.existsSync(exampleLogPath)).toBe(true);
      
    } catch (error: any) {
      // If there's an error, still check the command format and verify files
      if (error.stdout) {
        expect(error.stdout).toContain('npm test -- example.test.js --coverage');
        expect(error.stdout).toContain('example.test.js');
        
        // Verify files were still created
        const runDir = getLatestRunDir(projectDir);
        expect(fs.existsSync(path.join(runDir, 'test-run.md'))).toBe(true);
        expect(fs.existsSync(path.join(runDir, 'output.log'))).toBe(true);
      } else {
        throw error;
      }
    }
  });
});