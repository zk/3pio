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
  // Vitest v2 tracking structures
  v2TaskIndex = /* @__PURE__ */ new Map();
  v2Emitted = /* @__PURE__ */ new Set();
  v2Planned = /* @__PURE__ */ new Map(); // key -> { filePath, suiteChain, name }
  v2Completed = /* @__PURE__ */ new Set(); // keys that reached terminal/backfill
  v2PendingEmitted = /* @__PURE__ */ new Set();

  // Suite tracking removed - using modern Vitest 3+ API methods
  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
    this.logger = Logger.create('vitest-adapter');

    // Check Vitest version - require 2.0 or higher; branch for v2 vs v3 APIs
    try {
      const vitestPkg = require('vitest/package.json');
      const version = vitestPkg.version;
      const majorVersion = parseInt(version.split('.')[0], 10);
      this.vitestMajor = Number.isFinite(majorVersion) ? majorVersion : 3;
      if (majorVersion < 2) {
        console.error(`\n[3pio] ERROR: Vitest ${version} is not supported. 3pio requires Vitest 2.0 or higher.\n`);
        console.error('Please upgrade Vitest: npm install --save-dev vitest@^2\n');
        process.exit(1);
      }
    } catch (e) {
      // If we can't check version, proceed but default to v3 behavior
      this.vitestMajor = 3;
      console.warn('[3pio] WARNING: Could not verify Vitest version. Assuming Vitest 3.x compatible reporter API.');
    }

    const ipcPath =
      process.env.THREEPIO_IPC_PATH || /* __IPC_PATH__ */ 'WILL_BE_REPLACED'; /* __IPC_PATH__ */
    this.logger.startupPreamble([
      '==================================',
      `3pio Vitest Adapter v${packageJson.version}`,
      'Configuration:',
      `  - IPC Path: ${ipcPath}`,
      `  - Process ID: ${process.pid}`,
      `  - Worker: ${process.env.VITEST_POOL_ID || 'main'}`,
      `  - Requires: Vitest 2.0+`,
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
    // For Vitest v2, index tasks for onTaskUpdate processing
    if (this.vitestMajor && this.vitestMajor < 3 && Array.isArray(files)) {
      try {
        this.indexV2Tasks(files);
        this.planV2Tests(files);
        this.logger.debug('[V2] Indexed tasks for onTaskUpdate processing', {
          count: this.v2TaskIndex.size,
        });
      } catch (e) {
        this.logger.error('Failed to index v2 tasks during onCollected', e);
      }
    }
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
    // Vitest v3+ only
    if (this.vitestMajor && this.vitestMajor < 3) return;
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

    // Ensure file group exists for tracking (Vitest 3 may not call onTestFileStart)
    if (filePath && !this.fileGroups.has(filePath)) {
      this.fileGroups.set(filePath, {
        startTime: Date.now(),
        tests: [],
      });
      this.logger.debug('Created file group for', filePath);
    }

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
    // Vitest v3+ only
    if (this.vitestMajor && this.vitestMajor < 3) return;
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

        // Group (file) is PASS only if all tests passed with no skips; otherwise FAIL
        const status = totals.failed === 0 && totals.skipped === 0 ? 'PASS' : 'FAIL';

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
    // Vitest v3+ only
    if (this.vitestMajor && this.vitestMajor < 3) return;
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
    // Vitest v3+ only
    if (this.vitestMajor && this.vitestMajor < 3) return;
    this.logger.debug('[V3] onHookStart called', {
      type: hook?.type,
      name: hook?.name,
    });
  }

  onHookEnd(hook) {
    // Vitest v3+ only
    if (this.vitestMajor && this.vitestMajor < 3) return;
    this.logger.debug('[V3] onHookEnd called', {
      type: hook?.type,
      name: hook?.name,
    });
  }

  onTestAnnotate(testCase, annotation) {
    // Vitest v3+ only
    if (this.vitestMajor && this.vitestMajor < 3) return;
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

    // For Vitest v2, walk files to backfill any missing events and group results
    if (this.vitestMajor && this.vitestMajor < 3) {
      try {
        // First, traverse finished files to backfill terminal cases and emit group results later
        this.processV2Files(files || []);

        // Then, emit SKIP for any planned tests that never completed (suite aborts)
        if (this.v2Planned && this.v2Planned.size > 0) {
          const hasUnhandled = Array.isArray(errors) && errors.length > 0;
          const fileErrored = /* @__PURE__ */ new Set();
          for (const [key, meta] of this.v2Planned.entries()) {
            if (this.v2Completed && this.v2Completed.has(key)) continue;
            const { filePath, suiteChain, name } = meta;
            this.ensureGroupsDiscovered(filePath, suiteChain);
            for (let i = 0; i <= suiteChain.length; i++) {
              const hierarchy = [filePath, ...suiteChain.slice(0, i)];
              this.ensureGroupStarted(hierarchy);
            }
            const parentNames = this.buildHierarchyFromFile(filePath, suiteChain);
            const finalStatus = hasUnhandled ? 'ERROR' : 'SKIP';
            this.logger.ipc('send', 'testCase', { testName: name, parentNames, status: finalStatus });
            await IPCSender.sendEvent({
              eventType: 'testCase',
              payload: { testName: name, parentNames, status: finalStatus, duration: undefined, error: null },
            }).catch((error) => this.logger.error('Failed to send planned testCase SKIP (v2)', error));
            if (!this.v2Completed) this.v2Completed = /* @__PURE__ */ new Set();
            this.v2Completed.add(key);
            if (!this.fileGroups.has(filePath)) {
              this.fileGroups.set(filePath, { startTime: Date.now(), tests: [] });
            }
            const fg = this.fileGroups.get(filePath);
            if (fg) fg.tests.push({ name, status: finalStatus, duration: undefined });

            // Emit a group error once per file when we classify aborted tests as ERROR
            if (hasUnhandled && !fileErrored.has(filePath)) {
              fileErrored.add(filePath);
              const message = errors && errors[0] && (errors[0].message || String(errors[0])) || 'Unhandled error';
              IPCSender.sendEvent({
                eventType: 'testGroupError',
                payload: {
                  groupName: filePath,
                  parentNames: [],
                  errorType: 'SETUP_FAILURE',
                  duration: undefined,
                  error: { message, phase: 'setup' },
                  metadata: undefined,
                },
              }).catch((err) => this.logger.error('Failed to send testGroupError (v2)', err));
            }
          }
        }
      } catch (e) {
        this.logger.error('Error while processing Vitest v2 files', e);
      }
    }

    this.stopCapture();
    this.logger.lifecycle('Vitest adapter shutdown complete');
  }

  // Vitest v2 compatibility: walk finished file tasks and emit events
  processV2Files(files) {
    if (!Array.isArray(files)) return;

    const toArray = (val) => (Array.isArray(val) ? val : val ? [val] : []);

    for (const file of files) {
      const filePath = file?.filepath || file?.file?.filepath || file?.name || file?.moduleId;
      if (!filePath) continue;

      // Ensure root group exists and is started
      this.ensureGroupsDiscovered(filePath, []);
      this.ensureGroupStarted([filePath]);

      // Store file group info if not present
      if (!this.fileGroups.has(filePath)) {
        this.fileGroups.set(filePath, { startTime: Date.now(), tests: [] });
      }

      const visitSuite = (suite, chain) => {
        const name = suite?.name;
        const nextChain = name ? [...chain, name] : chain;

        // Discover and start suite groups along the chain
        this.ensureGroupsDiscovered(filePath, nextChain);
        for (let i = 0; i <= nextChain.length; i++) {
          const hierarchy = [filePath, ...nextChain.slice(0, i)];
          this.ensureGroupStarted(hierarchy);
        }

        const tasks = toArray(suite?.tasks);
        for (const t of tasks) {
          if (!t) continue;
          if (t.type === 'suite') {
            visitSuite(t, nextChain);
          } else if (t.type === 'test') {
            const result = t.result || {};
            const state = result.state || t.state || 'unknown';
            const status = state === 'pass' || state === 'passed'
              ? 'PASS'
              : state === 'fail' || state === 'failed'
                ? 'FAIL'
                : state === 'skip' || state === 'skipped' || state === 'todo' || !state
                  ? 'SKIP'
                  : 'SKIP';

            const parentNames = this.buildHierarchyFromFile(filePath, nextChain);
            // Skip if already completed via onTaskUpdate
            const key = `${filePath}::${nextChain.join('::')}::${t.name}`;
            if (this.v2Completed && this.v2Completed.has(key)) continue;

            let errorObj = null;
            const firstError = (result.errors && result.errors[0]) || (t.errors && t.errors[0]);
            if (firstError) {
              errorObj = {
                message: firstError.message || String(firstError),
                stack: firstError.stack || '',
                expected: firstError.expected || '',
                actual: firstError.actual || '',
                location: '',
                errorType: firstError.name || 'Error',
              };
            }

            this.logger.ipc('send', 'testCase', { testName: t.name, parentNames, status });
            IPCSender.sendEvent({
              eventType: 'testCase',
              payload: {
                testName: t.name,
                parentNames,
                status,
                duration: result.duration,
                error: errorObj,
              },
            }).catch((error) => {
              this.logger.error('Failed to send testCase event (v2 backfill)', error);
            });

            const fg = this.fileGroups.get(filePath);
            if (fg) fg.tests.push({ name: t.name, status, duration: result.duration });
            if (!this.v2Completed) this.v2Completed = /* @__PURE__ */ new Set();
            this.v2Completed.add(key);
          }
        }
      };

      // v2 file may have tasks directly or nested under .tasks
      visitSuite(file, []);

      // Emit file group result
      const fileGroup = this.fileGroups.get(filePath);
      if (fileGroup) {
        const fileDuration = fileGroup.startTime ? Date.now() - fileGroup.startTime : undefined;
        const totals = {
          total: fileGroup.tests.length,
          passed: fileGroup.tests.filter((t) => t.status === 'PASS').length,
          failed: fileGroup.tests.filter((t) => t.status === 'FAIL').length,
          skipped: fileGroup.tests.filter((t) => t.status === 'SKIP').length,
        };
        // Group (file) is PASS only if all tests passed with no skips; otherwise FAIL
        const status = totals.failed === 0 && totals.skipped === 0 ? 'PASS' : 'FAIL';
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
          this.logger.error('Failed to send testGroupResult (v2)', error);
        });
      }
    }
  }

  // Heuristic: treat declaration files as typecheck tasks; exclude from test counts
  isTypeCheckPath(p) {
    if (!p || typeof p !== 'string') return false;
    // Normalize path separators
    const fp = p.toLowerCase();
    return fp.endsWith('.d.ts') || fp.endsWith('.d.tsx') || fp.includes('/types/') || fp.includes('test-d.ts');
  }

  // Vitest v2: build an index of test tasks for quick lookup by id
  indexV2Tasks(files) {
    const toArray = (val) => (Array.isArray(val) ? val : val ? [val] : []);
    this.v2TaskIndex.clear();
    for (const file of files) {
      const filePath = file?.filepath || file?.file?.filepath || file?.name || file?.moduleId;
      if (!filePath) continue;

      const visit = (node, chain) => {
        const name = node?.name;
        const nextChain = node?.type === 'suite' && name ? [...chain, name] : chain;
        const tasks = toArray(node?.tasks);
        for (const t of tasks) {
          if (!t) continue;
          if (t.type === 'suite') {
            visit(t, nextChain);
          } else if (t.type === 'test') {
            if (t.id != null) {
              this.v2TaskIndex.set(t.id, { filePath, suiteChain: nextChain, name: t.name });
            }
          }
        }
      };
      visit(file, []);
    }
  }

  // Vitest v2: plan all tests discovered during collection
  planV2Tests(files) {
    const toArray = (val) => (Array.isArray(val) ? val : val ? [val] : []);
    if (!this.v2Planned) this.v2Planned = /* @__PURE__ */ new Map();
    for (const file of files || []) {
      const filePath = file?.filepath || file?.file?.filepath || file?.name || file?.moduleId;
      if (!filePath) continue;
      const visit = (node, chain) => {
        const name = node?.name;
        const nextChain = node?.type === 'suite' && name ? [...chain, name] : chain;
        const tasks = toArray(node?.tasks);
        for (const t of tasks) {
          if (!t) continue;
          if (t.type === 'suite') {
            visit(t, nextChain);
          } else if (t.type === 'test') {
            const key = `${filePath}::${nextChain.join('::')}::${t.name}`;
            if (!this.v2Planned.has(key)) {
              this.v2Planned.set(key, { filePath, suiteChain: nextChain, name: t.name });
            }
          }
        }
      };
      visit(file, []);
    }
  }

  // Vitest v2: handle incremental result updates
  onTaskUpdate(packs) {
    if (!(this.vitestMajor && this.vitestMajor < 3)) return;
    if (!Array.isArray(packs)) return;

    const terminal = (st) => st === 'pass' || st === 'passed' || st === 'fail' || st === 'failed' || st === 'skip' || st === 'skipped' || st === 'todo';
    for (const pack of packs) {
      try {
        const id = Array.isArray(pack) ? pack[0] : pack?.id;
        const res = Array.isArray(pack) ? pack[1] : pack?.result;
        if (id == null || !res) continue;
        const meta = this.v2TaskIndex.get(id);
        if (!meta) continue;

        const state = res.state || 'unknown';
        const { filePath, suiteChain, name } = meta;
        const contentKey = `${filePath}::${suiteChain.join('::')}::${name}`;

        // Handle non-terminal states: record planned and emit PENDING once
        if (!terminal(state)) {
          if (!this.v2Planned.has(contentKey)) {
            this.v2Planned.set(contentKey, { filePath, suiteChain, name });
          }
          if (!this.v2PendingEmitted.has(contentKey)) {
            this.ensureGroupsDiscovered(filePath, suiteChain);
            for (let i = 0; i <= suiteChain.length; i++) {
              const hierarchy = [filePath, ...suiteChain.slice(0, i)];
              this.ensureGroupStarted(hierarchy);
            }
            const parentNames = this.buildHierarchyFromFile(filePath, suiteChain);
            this.logger.ipc('send', 'testCase', { testName: name, parentNames, status: 'PENDING' });
            IPCSender.sendEvent({
              eventType: 'testCase',
              payload: { testName: name, parentNames, status: 'PENDING', duration: undefined, error: null },
            }).catch((error) => this.logger.error('Failed to send testCase PENDING (v2)', error));
            this.v2PendingEmitted.add(contentKey);
          }
          continue;
        }

        // Terminal state: emit final if not already emitted
        if (this.v2Emitted.has(id)) continue;
        this.v2Emitted.add(id);

        // Ensure file group exists
        if (filePath && !this.fileGroups.has(filePath)) {
          this.fileGroups.set(filePath, { startTime: Date.now(), tests: [] });
        }

        // Ensure groups discovered & started
        this.ensureGroupsDiscovered(filePath, suiteChain);
        for (let i = 0; i <= suiteChain.length; i++) {
          const hierarchy = [filePath, ...suiteChain.slice(0, i)];
          this.ensureGroupStarted(hierarchy);
        }

        const parentNames = this.buildHierarchyFromFile(filePath, suiteChain);
        const status = state === 'pass' || state === 'passed' ? 'PASS' : state === 'fail' || state === 'failed' ? 'FAIL' : state === 'skip' || state === 'skipped' || state === 'todo' ? 'SKIP' : 'UNKNOWN';

        let errorObj = null;
        const firstError = res.errors && res.errors[0];
        if (firstError) {
          errorObj = {
            message: firstError.message || String(firstError),
            stack: firstError.stack || '',
            expected: firstError.expected || '',
            actual: firstError.actual || '',
            location: '',
            errorType: firstError.name || 'Error',
          };
        }

        this.logger.ipc('send', 'testCase', { testName: name, parentNames, status });
        IPCSender.sendEvent({
          eventType: 'testCase',
          payload: { testName: name, parentNames, status, duration: res.duration, error: errorObj },
        }).catch((error) => this.logger.error('Failed to send testCase event (v2 update)', error));

        const fg = this.fileGroups.get(filePath);
        if (fg) fg.tests.push({ name, status, duration: res.duration });
        // Mark completed by content key for reconciliation with planned tests
        if (!this.v2Completed) this.v2Completed = /* @__PURE__ */ new Set();
        this.v2Completed.add(contentKey);
      } catch (e) {
        this.logger.error('Error handling v2 onTaskUpdate pack', e);
      }
    }
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
};
export { ThreePioVitestReporter as default };
// # sourceMappingURL=vitest.js.map
