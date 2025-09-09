import { describe, it, expect, beforeAll } from 'vitest';
import { execSync } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

describe('Console Output System Test', () => {
  const testProjectDir = path.join(__dirname, 'jest-project');
  const cliPath = path.resolve(__dirname, '../../../dist/cli.js');
  
  beforeAll(() => {
    // Ensure the CLI is built
    if (!fs.existsSync(cliPath)) {
      execSync('npm run build', { cwd: path.resolve(__dirname, '../../..') });
    }
    
    // Clean up any existing .3pio directory from previous test runs
    const threepioDir = path.join(testProjectDir, '.3pio');
    if (fs.existsSync(threepioDir)) {
      fs.rmSync(threepioDir, { recursive: true, force: true });
    }
    
    // Copy sample jest project files to test directory
    const sourceDir = path.resolve(__dirname, '../../../sample-projects/jest-project');
    
    // Copy package.json
    fs.copyFileSync(
      path.join(sourceDir, 'package.json'),
      path.join(testProjectDir, 'package.json')
    );
    
    // Copy test files
    fs.copyFileSync(
      path.join(sourceDir, 'math.test.js'),
      path.join(testProjectDir, 'math.test.js')
    );
    
    fs.copyFileSync(
      path.join(sourceDir, 'string.test.js'),
      path.join(testProjectDir, 'string.test.js')
    );
    
    // Install dependencies
    execSync('npm install', { cwd: testProjectDir, stdio: 'ignore' });
  });
  
  it('should produce expected console output format', async () => {
    // Run the CLI in the test project directory (expect it to fail since tests fail)
    let output: string;
    try {
      output = execSync(`node ${cliPath} npm test`, {
        cwd: testProjectDir,
        encoding: 'utf-8',
        env: { ...process.env, NO_COLOR: '1' }
      });
    } catch (error: any) {
      // The command will fail because tests fail, but we still want the output
      output = error.stdout;
    }
    
    // Replace dynamic timestamp and time values with placeholders for comparison
    const normalizedOutput = output
      .replace(/\d{4}-\d{2}-\d{2}T\d{6,9}Z(-[a-z0-9-]+)?/gi, 'TIMESTAMP')
      .replace(/Time:\s+\d+\.\d+s/g, 'Time:        X.XXXs');
    
    const expectedOutput = `
Greetings! I will now execute the test command:
\`npm test\`

Full report: .3pio/runs/TIMESTAMP/test-run.md

The following 2 test files will be run:
- /Users/zk/code/3pio/tests/system/console-output/jest-project/string.test.js
- /Users/zk/code/3pio/tests/system/console-output/jest-project/math.test.js

Beginning test execution now...

RUNNING  ./string.test.js
FAIL     ./string.test.js
  String operations
    ✕ should fail this test (3 ms)
  String operations
    ✕ should skip this test (0 ms)
  See .3pio/runs/TIMESTAMP/logs/math.test.js.log
    
RUNNING  ./math.test.js
PASS     ./math.test.js

Test Files: 1 failed,  1 passed, 2 total
Time:        X.XXXs

`;
    
    expect(normalizedOutput).toBe(expectedOutput);
    
    // Check that .3pio directory structure was created
    const threepioDir = path.join(testProjectDir, '.3pio');
    expect(fs.existsSync(threepioDir)).toBe(true);
    
    // Find the run directory (there should be exactly one)
    const runsDir = path.join(threepioDir, 'runs');
    expect(fs.existsSync(runsDir)).toBe(true);
    
    const runDirs = fs.readdirSync(runsDir);
    expect(runDirs.length).toBe(1);
    
    const runDir = path.join(runsDir, runDirs[0]);
    
    // Check for test-run.md
    const testRunFile = path.join(runDir, 'test-run.md');
    expect(fs.existsSync(testRunFile)).toBe(true);
    
    // Check for output.log
    const outputLogFile = path.join(runDir, 'output.log');
    expect(fs.existsSync(outputLogFile)).toBe(true);
    
    // Check for logs directory
    const logsDir = path.join(runDir, 'logs');
    expect(fs.existsSync(logsDir)).toBe(true);
    
    // Check that log files were created for each test file
    const mathLogFile = path.join(logsDir, 'math.test.js.log');
    const stringLogFile = path.join(logsDir, 'string.test.js.log');
    
    // Wait a bit for log files to be created during finalization
    await new Promise(resolve => setTimeout(resolve, 100));
    
    expect(fs.existsSync(mathLogFile)).toBe(true);
    expect(fs.existsSync(stringLogFile)).toBe(true);
    
    // Note: For Jest, individual log files may be empty because Jest runs tests in worker processes
    // and the reporter cannot capture console output. All output is captured in output.log instead.
    // We just verify the files exist, not their content.
    
    // Verify test-run.md has content
    const testRunContent = fs.readFileSync(testRunFile, 'utf-8');
    expect(testRunContent).toContain('# 3pio Test Run');
    expect(testRunContent).toContain('math.test.js');
    expect(testRunContent).toContain('string.test.js');
    
    // Verify output.log has content
    const outputLogContent = fs.readFileSync(outputLogFile, 'utf-8');
    expect(outputLogContent).toContain('# 3pio Test Output Log');
  });
});