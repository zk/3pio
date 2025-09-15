var __getOwnPropNames = Object.getOwnPropertyNames;
var __commonJS = (cb, mod) => function __require() {
  return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
};

// package.json
var require_package = __commonJS({
  "package.json"(exports, module) {
    module.exports = {
      name: "@heyzk/3pio",
      version: "0.0.1",
      description: "A context-competent test runner for coding agents",
      main: "dist/index.js",
      bin: {
        "3pio": "./dist/cli.js"
      },
      scripts: {
        build: "node build.js",
        dev: "node build.js --watch",
        test: "vitest run",
        "test:watch": "vitest",
        "test:coverage": "vitest run --coverage",
        "test:unit": "vitest run tests/unit",
        "test:integration": "vitest run tests/integration",
        lint: "eslint src --ext .ts",
        typecheck: "tsc --noEmit",
        prepublishOnly: "npm run build"
      },
      keywords: [
        "test",
        "testing",
        "jest",
        "vitest",
        "ai",
        "adapter",
        "reporter"
      ],
      author: "Zachary Kim (https://github.com/zk)",
      license: "MIT",
      repository: {
        type: "git",
        url: "git+https://github.com/zk/3pio.git"
      },
      bugs: {
        url: "https://github.com/zk/3pio/issues"
      },
      homepage: "https://github.com/zk/3pio#readme",
      dependencies: {
        chokidar: "^3.6.0",
        commander: "^12.0.0",
        "lodash.debounce": "^4.0.8",
        "unique-names-generator": "^4.7.1",
        zx: "^8.1.0"
      },
      devDependencies: {
        "@types/lodash.debounce": "^4.0.9",
        "@types/node": "^20.14.0",
        "@typescript-eslint/eslint-plugin": "^7.0.0",
        "@typescript-eslint/parser": "^7.0.0",
        esbuild: "^0.21.0",
        eslint: "^8.57.0",
        typescript: "^5.4.0",
        vitest: "^1.6.0"
      },
      peerDependencies: {
        jest: ">=27.0.0",
        vitest: ">=0.34.0"
      },
      peerDependenciesMeta: {
        jest: {
          optional: true
        },
        vitest: {
          optional: true
        }
      },
      engines: {
        node: ">=18.0.0"
      },
      files: [
        "dist",
        "README.md"
      ],
      exports: {
        ".": "./dist/index.js",
        "./jest": "./dist/jest.js",
        "./vitest": "./dist/vitest.js"
      }
    };
  }
});

// src/ipc-sender.ts
import * as fs from "fs";
import * as path from "path";
var IPCSender = class {
  /**
   * Send an event to the IPC file (used by adapters)
   */
  static sendEvent(event) {
    return Promise.resolve(this.sendEventSync(event));
  }
  /**
   * Synchronous version of sendEvent
   */
  static sendEventSync(event) {
    // Try to get IPC path from environment variable first (for workers)
    // Fall back to injected path
    const ipcPath = process.env.THREEPIO_IPC_PATH || /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;
    try {
      const dir = path.dirname(ipcPath);
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
      const line = JSON.stringify(event) + "\n";
      fs.appendFileSync(ipcPath, line);
    } catch (error) {
    }
  }
};

// src/utils/logger.ts
import * as fs2 from "fs";
import * as path2 from "path";
var Logger = class _Logger {
  static instance = null;
  logPath;
  component;
  isInitComplete = false;
  constructor(component) {
    this.component = component;
    this.logPath = path2.join(process.cwd(), ".3pio", "debug.log");
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
    } catch {
    }
  }
  formatMessage(level, message, data) {
    const timestamp = (/* @__PURE__ */ new Date()).toISOString();
    const dataStr = data ? ` | ${JSON.stringify(data)}` : "";
    return `${timestamp} ${level.padEnd(5)} | [${this.component}] ${message}${dataStr}`;
  }
  writeLog(level, message, data) {
    try {
      const formattedMessage = this.formatMessage(level, message, data);
      fs2.appendFileSync(this.logPath, formattedMessage + "\n", "utf8");
    } catch {
    }
  }
  /**
   * Log human-readable startup preamble without timestamps
   */
  startupPreamble(lines) {
    try {
      const preamble = lines.map((line) => `[${this.component}] ${line}`).join("\n");
      fs2.appendFileSync(this.logPath, preamble + "\n", "utf8");
    } catch {
    }
  }
  /**
   * Log machine-readable initialization complete
   */
  initComplete(config) {
    this.isInitComplete = true;
    this.info("Initialization complete", config);
  }
  debug(message, data) {
    this.writeLog("DEBUG", message, data);
  }
  info(message, data) {
    this.writeLog("INFO", message, data);
  }
  warn(message, data) {
    this.writeLog("WARN", message, data);
  }
  error(message, error, data) {
    const errorData = {
      ...data,
      ...error && {
        error: error.message || String(error),
        stack: error.stack
      }
    };
    this.writeLog("ERROR", message, errorData);
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
var packageJson = require_package();
var ThreePioVitestReporter = class {
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
  // Track test results for accurate suite totals
  suiteTestResults = /* @__PURE__ */ new Map(); // Map<suiteName, {passed: number, failed: number, skipped: number}>
  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
    this.logger = Logger.create("vitest-adapter");
    const ipcPath = process.env.THREEPIO_IPC_PATH || /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;
    this.logger.startupPreamble([
      "==================================",
      `3pio Vitest Adapter v${packageJson.version}`,
      "Configuration:",
      `  - IPC Path: ${ipcPath}`,
      `  - Process ID: ${process.pid}`,
      `  - Worker: ${process.env.VITEST_POOL_ID || 'main'}`,
      "=================================="
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
      parentNames: []
    });

    // Then each level of suites creates a nested group
    if (suiteChain && suiteChain.length > 0) {
      for (let i = 0; i < suiteChain.length; i++) {
        const parentNames = [filePath, ...suiteChain.slice(0, i)];
        const groupName = suiteChain[i];
        groups.push({
          hierarchy: [...parentNames, groupName],
          name: groupName,
          parentNames: parentNames
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
        this.logger.ipc("send", "testGroupDiscovered", { groupName: group.name, parentNames: group.parentNames });
        IPCSender.sendEvent({
          eventType: 'testGroupDiscovered',
          payload: {
            groupName: group.name,
            parentNames: group.parentNames
          }
        }).catch((error) => {
          this.logger.error("Failed to send testGroupDiscovered event", error);
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
        this.logger.ipc("send", "testGroupStart", { groupName: group.name, parentNames: group.parentNames });
        IPCSender.sendEvent({
          eventType: 'testGroupStart',
          payload: {
            groupName: group.name,
            parentNames: group.parentNames
          }
        }).catch((error) => {
          this.logger.error("Failed to send testGroupStart event", error);
        });
      }
    }
  }
  onInit(ctx) {
    this.logger.lifecycle("Test run initializing");
    const ipcPath = process.env.THREEPIO_IPC_PATH || /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;
    this.logger.info("IPC communication channel ready", { path: ipcPath });
    this.logger.initComplete({ ipcPath });

    // Send collection start event
    IPCSender.sendEvent({
      eventType: "collectionStart",
      payload: { phase: "collection" }
    }).catch((error) => {
      this.logger.error("Failed to send collectionStart event", error);
    });
    
    this.logger.debug("Starting global capture for test output");
    this.startCapture();
  }
  onPathsCollected(paths) {
    this.logger.info("Test paths collected", { count: paths?.length || 0 });
    
    // Send collection finish event when we have the full paths list
    // This is called before files are distributed to workers
    if (paths && paths.length > 0) {
      IPCSender.sendEvent({
        eventType: "collectionFinish",
        payload: { collected: paths.length }
      }).catch((error) => {
        this.logger.error("Failed to send collectionFinish event", error);
      });
    }
  }
  onCollected(files) {
    this.logger.info("Test files collected", { count: files?.length || 0 });
    
    // Only send collection finish if onPathsCollected wasn't called
    // (for older Vitest versions or single-threaded mode)
    // Don't send in parallel mode as each worker only sees its subset
  }
  
  // New Vitest 3+ Reporter Methods
  onTestRunStart(specifications) {
    this.logger.info("[V3] onTestRunStart called", { 
      count: specifications?.length || 0,
      specs: specifications?.map(s => s.moduleId || s.filepath || s) 
    });
  }
  
  onTestModuleCollected(testModule) {
    this.logger.info("[V3] onTestModuleCollected called", {
      moduleId: testModule?.moduleId,
      filepath: testModule?.filepath,
      name: testModule?.name
    });

    // Discover the file as a root group
    const filePath = testModule?.filepath || testModule?.moduleId;
    if (filePath) {
      this.ensureGroupsDiscovered(filePath, []);
      // testFileStart event removed - using group events instead
    }
  }
  
  onTestSuiteReady(testSuite) {
    this.logger.info("[V3] onTestSuiteReady called", { 
      name: testSuite?.name,
      filepath: testSuite?.filepath,
      id: testSuite?.id 
    });
  }
  
  onTestCaseReady(testCase) {
    this.logger.info("[V3] onTestCaseReady called", { 
      name: testCase?.name,
      fullName: testCase?.fullName,
      id: testCase?.id,
      filepath: testCase?.filepath 
    });
  }
  
  onTestCaseResult(testCase) {
    const result = testCase?.result?.();
    const diagnostic = testCase?.diagnostic?.();
    const filePath = testCase?.module?.moduleId || testCase?.filepath;

    this.logger.info("[V3] onTestCaseResult called", {
      name: testCase?.name,
      fullName: testCase?.fullName,
      result: result,
      state: result?.state,
      filepath: testCase?.filepath,
      moduleId: testCase?.module?.moduleId,
      diagnostic: diagnostic,
      duration: diagnostic?.duration
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

      const status = result.state === 'passed' ? 'PASS' :
                     result.state === 'failed' ? 'FAIL' :
                     result.state === 'skipped' ? 'SKIP' : 'UNKNOWN';

      // Send test case event with group hierarchy
      this.logger.ipc("send", "testCase", { testName: testCase.name, parentNames, status });

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
          errorType: firstError.name || 'Error'
        };
      }

      IPCSender.sendEvent({
        eventType: "testCase",
        payload: {
          testName: testCase.name,
          parentNames: parentNames,
          status: status,
          duration: diagnostic?.duration,
          error: errorObj
        }
      }).catch((error) => {
        this.logger.error("Failed to send testCase event", error);
      });

      // Track test in file group
      const fileGroup = this.fileGroups.get(filePath);
      if (fileGroup) {
        fileGroup.tests.push({
          name: testCase.name,
          status: status,
          duration: diagnostic?.duration
        });
      }
    }
  }
  
  onTestSuiteResult(testSuite) {
    this.logger.info("[V3] onTestSuiteResult called", { 
      name: testSuite?.name,
      filepath: testSuite?.filepath,
      result: testSuite?.result?.(),
      state: testSuite?.result?.()?.state 
    });
  }
  
  onTestModuleEnd(testModule) {
    // Log ALL data in testModule to see what's available
    this.logger.info("[V3] onTestModuleEnd - Full module data", {
      hasModule: !!testModule,
      moduleKeys: testModule ? Object.keys(testModule) : [],
      moduleId: testModule?.moduleId,
      filepath: testModule?.filepath,
      name: testModule?.name,
      hasChildren: !!testModule?.children,
      childrenType: typeof testModule?.children,
      childrenIsArray: Array.isArray(testModule?.children),
      childrenKeys: testModule?.children && typeof testModule?.children === 'object' ? Object.keys(testModule?.children).slice(0, 10) : [],
      hasTests: !!testModule?.tests,
      testsLength: testModule?.tests?.length,
      hasTasks: !!testModule?.tasks,
      tasksLength: testModule?.tasks?.length,
      // Check task field
      hasTask: !!testModule?.task,
      taskKeys: testModule?.task ? Object.keys(testModule.task).slice(0, 20) : [],
      taskHasTasks: !!testModule?.task?.tasks,
      taskTasksLength: testModule?.task?.tasks?.length
    });
    
    // Send testFileResult event when module completes
    const filePath = testModule?.filepath || testModule?.moduleId;
    if (filePath) {
      // Determine module status from test results
      let status = "PASS";
      const failedTests = [];
      
      // Try to get test data from various possible locations
      // testModule.children is a Set<Task> according to Vitest API, but sometimes it's an empty object
      let testData = null;
      if (testModule.children && testModule.children instanceof Set && testModule.children.size > 0) {
        testData = Array.from(testModule.children);
      } else if (testModule.task?.tasks && testModule.task.tasks.length > 0) {
        testData = testModule.task.tasks;
      } else if (testModule.tasks && testModule.tasks.length > 0) {
        testData = testModule.tasks;
      }
      
      if (testData && Array.isArray(testData)) {
        this.logger.debug("Found test data array", { length: testData.length });
        // Debug: Log first child to see what data is available
        if (testData.length > 0) {
          const firstChild = testData[0];
          this.logger.debug("First child in module", {
            type: firstChild.type,
            name: firstChild.name,
            hasResult: !!firstChild.result,
            resultKeys: firstChild.result ? Object.keys(firstChild.result) : [],
            resultState: firstChild.result?.state,
            resultDuration: firstChild.result?.duration
          });
        }
        this.sendTestCasesFromModule(filePath, testData);
        
        for (const child of testData) {
          if (child.type === 'test' && child.result?.state === 'failed') {
            status = "FAIL";
            failedTests.push({
              name: child.name,
              duration: child.result?.duration
            });
          }
        }
      }

      // testFileResult event removed - using group events instead
    }
  }
  
  // Helper method to send test case events from module children
  sendTestCasesFromModule(filePath, children, suiteName = null) {
    for (const child of children) {
      if (child.type === 'test') {
        // Send test case event
        const testStatus = child.result?.state === 'failed' ? 'FAIL' : 
                          child.result?.state === 'skipped' ? 'SKIP' : 'PASS';
        // Build error object if test failed
        let errorObj = null;
        if (child.result?.errors && child.result.errors.length > 0) {
          const firstError = child.result.errors[0];
          errorObj = {
            message: firstError.message || String(firstError),
            stack: firstError.stack || '',
            expected: firstError.expected || '',
            actual: firstError.actual || '',
            location: '',
            errorType: firstError.name || 'Error'
          };
        }

        const testCase = {
          eventType: "testCase",
          payload: {
            filePath,
            testName: child.name,
            suiteName: suiteName,
            status: testStatus,
            duration: child.result?.duration,
            error: errorObj
          }
        };
        
        this.logger.ipc("send", "testCase", testCase.payload);
        IPCSender.sendEvent(testCase).catch((error) => {
          this.logger.error("Failed to send testCase event", error);
        });
      } else if (child.type === 'suite' && child.children) {
        // Recursively process suite children
        this.sendTestCasesFromModule(filePath, child.children, child.name);
      }
    }
  }
  
  onTestRunEnd(testModules, unhandledErrors, reason) {
    this.logger.info("[V3] onTestRunEnd called", { 
      modules: testModules?.length || 0,
      errors: unhandledErrors?.length || 0,
      reason: reason 
    });
    this.logger.lifecycle("Test run complete (V3)", {
      modules: testModules?.length || 0,
      errors: unhandledErrors?.length || 0
    });
  }
  
  onHookStart(hook) {
    this.logger.debug("[V3] onHookStart called", { 
      type: hook?.type,
      name: hook?.name 
    });
  }
  
  onHookEnd(hook) {
    this.logger.debug("[V3] onHookEnd called", { 
      type: hook?.type,
      name: hook?.name 
    });
  }
  
  onTestAnnotate(testCase, annotation) {
    this.logger.debug("[V3] onTestAnnotate called", { 
      testName: testCase?.name,
      annotation: annotation 
    });
  }
  
  onTestFileStart(file) {
    this.logger.testFlow("Starting test file", file.filepath);
    this.currentTestFile = file.filepath;

    if (!this.filesStarted.has(file.filepath)) {
      this.filesStarted.add(file.filepath);

      // Discover the file as a root group and start it
      this.ensureGroupsDiscovered(file.filepath, []);
      this.ensureGroupStarted([file.filepath]);

      // Store file group info
      this.fileGroups.set(file.filepath, {
        startTime: Date.now(),
        tests: []
      });

      // testFileStart event removed - using group events instead
    }
    this.startCapture();
  }
  onTestFileResult(file) {
    if (!this.filesStarted.has(file.filepath)) {
      this.filesStarted.add(file.filepath);
      // testFileStart event removed - using group events instead
    }
    this.stopCapture();
    if (file.tasks) {
      this.sendTestCaseEvents(file.filepath, file.tasks);
    }
    let status = "PASS";
    if (file.result?.state === "fail") {
      status = "FAIL";
    } else if (file.result?.state === "skip" || file.mode === "skip") {
      status = "SKIP";
    }
    const testStats = file.result ? {
      tests: file.result.tests?.length || 0,
      duration: file.result.duration || 0,
      state: file.result.state
    } : {};
    
    // Collect failed tests for the payload (handle nested tasks)
    const failedTests = [];
    if (file.tasks) {
      const collectFailedTests = (tasks) => {
        for (const task of tasks) {
          if (task.type === "test" && task.result?.state === "fail") {
            failedTests.push({
              name: task.name,
              duration: task.result?.duration || 0
            });
          }
          // Recursively check nested tasks (suites)
          if (task.tasks && task.tasks.length > 0) {
            collectFailedTests(task.tasks);
          }
        }
      };
      collectFailedTests(file.tasks);
    }
    
    // Send GroupResult for the file
    const fileGroup = this.fileGroups.get(file.filepath);
    const fileDuration = fileGroup?.startTime ? Date.now() - fileGroup.startTime : undefined;

    const totals = {
      total: failedTests.length + (testStats.tests || 0),
      passed: (testStats.tests || 0) - failedTests.length,
      failed: failedTests.length,
      skipped: 0 // TODO: Extract from file.tasks if available
    };

    this.logger.testFlow("Test file completed", file.filepath, { status, ...testStats, failedTests: failedTests.length });
    this.logger.ipc("send", "testGroupResult", { groupName: file.filepath, status, totals });
    IPCSender.sendEvent({
      eventType: "testGroupResult",
      payload: {
        groupName: file.filepath,
        parentNames: [],
        status: status,
        duration: fileDuration,
        totals: totals
      }
    }).catch((error) => {
      this.logger.error("Failed to send testGroupResult", error);
    });

    // testFileResult event removed - using group events instead
    this.currentTestFile = null;
  }
  sendTestCaseEvents(filePath, tasks) {
    this.logger.debug("sendTestCaseEvents called", {
      filePath,
      taskCount: tasks.length,
      taskTypes: tasks.map(t => t.type)
    });
    for (const task of tasks) {
      if (task.type === "test") {
        const test = task;
        const suiteName = test.suite?.name;
        let status = "PASS";
        if (test.result?.state === "fail") {
          status = "FAIL";
        } else if (test.result?.state === "skip" || test.mode === "skip") {
          status = "SKIP";
        }
        // Build error object if test failed
        let error = null;
        if (test.result?.errors && test.result.errors.length > 0) {
          const firstError = test.result.errors[0];
          error = {
            message: typeof firstError === "string" ? firstError : (firstError.message || String(firstError)),
            stack: firstError.stack || '',
            expected: firstError.expected || '',
            actual: firstError.actual || '',
            location: '',
            errorType: firstError.name || 'Error'
          };
        }
        
        // Debug: Log the full test result object
        this.logger.debug("Test result details", {
          name: test.name,
          hasResult: !!test.result,
          resultKeys: test.result ? Object.keys(test.result) : [],
          duration: test.result?.duration,
          state: test.result?.state,
          fullResult: JSON.stringify(test.result)
        });
        
        // Extract hierarchy for this test case
        const suiteChain = this.extractHierarchyFromTask(test, filePath);
        const parentNames = this.buildHierarchyFromFile(filePath, suiteChain);

        // Ensure groups are discovered and started
        this.ensureGroupsDiscovered(filePath, suiteChain);

        // Start all parent groups
        for (let i = 0; i <= suiteChain.length; i++) {
          const hierarchy = [filePath, ...suiteChain.slice(0, i)];
          this.ensureGroupStarted(hierarchy);
        }

        this.logger.testFlow("Sending test case event", test.name, {
          suite: suiteName,
          status,
          duration: test.result?.duration,
          parentNames: parentNames
        });
        // Track test result for suite totals
        if (parentNames.length > 1) {
          const suiteName = parentNames[parentNames.length - 1];
          if (!this.suiteTestResults.has(suiteName)) {
            this.suiteTestResults.set(suiteName, { passed: 0, failed: 0, skipped: 0, total: 0 });
          }
          const results = this.suiteTestResults.get(suiteName);
          results.total++;
          if (status === 'PASS') results.passed++;
          else if (status === 'FAIL') results.failed++;
          else if (status === 'SKIP') results.skipped++;
        }

        IPCSender.sendEvent({
          eventType: "testCase",
          payload: {
            testName: test.name,
            parentNames: parentNames,
            status,
            duration: test.result?.duration,
            error
          }
        }).catch((error2) => {
          this.logger.error("Failed to send testCase event", error2);
        });
      } else if (task.type === "suite") {
        const suite = task;

        // Extract hierarchy for this suite
        const suiteChain = this.extractHierarchyFromTask(suite, filePath);
        const parentNames = this.buildHierarchyFromFile(filePath, suiteChain.slice(0, -1)); // Remove the suite itself from parents

        // Send group start event for the suite
        this.ensureGroupsDiscovered(filePath, suiteChain);
        this.ensureGroupStarted([filePath, ...suiteChain]);

        if (suite.tasks) {
          this.sendTestCaseEvents(filePath, suite.tasks);
        }

        // Send group result event for the suite
        // Calculate suite status based on test results
        // The tasks have already been processed and sent, so their state might be removed
        // Instead, count from what we know we sent
        // For suites, we need to count based on the actual test results we already sent
        // This is a limitation when processing in onFinished fallback mode
        const failedCount = 0;  // Will be calculated from actual test results
        const skippedCount = 0;
        const passedCount = 0;
        const totalCount = suite.tasks ? suite.tasks.length : 0;

        // Debug counting
        this.logger.debug("Suite task counting", {
          suiteName: suite.name,
          failedCount,
          skippedCount,
          passedCount,
          totalCount,
          hasTasks: !!suite.tasks,
          tasksLength: suite.tasks?.length
        });

        let suiteStatus = 'PASS';
        if (failedCount > 0) {
          suiteStatus = 'FAIL';
        } else if (skippedCount === totalCount && totalCount > 0) {
          suiteStatus = 'SKIP';
        }
        const suiteDuration = suite.result?.duration || 0;
        const suiteTotals = {
          total: suite.tasks ? this.countTests(suite.tasks) : 0,
          passed: suite.tasks ? this.countTestsByStatus(suite.tasks, 'passed') : 0,
          failed: suite.tasks ? this.countTestsByStatus(suite.tasks, 'failed') : 0,
          skipped: suite.tasks ? this.countTestsByStatus(suite.tasks, 'skipped') : 0
        };

        IPCSender.sendEvent({
          eventType: "testGroupResult",
          payload: {
            groupName: suite.name,
            parentNames: parentNames,
            status: suiteStatus,
            duration: suiteDuration,
            totals: suiteTotals
          }
        }).catch((error) => {
          this.logger.error("Failed to send testGroupResult for suite", error);
        });
      }
    }
  }
  async onFinished(files, errors) {
    this.logger.lifecycle("Test run finishing", {
      files: files?.length || 0,
      errors: errors?.length || 0
    });
    this.stopCapture();
    if (files && files.length > 0) {
      this.logger.info("Processing files in onFinished (fallback mode)", { count: files.length });
      for (const file of files) {
        if (!this.filesStarted.has(file.filepath)) {
          this.filesStarted.add(file.filepath);
          // testFileStart event removed - using group events instead
        }
        if (file.tasks) {
          // In onFinished, file.tasks typically contains one suite representing the describe block
          // Check if we have a single suite that contains all tests
          if (file.tasks.length === 1 && file.tasks[0].type === 'suite') {
            // Send suite-level group events
            const suite = file.tasks[0];
            const suiteHierarchy = [file.filepath, suite.name];

            // Ensure suite is discovered and started
            this.ensureGroupsDiscovered(file.filepath, [suite.name]);
            this.ensureGroupStarted(suiteHierarchy);

            // Process the tests in the suite
            if (suite.tasks) {
              this.sendTestCaseEvents(file.filepath, suite.tasks);
            }

            // Send suite result based on tracked test results
            const trackedResults = this.suiteTestResults.get(suite.name) || { passed: 0, failed: 0, skipped: 0, total: 0 };
            let suiteStatus = 'PASS';
            if (trackedResults.failed > 0) {
              suiteStatus = 'FAIL';
            } else if (trackedResults.skipped === trackedResults.total && trackedResults.total > 0) {
              suiteStatus = 'SKIP';
            }
            const suiteDuration = suite.result?.duration || 0;

            const suiteTotals = {
              total: trackedResults.total,
              passed: trackedResults.passed,
              failed: trackedResults.failed,
              skipped: trackedResults.skipped
            };

            IPCSender.sendEvent({
              eventType: "testGroupResult",
              payload: {
                groupName: suite.name,
                parentNames: [file.filepath],
                status: suiteStatus,
                duration: suiteDuration,
                totals: suiteTotals
              }
            }).catch((error) => {
              this.logger.error("Failed to send testGroupResult for suite", error);
            });
          } else {
            // Multiple suites or direct tests
            this.sendTestCaseEvents(file.filepath, file.tasks);
          }
        }
        let status = "PASS";
        if (file.result?.state === "fail") {
          status = "FAIL";
        } else if (file.result?.state === "skip" || file.mode === "skip") {
          status = "SKIP";
        }
        // Collect failed tests for the payload (handle nested tasks)
        const failedTests = [];
        if (file.tasks) {
          const collectFailedTests = (tasks) => {
            for (const task of tasks) {
              if (task.type === "test" && task.result?.state === "fail") {
                failedTests.push({
                  name: task.name,
                  duration: task.result?.duration
                });
              }
              // Recursively check nested tasks (suites)
              if (task.tasks && task.tasks.length > 0) {
                collectFailedTests(task.tasks);
              }
            }
          };
          collectFailedTests(file.tasks);
        }

        // Send group start event for the file
        this.ensureGroupStarted([file.filepath]);

        // Send group result event for the file
        const fileDuration = file.result?.duration || 0;

        // Calculate file totals from all tracked suite results
        let fileTotals = { total: 0, passed: 0, failed: 0, skipped: 0 };

        // If we have a single suite, use its totals
        if (file.tasks && file.tasks.length === 1 && file.tasks[0].type === 'suite') {
          const suite = file.tasks[0];
          const trackedResults = this.suiteTestResults.get(suite.name);
          if (trackedResults) {
            fileTotals = trackedResults;
          }
        } else {
          // Otherwise count all tests recursively
          fileTotals = {
            total: file.tasks ? this.countTests(file.tasks) : 0,
            passed: file.tasks ? this.countTestsByStatus(file.tasks, 'passed') : 0,
            failed: file.tasks ? this.countTestsByStatus(file.tasks, 'failed') : 0,
            skipped: file.tasks ? this.countTestsByStatus(file.tasks, 'skipped') : 0
          };
        }

        const totals = fileTotals;

        this.logger.ipc("send", "testGroupResult", { groupName: file.filepath, status, totals });
        IPCSender.sendEvent({
          eventType: "testGroupResult",
          payload: {
            groupName: file.filepath,
            parentNames: [],
            status: status,
            duration: fileDuration,
            totals: totals
          }
        }).catch((error) => {
          this.logger.error("Failed to send testGroupResult", error);
        });
      }
    }
    this.logger.lifecycle("Vitest adapter shutdown complete");
  }
  startCapture() {
    if (this.captureEnabled) return;
    this.captureEnabled = true;
    this.logger.debug("Starting stdout/stderr capture", { currentFile: this.currentTestFile });
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
    this.logger.debug("Stopping stdout/stderr capture");
    process.stdout.write = this.originalStdoutWrite;
    process.stderr.write = this.originalStderrWrite;
  }

  // Helper method to count total tests in a task tree
  countTests(tasks) {
    let count = 0;
    for (const task of tasks) {
      if (task.type === "test") {
        count++;
      } else if (task.tasks) {
        count += this.countTests(task.tasks);
      }
    }
    return count;
  }

  // Helper method to count tests by status in a task tree
  countTestsByStatus(tasks, status) {
    let count = 0;
    for (const task of tasks) {
      if (task.type === "test") {
        // Debug: Log what we're checking
        if (task.name && task.name.includes("fail")) {
          this.logger.debug("Checking test status", {
            name: task.name,
            hasResult: !!task.result,
            state: task.result?.state,
            lookingFor: status,
            matches: task.result?.state === status
          });
        }
        if (task.result?.state === status) {
          count++;
        }
      } else if (task.tasks) {
        count += this.countTestsByStatus(task.tasks, status);
      }
    }
    return count;
  }
};
export {
  ThreePioVitestReporter as default
};
//# sourceMappingURL=vitest.js.map
