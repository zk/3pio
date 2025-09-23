/**
 * 3pio Mocha Adapter (Custom Reporter)
 * Emits hierarchical group/test events to THREEPIO_IPC_PATH.
 * Silent by design: no stdout/stderr logs.
 */

/* eslint-disable */
const fs = require('fs');
const path = require('path');

// Runtime-injected values from Go embedder
const IPC_PATH = /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;
const LOG_LEVEL = /*__LOG_LEVEL__*/"WARN"/*__LOG_LEVEL__*/;

function now() { return Date.now(); }

function safeAppend(line) {
  try {
    const dir = path.dirname(IPC_PATH);
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
    fs.appendFileSync(IPC_PATH, line + '\n');
  } catch (_) {
    // intentionally silent
  }
}

function sendEvent(event) {
  safeAppend(JSON.stringify(event));
}

// Basic discovered/start trackers to avoid duplicates
const discovered = new Set();
const started = new Set();

// Disambiguation for duplicate suite titles under the same parent
// Map Mocha Suite object -> disambiguated title (e.g., "in router #2")
const suiteNameMap = new WeakMap();
// Map parent Suite object -> Map<title, count>
const titleCounters = new WeakMap();

function groupId(hierarchy) { return hierarchy.join(':'); }

function discoverHierarchy(filePath, suiteChain) {
  const groups = [];
  groups.push({ name: filePath, parentNames: [] });
  if (suiteChain && suiteChain.length > 0) {
    for (let i = 0; i < suiteChain.length; i++) {
      const parent = [filePath, ...suiteChain.slice(0, i)];
      groups.push({ name: suiteChain[i], parentNames: parent });
    }
  }
  return groups;
}

function ensureDiscovered(filePath, suiteChain) {
  const groups = discoverHierarchy(filePath, suiteChain);
  for (const g of groups) {
    const id = groupId([...g.parentNames, g.name]);
    if (!discovered.has(id)) {
      discovered.add(id);
      sendEvent({
        eventType: 'testGroupDiscovered',
        payload: { groupName: g.name, parentNames: g.parentNames }
      });
    }
  }
}

function ensureStarted(filePath, suiteChain) {
  const id = groupId([...(suiteChain ? [filePath, ...suiteChain] : [filePath])]);
  if (!started.has(id)) {
    started.add(id);
    const name = suiteChain && suiteChain.length > 0 ? suiteChain[suiteChain.length - 1] : filePath;
    const parentNames = suiteChain && suiteChain.length > 0 ? [filePath, ...suiteChain.slice(0, -1)] : [];
    sendEvent({
      eventType: 'testGroupStart',
      payload: { groupName: name, parentNames }
    });
  }
}

// Build a disambiguated chain of suite titles for a test or suite
function toDisambiguatedChain(node) {
  const chain = [];
  let cur = node.parent;
  while (cur && !cur.root) {
    const name = suiteNameMap.get(cur) || cur.title || '';
    if (name) chain.unshift(name);
    cur = cur.parent;
  }
  return chain;
}

// Mocha reporter API
function ThreePioMochaReporter(runner /*, options */) {
  // Track per-file statistics to support multi-file runs accurately
  const fileStats = new Map(); // filePath -> { startedAt, passed, failed, skipped }
  let currentFile = null;

  function resolveSpecFrom(obj) {
    if (!obj) return null;
    // Mocha commonly exposes file on test/suite
    if (obj.file) return obj.file;
    if (obj.parent && obj.parent.file) return obj.parent.file;
    // Fallbacks
    if (obj.parent && obj.parent.root && obj.parent.suites && obj.parent.suites[0] && obj.parent.suites[0].file) {
      return obj.parent.suites[0].file;
    }
    return null;
  }

  function toChain(testOrSuite) {
    const chain = [];
    let node = testOrSuite.parent; // Exclude the test/suite itself
    while (node && !node.root) {
      if (node.title) chain.unshift(node.title);
      node = node.parent;
    }
    return chain;
  }

  function statusFrom(type) {
    if (type === 'pass') return 'PASS';
    if (type === 'fail') return 'FAIL';
    if (type === 'pending') return 'SKIP';
    return 'PENDING';
  }

  let runStartedAt = 0;
  runner.on('start', () => {
    runStartedAt = now();
  });

  runner.on('suite', (suite) => {
    // Ignore root suite (empty title)
    if (!suite || !suite.title) return;
    const specFile = resolveSpecFrom(suite);
    // Disambiguate this suite title under its parent
    const parent = suite.parent;
    if (parent) {
      let counter = titleCounters.get(parent);
      if (!counter) { counter = new Map(); titleCounters.set(parent, counter); }
      const prev = counter.get(suite.title) || 0;
      const next = prev + 1;
      counter.set(suite.title, next);
      const disName = next > 1 ? `${suite.title} #${next}` : suite.title;
      suiteNameMap.set(suite, disName);
    }
    const chain = toDisambiguatedChain(suite);
    if (specFile) {
      currentFile = specFile;
      // Ensure file root discovered/started
      ensureDiscovered(specFile, []);
      ensureStarted(specFile, []);
      // Ensure suite hierarchy discovered/started
      const thisName = suiteNameMap.get(suite) || suite.title;
      ensureDiscovered(specFile, chain.concat([thisName]));
      ensureStarted(specFile, chain.concat([thisName]));
      // Initialize per-file stats
      if (!fileStats.has(specFile)) {
        fileStats.set(specFile, { startedAt: now(), passed: 0, failed: 0, skipped: 0 });
      }
    }
  });

  runner.on('test', (test) => {
    const specFile = resolveSpecFrom(test) || currentFile;
    const chain = toDisambiguatedChain(test); // excludes the test title itself
    if (specFile) {
      currentFile = specFile;
      // Ensure file and parent groups exist and are started
      ensureDiscovered(specFile, chain);
      ensureStarted(specFile, []);
      if (!fileStats.has(specFile)) {
        fileStats.set(specFile, { startedAt: now(), passed: 0, failed: 0, skipped: 0 });
      }
    }
  });

  runner.on('pass', (test) => {
    const specFile = resolveSpecFrom(test) || currentFile;
    if (specFile) {
      const stats = fileStats.get(specFile) || { startedAt: now(), passed: 0, failed: 0, skipped: 0 };
      stats.passed += 1;
      fileStats.set(specFile, stats);
    }
    emitTestCase(test, 'pass');
  });

  runner.on('fail', (test, err) => {
    const specFile = resolveSpecFrom(test) || currentFile;
    if (specFile) {
      const stats = fileStats.get(specFile) || { startedAt: now(), passed: 0, failed: 0, skipped: 0 };
      stats.failed += 1;
      fileStats.set(specFile, stats);
    }
    emitTestCase(test, 'fail', err);
  });

  runner.on('pending', (test) => {
    const specFile = resolveSpecFrom(test) || currentFile;
    if (specFile) {
      const stats = fileStats.get(specFile) || { startedAt: now(), passed: 0, failed: 0, skipped: 0 };
      stats.skipped += 1;
      fileStats.set(specFile, stats);
    }
    emitTestCase(test, 'pending');
  });

  function emitTestCase(test, kind, err) {
    const testFile = resolveSpecFrom(test) || specFile || 'unknown.spec';
    const chain = toDisambiguatedChain(test);
    const duration = typeof test.duration === 'number' ? test.duration : 0;

    // Ensure discovery for all parent groups
    ensureDiscovered(testFile, chain);

    const payload = {
      testName: test.title || 'Unnamed test',
      parentNames: [testFile, ...chain],
      status: statusFrom(kind),
      duration,
    };
    if (err) {
      payload.error = {
        message: String(err && (err.message || err)) || 'Error',
        stack: (err && err.stack) || '',
        errorType: (err && err.name) || 'Error',
      };
    }

    sendEvent({ eventType: 'testCase', payload });
  }

  runner.once('end', () => {
    // Emit a file-level result for every file we saw
    for (const [file, stats] of fileStats.entries()) {
      const total = (stats.passed || 0) + (stats.failed || 0) + (stats.skipped || 0);
      const status = (stats.failed || 0) > 0
        ? 'FAIL'
        : (stats.passed || 0) > 0 && (stats.failed || 0) === 0
          ? 'PASS'
          : (stats.skipped || 0) > 0 ? 'SKIP' : 'PASS';
      const duration = Math.max(0, now() - (stats.startedAt || now()));

      // Ensure file group is discovered/started
      ensureDiscovered(file, []);
      ensureStarted(file, []);

      sendEvent({
        eventType: 'testGroupResult',
        payload: {
          groupName: file,
          parentNames: [],
          status,
          duration,
          totals: { passed: stats.passed || 0, failed: stats.failed || 0, skipped: stats.skipped || 0, total },
        },
      });
    }

    // End of run (optional, used by manager)
    sendEvent({ eventType: 'runComplete', payload: {} });
  });
}

module.exports = ThreePioMochaReporter;
