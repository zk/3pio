// src/ipc-sender.ts
import * as fs from 'node:fs';
import * as path from 'node:path';

// src/utils/logger.ts
import * as fs2 from 'node:fs';
import * as path2 from 'node:path';

const __getOwnPropNames = Object.getOwnPropertyNames;
const __commonJS = (cb, mod) =>
  function __require() {
    return (
      mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod),
      mod.exports
    );
  };

// package.json
const require_package = __commonJS({
  'package.json': function (exports, module) {
    module.exports = {
      name: '@heyzk/3pio',
      version: '0.0.1',
      description: 'A context-competent test runner for coding agents',
      main: 'dist/index.js',
      bin: {
        '3pio': './dist/cli.js',
      },
      scripts: {
        build: 'node build.js',
        dev: 'node build.js --watch',
        test: 'vitest run',
        'test:watch': 'vitest',
        'test:coverage': 'vitest run --coverage',
        'test:unit': 'vitest run tests/unit',
        'test:integration': 'vitest run tests/integration',
        lint: 'eslint src --ext .ts',
        typecheck: 'tsc --noEmit',
        prepublishOnly: 'npm run build',
      },
      keywords: ['test', 'testing', 'jest', 'vitest', 'ai', 'adapter', 'reporter'],
      author: 'Zachary Kim (https://github.com/zk)',
      license: 'MIT',
      repository: {
        type: 'git',
        url: 'git+https://github.com/zk/3pio.git',
      },
      bugs: {
        url: 'https://github.com/zk/3pio/issues',
      },
      homepage: 'https://github.com/zk/3pio#readme',
      dependencies: {
        chokidar: '^3.6.0',
        commander: '^12.0.0',
        'lodash.debounce': '^4.0.8',
        'unique-names-generator': '^4.7.1',
        zx: '^8.1.0',
      },
      devDependencies: {
        '@types/lodash.debounce': '^4.0.9',
        '@types/node': '^20.14.0',
        '@typescript-eslint/eslint-plugin': '^7.0.0',
        '@typescript-eslint/parser': '^7.0.0',
        esbuild: '^0.21.0',
        eslint: '^8.57.0',
        typescript: '^5.4.0',
        vitest: '^1.6.0',
      },
      peerDependencies: {
        jest: '>=27.0.0',
        vitest: '>=0.34.0',
      },
      peerDependenciesMeta: {
        jest: {
          optional: true,
        },
        vitest: {
          optional: true,
        },
      },
      engines: {
        node: '>=18.0.0',
      },
      files: ['dist', 'README.md'],
      exports: {
        '.': './dist/index.js',
        './jest': './dist/jest.js',
        './vitest': './dist/vitest.js',
      },
    };
  },
});
const IPCSender = {
  /**
   * Send an event to the IPC file (used by adapters)
   */
  sendEvent(event) {
    return Promise.resolve(this.sendEventSync(event));
  },
  /**
   * Synchronous version of sendEvent
   */
  sendEventSync(event) {
    // Try to get IPC path from environment variable first (for workers)
    // Fall back to injected path
    const ipcPath =
      process.env.THREEPIO_IPC_PATH || /* __IPC_PATH__ */ 'WILL_BE_REPLACED'; /* __IPC_PATH__ */
    try {
      const dir = path.dirname(ipcPath);
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
      const line = `${JSON.stringify(event)}\n`;
      fs.appendFileSync(ipcPath, line);
    } catch {}
  },
};

const Logger = class _Logger {
  static instance = null;

  logPath;

  component;

  isInitComplete = false;

  constructor(component) {
    this.component = component;
    this.logPath = path2.join(process.cwd(), '.3pio', 'debug.log');
    this.ensureLogDirectory();
  }

  static getInstance(component) {
    if (!_Logger.instance) {
      _Logger.instance = new _Logger(component);
    }
    return _Logger.instance;
  }

  static create(component) {
    return new _Logger(component);
  }

  ensureLogDirectory() {
    try {
      fs2.mkdirSync(path2.dirname(this.logPath), { recursive: true });
    } catch {}
  }

  formatMessage(level, message, data) {
    const timestamp = /* @__PURE__ */ new Date().toISOString();
    const dataStr = data ? ` | ${JSON.stringify(data)}` : '';
    return `${timestamp} ${level.padEnd(5)} | [${this.component}] ${message}${dataStr}`;
  }

  writeLog(level, message, data) {
    try {
      const formattedMessage = this.formatMessage(level, message, data);
      fs2.appendFileSync(this.logPath, `${formattedMessage}\n`, 'utf8');
    } catch {}
  }

  /**
   * Log human-readable startup preamble without timestamps
   */
  startupPreamble(lines) {
    try {
      const preamble = lines.map((line) => `[${this.component}] ${line}`).join('\n');
      fs2.appendFileSync(this.logPath, `${preamble}\n`, 'utf8');
    } catch {}
  }

  /**
   * Log machine-readable initialization complete
   */
  initComplete(config) {
    this.isInitComplete = true;
    this.info('Initialization complete', config);
  }

  debug(message, data) {
    this.writeLog('DEBUG', message, data);
  }

  info(message, data) {
    this.writeLog('INFO', message, data);
  }

  warn(message, data) {
    this.writeLog('WARN', message, data);
  }

  error(message, error, data) {
    const errorData = {
      ...data,
      ...(error && {
        error: error.message || String(error),
        stack: error.stack,
      }),
    };
    this.writeLog('ERROR', message, errorData);
  }

  /**
   * Log lifecycle events with consistent narrative structure
   */
  lifecycle(event, details) {
    this.info(`Lifecycle: ${event}`, details);
  }

  /**
   * Log test execution flow
   */
  testFlow(action, testFile, details) {
    const message = testFile ? `Test flow: ${action} for ${testFile}` : `Test flow: ${action}`;
    this.info(message, details);
  }

  /**
   * Log IPC events
   */
  ipc(direction, eventType, details) {
    this.debug(`IPC ${direction}: ${eventType}`, details);
  }

  /**
   * Log command execution
   */
  command(cmd, args) {
    this.info(`Executing command: ${cmd}`, { args });
  }

  /**
   * Log decision points
   */
  decision(description, choice, reason) {
    this.info(`Decision: ${description}`, { choice, reason });
  }
};

// src/adapters/vitest.ts
const packageJson = require_package();

// Log level will be replaced at runtime
const LOG_LEVEL = /* __LOG_LEVEL__ */ 'WARN'; /* __LOG_LEVEL__ */

const ThreePioVitestReporter = class {
  originalStdoutWrite;

  originalStderrWrite;

  currentTestFile = null;

  captureEnabled = false;

  logger;

  filesStarted = /* @__PURE__ */ new Set();

  // Group tracking for universal abstractions
  discoveredGroups = /* @__PURE__ */ new Map();

  groupStarts = /* @__PURE__ */ new Map();

  fileGroups = /* @__PURE__ */ new Map();

  // Suite tracking removed - using modern Vitest 3+ API methods
  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
    this.logger = Logger.create('vitest-adapter');
    const ipcPath =
      process.env.THREEPIO_IPC_PATH || /* __IPC_PATH__ */ 'WILL_BE_REPLACED'; /* __IPC_PATH__ */
    this.logger.startupPreamble([
      '==================================',
      `3pio Vitest Adapter v${packageJson.version}`,
      'Configuration:',
      `  - IPC Path: ${ipcPath}`,
      `  - Process ID: ${process.pid}`,
      `  - Worker: ${process.env.VITEST_POOL_ID || 'main'}`,
      '==================================',
    ]);
  }

  // Group management helper methods
  getGroupId(hierarchy) {
    return hierarchy.join(':');
  }

  extractHierarchyFromTask(task, filePath) {
    if (!task) return [];

    const suites = [];
    let current = task;

    // Walk up parent chain to collect suite names
    while (current) {
      if (current.type === 'suite' && current.name) {
        suites.unshift(current.name);
      }
      current = current.parent || current.suite;
    }

    return suites;
  }

  buildHierarchyFromFile(filePath, suiteChain = []) {
    const hierarchy = [filePath];
    if (suiteChain && suiteChain.length > 0) {
      hierarchy.push(...suiteChain);
    }
    return hierarchy;
  }

  discoverGroups(filePath, suiteChain = []) {
    const groups = [];

    // First, the file itself is a group
    groups.push({
      hierarchy: [filePath],
      name: filePath,
      parentNames: [],
    });

    // Then each level of suites creates a nested group
    if (suiteChain && suiteChain.length > 0) {
      for (let i = 0; i < suiteChain.length; i++) {
        const parentNames = [filePath, ...suiteChain.slice(0, i)];
        const groupName = suiteChain[i];
        groups.push({
          hierarchy: [...parentNames, groupName],
          name: groupName,
          parentNames,
        });
      }
    }

    return groups;
  }

  ensureGroupsDiscovered(filePath, suiteChain = []) {
    const groups = this.discoverGroups(filePath, suiteChain);

    for (const group of groups) {
      const groupId = this.getGroupId(group.hierarchy);
      if (!this.discoveredGroups.has(groupId)) {
        this.discoveredGroups.set(groupId, group);
        this.logger.ipc('send', 'testGroupDiscovered', {
          groupName: group.name,
          parentNames: group.parentNames,
        });
        IPCSender.sendEvent({
          eventType: 'testGroupDiscovered',
          payload: {
            groupName: group.name,
            parentNames: group.parentNames,
          },
        }).catch((error) => {
          this.logger.error('Failed to send testGroupDiscovered event', error);
        });
      }
    }
  }

  ensureGroupStarted(hierarchy) {
    const groupId = this.getGroupId(hierarchy);
    if (!this.groupStarts.has(groupId)) {
      this.groupStarts.set(groupId, Date.now());

      const group = this.discoveredGroups.get(groupId);
      if (group) {
        this.logger.ipc('send', 'testGroupStart', {
          groupName: group.name,
          parentNames: group.parentNames,
        });
        IPCSender.sendEvent({
          eventType: 'testGroupStart',
          payload: {
            groupName: group.name,
            parentNames: group.parentNames,
          },
        }).catch((error) => {
          this.logger.error('Failed to send testGroupStart event', error);
        });
      }
    }
  }

  onInit(ctx) {
    this.logger.lifecycle('Test run initializing');
    const ipcPath =
      process.env.THREEPIO_IPC_PATH || /* __IPC_PATH__ */ 'WILL_BE_REPLACED'; /* __IPC_PATH__ */
    this.logger.info('IPC communication channel ready', { path: ipcPath });
    this.logger.initComplete({ ipcPath });

    // Send collection start event
    IPCSender.sendEvent({
      eventType: 'collectionStart',
      payload: { phase: 'collection' },
    }).catch((error) => {
      this.logger.error('Failed to send collectionStart event', error);
    });

    this.logger.debug('Starting global capture for test output');
    this.startCapture();
  }

  onPathsCollected(paths) {
    this.logger.info('Test paths collected', { count: paths?.length || 0 });

    // Send collection finish event when we have the full paths list
    // This is called before files are distributed to workers
    if (paths && paths.length > 0) {
      IPCSender.sendEvent({
        eventType: 'collectionFinish',
        payload: { collected: paths.length },
      }).catch((error) => {
        this.logger.error('Failed to send collectionFinish event', error);
      });
    }
  }

  onCollected(files) {
    this.logger.info('Test files collected', { count: files?.length || 0 });

    // Only send collection finish if onPathsCollected wasn't called
    // (for older Vitest versions or single-threaded mode)
    // Don't send in parallel mode as each worker only sees its subset
  }

  // New Vitest 3+ Reporter Methods
  onTestRunStart(specifications) {
    this.logger.info('[V3] onTestRunStart called', {
      count: specifications?.length || 0,
      specs: specifications?.map((s) => s.moduleId || s.filepath || s),
    });
  }

  onTestModuleCollected(testModule) {
    this.logger.info('[V3] onTestModuleCollected called', {
      moduleId: testModule?.moduleId,
      filepath: testModule?.filepath,
      name: testModule?.name,
    });

    // Discover the file as a root group
    const filePath = testModule?.filepath || testModule?.moduleId;
    if (filePath) {
      this.ensureGroupsDiscovered(filePath, []);
      // testFileStart event removed - using group events instead
    }
  }

  onTestSuiteReady(testSuite) {
    this.logger.info('[V3] onTestSuiteReady called', {
      name: testSuite?.name,
      filepath: testSuite?.filepath,
      id: testSuite?.id,
    });
  }

  onTestCaseReady(testCase) {
    this.logger.info('[V3] onTestCaseReady called', {
      name: testCase?.name,
      fullName: testCase?.fullName,
      id: testCase?.id,
      filepath: testCase?.filepath,
    });
  }

  onTestCaseResult(testCase) {
    const result = testCase?.result?.();
    const diagnostic = testCase?.diagnostic?.();
    const filePath = testCase?.module?.moduleId || testCase?.filepath;

    this.logger.info('[V3] onTestCaseResult called', {
      name: testCase?.name,
      fullName: testCase?.fullName,
      result,
      state: result?.state,
      filepath: testCase?.filepath,
      moduleId: testCase?.module?.moduleId,
      diagnostic,
      duration: diagnostic?.duration,
    });

    // Send IPC event for test case result with group hierarchy
    if (result && filePath) {
      // Extract hierarchy for this test case
      const suiteChain = this.extractHierarchyFromTask(testCase, filePath);
      const parentNames = this.buildHierarchyFromFile(filePath, suiteChain);

      // Ensure all parent groups are discovered and started
      this.ensureGroupsDiscovered(filePath, suiteChain);

      // Start all parent groups
      for (let i = 0; i <= suiteChain.length; i++) {
        const hierarchy = [filePath, ...suiteChain.slice(0, i)];
        this.ensureGroupStarted(hierarchy);
      }

      const status =
        result.state === 'passed'
          ? 'PASS'
          : result.state === 'failed'
            ? 'FAIL'
            : result.state === 'skipped'
              ? 'SKIP'
              : 'UNKNOWN';

      // Send test case event with group hierarchy
      this.logger.ipc('send', 'testCase', { testName: testCase.name, parentNames, status });

      // Build error object if test failed
      let errorObj = null;
      if (result.errors && result.errors.length > 0) {
        const firstError = result.errors[0];
        errorObj = {
          message: firstError.message || String(firstError),
          stack: firstError.stack || '',
          expected: firstError.expected || '',
          actual: firstError.actual || '',
          location: '', // Could extract from stack trace if needed
          errorType: firstError.name || 'Error',
        };
      }

      IPCSender.sendEvent({
        eventType: 'testCase',
        payload: {
          testName: testCase.name,
          parentNames,
          status,
          duration: diagnostic?.duration,
          error: errorObj,
        },
      }).catch((error) => {
        this.logger.error('Failed to send testCase event', error);
      });

      // Track test in file group
      const fileGroup = this.fileGroups.get(filePath);
      if (fileGroup) {
        fileGroup.tests.push({
          name: testCase.name,
          status,
          duration: diagnostic?.duration,
        });
      }
    }
  }

  onTestSuiteResult(testSuite) {
    this.logger.info('[V3] onTestSuiteResult called', {
      name: testSuite?.name,
      filepath: testSuite?.filepath,
      result: testSuite?.result?.(),
      state: testSuite?.result?.()?.state,
    });
  }

  onTestModuleEnd(testModule) {
    // Module end event - test results are handled via onTestCaseResult
    this.logger.info('[V3] onTestModuleEnd called', {
      moduleId: testModule?.moduleId,
      filepath: testModule?.filepath,
      name: testModule?.name,
    });

    // Send group result for the file when module completes
    const filePath = testModule?.filepath || testModule?.moduleId;
    if (filePath) {
      const fileGroup = this.fileGroups.get(filePath);
      if (fileGroup) {
        const fileDuration = fileGroup.startTime ? Date.now() - fileGroup.startTime : undefined;

        // Calculate totals from tracked tests
        const totals = {
          total: fileGroup.tests.length,
          passed: fileGroup.tests.filter((t) => t.status === 'PASS').length,
          failed: fileGroup.tests.filter((t) => t.status === 'FAIL').length,
          skipped: fileGroup.tests.filter((t) => t.status === 'SKIP').length,
        };

        const status =
          totals.failed > 0
            ? 'FAIL'
            : totals.skipped === totals.total && totals.total > 0
              ? 'SKIP'
              : 'PASS';

        this.logger.ipc('send', 'testGroupResult', { groupName: filePath, status, totals });
        IPCSender.sendEvent({
          eventType: 'testGroupResult',
          payload: {
            groupName: filePath,
            parentNames: [],
            status,
            duration: fileDuration,
            totals,
          },
        }).catch((error) => {
          this.logger.error('Failed to send testGroupResult', error);
        });
      }
    }
  }

  // sendTestCasesFromModule removed - using modern Vitest 3+ API methods instead

  onTestRunEnd(testModules, unhandledErrors, reason) {
    this.logger.info('[V3] onTestRunEnd called', {
      modules: testModules?.length || 0,
      errors: unhandledErrors?.length || 0,
      reason,
    });
    this.logger.lifecycle('Test run complete (V3)', {
      modules: testModules?.length || 0,
      errors: unhandledErrors?.length || 0,
    });
  }

  onHookStart(hook) {
    this.logger.debug('[V3] onHookStart called', {
      type: hook?.type,
      name: hook?.name,
    });
  }

  onHookEnd(hook) {
    this.logger.debug('[V3] onHookEnd called', {
      type: hook?.type,
      name: hook?.name,
    });
  }

  onTestAnnotate(testCase, annotation) {
    this.logger.debug('[V3] onTestAnnotate called', {
      testName: testCase?.name,
      annotation,
    });
  }

  onTestFileStart(file) {
    this.logger.testFlow('Starting test file', file.filepath);
    this.currentTestFile = file.filepath;

    if (!this.filesStarted.has(file.filepath)) {
      this.filesStarted.add(file.filepath);

      // Discover the file as a root group and start it
      this.ensureGroupsDiscovered(file.filepath, []);
      this.ensureGroupStarted([file.filepath]);

      // Store file group info
      this.fileGroups.set(file.filepath, {
        startTime: Date.now(),
        tests: [],
      });

      // testFileStart event removed - using group events instead
    }
    this.startCapture();
  }

  onTestFileResult(file) {
    // Legacy method - no longer processing here
    // All test results are handled via onTestCaseResult and onTestModuleEnd
    this.stopCapture();
    this.currentTestFile = null;
  }

  // sendTestCaseEvents removed - using modern Vitest 3+ API methods instead
  async onFinished(files, errors) {
    this.logger.lifecycle('Test run finishing', {
      files: files?.length || 0,
      errors: errors?.length || 0,
    });
    this.stopCapture();

    // Minimal fallback for Vitest 1.x compatibility
    // Modern Vitest 3+ uses onTestCaseResult and onTestModuleEnd instead
    if (files && files.length > 0 && this.filesStarted.size === 0) {
      this.logger.info('Using legacy fallback for older Vitest version', { count: files.length });
      for (const file of files) {
        this.processFileResults(file);
      }
    }
    this.logger.lifecycle('Vitest adapter shutdown complete');
  }

  startCapture() {
    if (this.captureEnabled) return;
    this.captureEnabled = true;
    this.logger.debug('Starting stdout/stderr capture', { currentFile: this.currentTestFile });
    // Old stdout/stderr capture removed - using group events instead
    // Output is now captured by group events (groupStdout/groupStderr)
    process.stdout.write = (chunk, ...args) => {
      // Silent capture - output handled by group events
      return true;
    };
    process.stderr.write = (chunk, ...args) => {
      // Silent capture - output handled by group events
      return true;
    };
  }

  stopCapture() {
    if (!this.captureEnabled) return;
    this.captureEnabled = false;
    this.logger.debug('Stopping stdout/stderr capture');
    process.stdout.write = this.originalStdoutWrite;
    process.stderr.write = this.originalStderrWrite;
  }

  // Simplified file processing for legacy Vitest compatibility
  processFileResults(file) {
    const filePath = file.filepath;

    // Ensure file group is discovered and started
    this.ensureGroupsDiscovered(filePath, []);
    this.ensureGroupStarted([filePath]);

    // Process test cases if available
    if (file.tasks) {
      this.processTasksSimple(filePath, file.tasks);
    }

    // Send file result
    let status = 'PASS';
    if (file.result?.state === 'fail') {
      status = 'FAIL';
    } else if (file.result?.state === 'skip' || file.mode === 'skip') {
      status = 'SKIP';
    }

    const totals = {
      total: file.tasks ? this.countTestsSimple(file.tasks) : 0,
      passed: file.tasks ? this.countPassedTestsSimple(file.tasks) : 0,
      failed: file.tasks ? this.countFailedTestsSimple(file.tasks) : 0,
      skipped: file.tasks ? this.countSkippedTestsSimple(file.tasks) : 0,
    };

    this.logger.ipc('send', 'testGroupResult', { groupName: filePath, status, totals });
    IPCSender.sendEvent({
      eventType: 'testGroupResult',
      payload: {
        groupName: filePath,
        parentNames: [],
        status,
        duration: file.result?.duration || 0,
        totals,
      },
    }).catch((error) => {
      this.logger.error('Failed to send testGroupResult', error);
    });
  }

  processTasksSimple(filePath, tasks) {
    for (const task of tasks) {
      if (task.type === 'test') {
        const status =
          task.result?.state === 'fail'
            ? 'FAIL'
            : task.result?.state === 'skip' || task.mode === 'skip'
              ? 'SKIP'
              : 'PASS';

        let error = null;
        if (task.result?.errors && task.result.errors.length > 0) {
          const firstError = task.result.errors[0];
          error = {
            message:
              typeof firstError === 'string'
                ? firstError
                : firstError.message || String(firstError),
            stack: firstError.stack || '',
            expected: firstError.expected || '',
            actual: firstError.actual || '',
            location: '',
            errorType: firstError.name || 'Error',
          };
        }

        // Simple hierarchy - just file and test name
        const suiteChain = this.extractHierarchyFromTask(task, filePath);
        const parentNames = this.buildHierarchyFromFile(filePath, suiteChain);

        // Ensure groups are discovered and started
        this.ensureGroupsDiscovered(filePath, suiteChain);
        for (let i = 0; i <= suiteChain.length; i++) {
          const hierarchy = [filePath, ...suiteChain.slice(0, i)];
          this.ensureGroupStarted(hierarchy);
        }

        this.logger.ipc('send', 'testCase', { testName: task.name, parentNames, status });
        IPCSender.sendEvent({
          eventType: 'testCase',
          payload: {
            testName: task.name,
            parentNames,
            status,
            duration: task.result?.duration,
            error,
          },
        }).catch((error_) => {
          this.logger.error('Failed to send testCase event', error_);
        });
      } else if (task.type === 'suite' && task.tasks) {
        this.processTasksSimple(filePath, task.tasks);
      }
    }
  }

  countTestsSimple(tasks) {
    let count = 0;
    for (const task of tasks) {
      if (task.type === 'test') {
        count++;
      } else if (task.tasks) {
        count += this.countTestsSimple(task.tasks);
      }
    }
    return count;
  }

  countPassedTestsSimple(tasks) {
    let count = 0;
    for (const task of tasks) {
      if (task.type === 'test' && task.result?.state === 'pass') {
        count++;
      } else if (task.tasks) {
        count += this.countPassedTestsSimple(task.tasks);
      }
    }
    return count;
  }

  countFailedTestsSimple(tasks) {
    let count = 0;
    for (const task of tasks) {
      if (task.type === 'test' && task.result?.state === 'fail') {
        count++;
      } else if (task.tasks) {
        count += this.countFailedTestsSimple(task.tasks);
      }
    }
    return count;
  }

  countSkippedTestsSimple(tasks) {
    let count = 0;
    for (const task of tasks) {
      if (task.type === 'test' && (task.result?.state === 'skip' || task.mode === 'skip')) {
        count++;
      } else if (task.tasks) {
        count += this.countSkippedTestsSimple(task.tasks);
      }
    }
    return count;
  }
};
export { ThreePioVitestReporter as default };
// # sourceMappingURL=vitest.js.map
