/**
 * 3pio Cypress Adapter (Mocha reporter)
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

// Cypress reporter (Mocha reporter API)
function ThreePioCypressReporter(runner /*, options */) {
  // Track per-spec statistics
  let specFile = null;
  let startedAt = 0;
  let passed = 0, failed = 0, skipped = 0;

  function resolveSpecFrom(obj) {
    if (!obj) return null;
    // Try common locations where Mocha exposes the loaded file
    if (obj.file) return obj.file;
    if (obj.parent && obj.parent.file) return obj.parent.file;
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

  runner.on('start', () => {
    startedAt = now();
  });

  runner.on('suite', (suite) => {
    // Ignore root suite (empty title)
    if (!suite || !suite.title) return;
    if (!specFile) specFile = resolveSpecFrom(suite) || specFile;
    const chain = toChain(suite);
    if (specFile) {
      ensureDiscovered(specFile, chain.concat([suite.title]));
      ensureStarted(specFile, chain.concat([suite.title]));
    }
  });

  runner.on('test', (test) => {
    if (!specFile) specFile = resolveSpecFrom(test) || specFile;
    const chain = toChain(test); // excludes the test title itself
    if (specFile) {
      ensureDiscovered(specFile, chain);
      ensureStarted(specFile, []); // ensure file-level started for RUNNING
    }
  });

  runner.on('pass', (test) => {
    passed++;
    emitTestCase(test, 'pass');
  });

  runner.on('fail', (test, err) => {
    failed++;
    emitTestCase(test, 'fail', err);
  });

  runner.on('pending', (test) => {
    skipped++;
    emitTestCase(test, 'pending');
  });

  function emitTestCase(test, kind, err) {
    if (!specFile) specFile = resolveSpecFrom(test) || 'unknown.spec';
    const chain = toChain(test);
    const duration = typeof test.duration === 'number' ? test.duration : 0;

    // Ensure discovery for all parent groups
    ensureDiscovered(specFile, chain);

    const payload = {
      testName: test.title || 'Unnamed test',
      parentNames: [specFile, ...chain],
      status: statusFrom(kind),
      duration,
    };
    if (err) {
      payload.error = {
        message: String(err && (err.message || err)) || 'Error',
        stack: err && err.stack || '',
        errorType: err && err.name || 'Error',
      };
    }

    sendEvent({ eventType: 'testCase', payload });
  }

  runner.once('end', () => {
    const endedAt = now();
    const total = passed + failed + skipped;
    const status = failed > 0 ? 'FAIL' : (passed > 0 && failed === 0 ? 'PASS' : (skipped > 0 ? 'SKIP' : 'PASS'));
    const duration = endedAt - startedAt;

    const file = specFile || 'unknown.spec';
    // Ensure file group at least
    ensureDiscovered(file, []);
    ensureStarted(file, []);

    sendEvent({
      eventType: 'testGroupResult',
      payload: {
        groupName: file,
        parentNames: [],
        status,
        duration,
        totals: { passed, failed, skipped, total },
      },
    });

    // End of run (optional, used by manager)
    sendEvent({ eventType: 'runComplete', payload: {} });
  });
}

module.exports = ThreePioCypressReporter;

