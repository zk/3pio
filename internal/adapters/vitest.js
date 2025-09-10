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
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    if (!ipcPath) {
      console.error("THREEPIO_IPC_PATH not set");
      return;
    }
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

// src/adapters/vitest.ts
var packageJson = require_package();
var ThreePioVitestReporter = class {
  originalStdoutWrite;
  originalStderrWrite;
  currentTestFile = null;
  captureEnabled = false;
  logger;
  filesStarted = /* @__PURE__ */ new Set();
  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
    this.logger = Logger.create("vitest-adapter");
    this.logger.startupPreamble([
      "==================================",
      `3pio Vitest Adapter v${packageJson.version}`,
      "Configuration:",
      `  - IPC Path: ${process.env.THREEPIO_IPC_PATH || "not set"}`,
      `  - Process ID: ${process.pid}`,
      "=================================="
    ]);
  }
  onInit(ctx) {
    this.logger.lifecycle("Test run initializing");
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    if (!ipcPath) {
      this.logger.error("THREEPIO_IPC_PATH not set - adapter cannot function");
    } else {
      this.logger.info("IPC communication channel ready", { path: ipcPath });
    }
    this.logger.initComplete({ ipcPath });
    this.logger.debug("Starting global capture for test output");
    this.startCapture();
  }
  onPathsCollected(paths) {
    this.logger.info("Test paths collected", { count: paths?.length || 0 });
  }
  onCollected(files) {
    this.logger.info("Test files collected", { count: files?.length || 0 });
  }
  onTestFileStart(file) {
    this.logger.testFlow("Starting test file", file.filepath);
    this.currentTestFile = file.filepath;
    if (!this.filesStarted.has(file.filepath)) {
      this.filesStarted.add(file.filepath);
      this.logger.ipc("send", "testFileStart", { filePath: file.filepath });
      IPCSender.sendEvent({
        eventType: "testFileStart",
        payload: {
          filePath: file.filepath
        }
      }).catch((error) => {
        this.logger.error("Failed to send testFileStart", error);
      });
    }
    this.startCapture();
  }
  onTestFileResult(file) {
    if (!this.filesStarted.has(file.filepath)) {
      this.filesStarted.add(file.filepath);
      this.logger.ipc("send", "testFileStart", { filePath: file.filepath });
      IPCSender.sendEvent({
        eventType: "testFileStart",
        payload: {
          filePath: file.filepath
        }
      }).catch((error) => {
        this.logger.error("Failed to send testFileStart", error);
      });
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
    this.logger.testFlow("Test file completed", file.filepath, { status, ...testStats });
    this.logger.ipc("send", "testFileResult", { filePath: file.filepath, status });
    IPCSender.sendEvent({
      eventType: "testFileResult",
      payload: {
        filePath: file.filepath,
        status
      }
    }).catch((error) => {
      this.logger.error("Failed to send testFileResult", error);
    });
    this.currentTestFile = null;
  }
  sendTestCaseEvents(filePath, tasks) {
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
        const error = test.result?.errors?.map(
          (e) => typeof e === "string" ? e : e.message || String(e)
        ).join("\n\n");
        this.logger.testFlow("Sending test case event", test.name, {
          suite: suiteName,
          status,
          duration: test.result?.duration
        });
        IPCSender.sendEvent({
          eventType: "testCase",
          payload: {
            filePath,
            testName: test.name,
            suiteName,
            status,
            duration: test.result?.duration,
            error
          }
        }).catch((error2) => {
          this.logger.error("Failed to send testCase event", error2);
        });
      } else if (task.type === "suite") {
        const suite = task;
        if (suite.tasks) {
          this.sendTestCaseEvents(filePath, suite.tasks);
        }
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
          this.logger.ipc("send", "testFileStart", { filePath: file.filepath });
          try {
            await IPCSender.sendEvent({
              eventType: "testFileStart",
              payload: {
                filePath: file.filepath
              }
            });
          } catch (error) {
            this.logger.error("Failed to send testFileStart", error);
          }
        }
        if (file.tasks) {
          this.sendTestCaseEvents(file.filepath, file.tasks);
        }
        let status = "PASS";
        if (file.result?.state === "fail") {
          status = "FAIL";
        } else if (file.result?.state === "skip" || file.mode === "skip") {
          status = "SKIP";
        }
        this.logger.debug("Sending deferred test result", { file: file.filepath, status });
        try {
          this.logger.ipc("send", "testFileResult", { filePath: file.filepath, status });
          await IPCSender.sendEvent({
            eventType: "testFileResult",
            payload: {
              filePath: file.filepath,
              status
            }
          });
        } catch (error) {
          this.logger.error("Failed to send deferred test result", error, { file: file.filepath });
        }
      }
    }
    this.logger.lifecycle("Vitest adapter shutdown complete");
  }
  startCapture() {
    if (this.captureEnabled) return;
    this.captureEnabled = true;
    this.logger.debug("Starting stdout/stderr capture", { currentFile: this.currentTestFile });
    process.stdout.write = (chunk, ...args) => {
      if (chunk) {
        const chunkStr = chunk.toString();
        const filePath = this.currentTestFile;
        if (!filePath) return true;
        IPCSender.sendEvent({
          eventType: "stdoutChunk",
          payload: {
            filePath,
            chunk: chunkStr
          }
        }).catch(() => {
        });
      }
      return true;
    };
    process.stderr.write = (chunk, ...args) => {
      if (chunk) {
        const chunkStr = chunk.toString();
        const filePath = this.currentTestFile;
        if (!filePath) return true;
        IPCSender.sendEvent({
          eventType: "stderrChunk",
          payload: {
            filePath,
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
};
export {
  ThreePioVitestReporter as default
};
//# sourceMappingURL=vitest.js.map
