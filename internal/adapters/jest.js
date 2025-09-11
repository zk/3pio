"use strict";
var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __commonJS = (cb, mod) => function __require() {
  return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
};
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
  // If the importer is in node compatibility mode or this is not an ESM
  // file that has been converted to a CommonJS file using a Babel-
  // compatible transform (i.e. "__esModule" has not been set), then set
  // "default" to the CommonJS "module.exports" for node compatibility.
  isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
  mod
));
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

// package.json
var require_package = __commonJS({
  "package.json"(exports2, module2) {
    module2.exports = {
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

// src/adapters/jest.ts
var jest_exports = {};
__export(jest_exports, {
  default: () => ThreePioJestReporter
});
module.exports = __toCommonJS(jest_exports);

// src/ipc-sender.ts
var fs = __toESM(require("fs"));
var path = __toESM(require("path"));
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
    const ipcPath = /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;
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
var fs2 = __toESM(require("fs"));
var path2 = __toESM(require("path"));
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
    if (process.env.THREEPIO_DEBUG === "1") {
      this.writeLog("DEBUG", message, data);
    }
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

// src/adapters/jest.ts
var packageJson = require_package();
var ThreePioJestReporter = class {
  originalStdoutWrite;
  originalStderrWrite;
  currentTestFile = null;
  captureEnabled = false;
  logger;
  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
    this.logger = Logger.create("jest-adapter");
    this.logger.startupPreamble([
      "==================================",
      `3pio Jest Adapter v${packageJson.version}`,
      "Configuration:",
      `  - IPC Path: ${/*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/}`,
      `  - Process ID: ${process.pid}`,
      "=================================="
    ]);
  }
  onRunStart() {
    this.logger.lifecycle("Test run starting");
    const ipcPath = /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;
    this.logger.info("IPC communication channel ready", { path: ipcPath });
    this.logger.initComplete({ ipcPath });
  }
  onTestCaseStart(test, testCaseStartInfo) {
    if (testCaseStartInfo?.ancestorTitles && testCaseStartInfo?.title) {
      const suiteName = testCaseStartInfo.ancestorTitles.join(" \u203A ");
      const testName = testCaseStartInfo.title;
      this.logger.testFlow("Test case starting", testName, { suite: suiteName });
      IPCSender.sendEvent({
        eventType: "testCase",
        payload: {
          filePath: test.path,
          testName,
          suiteName: suiteName || void 0,
          status: "RUNNING"
        }
      }).catch((error) => {
        this.logger.error("Failed to send testCase start", error);
      });
    }
  }
  onTestCaseResult(test, testCaseResult) {
    if (testCaseResult) {
      const suiteName = testCaseResult.ancestorTitles?.join(" \u203A ");
      const testName = testCaseResult.title;
      let status = "PASS";
      if (testCaseResult.status === "failed") {
        status = "FAIL";
      } else if (testCaseResult.status === "skipped" || testCaseResult.status === "pending") {
        status = "SKIP";
      }
      const error = testCaseResult.failureMessages?.join("\n\n");
      this.logger.testFlow("Test case completed", testName, {
        suite: suiteName,
        status,
        duration: testCaseResult.duration
      });
      IPCSender.sendEvent({
        eventType: "testCase",
        payload: {
          filePath: test.path,
          testName,
          suiteName: suiteName || void 0,
          status,
          duration: testCaseResult.duration,
          error
        }
      }).catch((error2) => {
        this.logger.error("Failed to send testCase result", error2);
      });
    }
  }
  onTestStart(test) {
    this.logger.testFlow("Starting test file", test.path);
    this.currentTestFile = test.path;
    this.logger.ipc("send", "testFileStart", { filePath: test.path });
    IPCSender.sendEvent({
      eventType: "testFileStart",
      payload: {
        filePath: test.path
      }
    }).catch((error) => {
      this.logger.error("Failed to send testFileStart", error);
    });
    this.startCapture();
  }
  onTestResult(test, testResult, aggregatedResult) {
    this.stopCapture();
    if (testResult.console && testResult.console.length > 0) {
      this.logger.info("Console output found in testResult!", {
        consoleLength: testResult.console.length
      });
      for (const log of testResult.console) {
        const chunk = `${log.message}
`;
        IPCSender.sendEvent({
          eventType: log.type === "error" ? "stderrChunk" : "stdoutChunk",
          payload: {
            filePath: test.path,
            chunk
          }
        }).catch(() => {
        });
      }
    }
    const status = testResult.numFailingTests > 0 ? "FAIL" : testResult.skipped ? "SKIP" : "PASS";
    if (testResult.testResults) {
      for (const testCase of testResult.testResults) {
        const suiteName = testCase.ancestorTitles?.join(" \u203A ");
        const testName = testCase.title;
        let testStatus = "PASS";
        if (testCase.status === "failed") {
          testStatus = "FAIL";
        } else if (testCase.status === "skipped" || testCase.status === "pending") {
          testStatus = "SKIP";
        }
        const error = testCase.failureMessages?.join("\n\n");
        IPCSender.sendEvent({
          eventType: "testCase",
          payload: {
            filePath: test.path,
            testName,
            suiteName: suiteName || void 0,
            status: testStatus,
            duration: testCase.duration,
            error
          }
        }).catch(() => {
        });
      }
    }
    const failedTests = [];
    if (testResult.testResults && status === "FAIL") {
      for (const suite of testResult.testResults) {
        if (suite.status !== "passed") {
          const fullName = suite.ancestorTitles.length > 0 ? `${suite.ancestorTitles.join(" \u203A ")} \u203A ${suite.title}` : suite.title;
          failedTests.push({
            name: fullName,
            duration: suite.duration
          });
        }
      }
    }
    this.logger.testFlow("Test file completed", test.path, {
      status,
      failures: testResult.numFailingTests,
      tests: testResult.numPassedTests + testResult.numFailingTests,
      passed: testResult.numPassedTests
    });
    this.logger.ipc("send", "testFileResult", { filePath: test.path, status });
    IPCSender.sendEvent({
      eventType: "testFileResult",
      payload: {
        filePath: test.path,
        status,
        failedTests: failedTests.length > 0 ? failedTests : void 0
      }
    }).catch((error) => {
      this.logger.error("Failed to send testFileResult", error);
    });
    this.currentTestFile = null;
  }
  onRunComplete(testContexts, results) {
    this.logger.lifecycle("Test run completing", {
      totalSuites: results.numTotalTestSuites,
      failedSuites: results.numFailedTestSuites,
      passedSuites: results.numPassedTestSuites,
      totalTests: results.numTotalTests,
      passedTests: results.numPassedTests,
      failedTests: results.numFailedTests
    });
    this.stopCapture();
    const syncFs = require("fs");
    const ipcPath = /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;
    {
      try {
        this.logger.ipc("send", "runComplete", {});
        syncFs.appendFileSync(ipcPath, JSON.stringify({
          eventType: "runComplete",
          payload: {}
        }) + "\n", "utf8");
        this.logger.info("Run completion marker sent");
      } catch (error) {
        this.logger.error("Failed to write runComplete marker", error);
      }
    }
    this.logger.lifecycle("Jest adapter shutdown complete");
  }
  startCapture() {
    if (this.captureEnabled) return;
    this.captureEnabled = true;
    this.logger.debug("Starting stdout/stderr capture for", { file: this.currentTestFile });
    process.stdout.write = (chunk, ...args) => {
      const chunkStr = chunk.toString();
      if (this.currentTestFile) {
        IPCSender.sendEvent({
          eventType: "stdoutChunk",
          payload: {
            filePath: this.currentTestFile,
            chunk: chunkStr
          }
        }).catch(() => {
        });
      }
      return true;
    };
    process.stderr.write = (chunk, ...args) => {
      const chunkStr = chunk.toString();
      if (this.currentTestFile) {
        IPCSender.sendEvent({
          eventType: "stderrChunk",
          payload: {
            filePath: this.currentTestFile,
            chunk: chunkStr
          }
        }).catch(() => {
        });
      }
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
  getLastError() {
  }
};
//# sourceMappingURL=jest.js.map
