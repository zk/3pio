/**
 * 3pio Jest Adapter with Group Events
 * Emits hierarchical group events instead of file-centric events
 */

const fs = require('fs');
const path = require('path');

// IPC Path will be replaced at runtime
const IPC_PATH = /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;

// Track discovered groups to avoid duplicates
const discoveredGroups = new Map();
const groupStarts = new Map();
const fileGroups = new Map();

/**
 * Build hierarchy from file path and ancestor titles
 */
function buildHierarchy(filePath, ancestorTitles) {
  const hierarchy = [filePath];
  if (ancestorTitles && ancestorTitles.length > 0) {
    hierarchy.push(...ancestorTitles);
  }
  return hierarchy;
}

/**
 * Generate a unique ID for a group path
 */
function getGroupId(hierarchy) {
  return hierarchy.join(':');
}

/**
 * Send event to IPC file
 */
function sendEvent(event) {
  try {
    const dir = path.dirname(IPC_PATH);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
    const line = JSON.stringify(event) + '\n';
    fs.appendFileSync(IPC_PATH, line);
  } catch (error) {
    // Silent failure - adapters should not write to stdout/stderr
  }
}

/**
 * Discover all groups in a hierarchy
 */
function discoverGroups(filePath, ancestorTitles) {
  const groups = [];
  
  // First, the file itself is a group
  groups.push({
    hierarchy: [filePath],
    name: filePath,
    parentNames: []
  });
  
  // Then each level of ancestorTitles creates a nested group
  if (ancestorTitles && ancestorTitles.length > 0) {
    for (let i = 0; i < ancestorTitles.length; i++) {
      const parentNames = [filePath, ...ancestorTitles.slice(0, i)];
      const groupName = ancestorTitles[i];
      groups.push({
        hierarchy: [...parentNames, groupName],
        name: groupName,
        parentNames: parentNames
      });
    }
  }
  
  return groups;
}

/**
 * Send GroupDiscovered events for new groups
 */
function ensureGroupsDiscovered(filePath, ancestorTitles) {
  const groups = discoverGroups(filePath, ancestorTitles);
  
  for (const group of groups) {
    const groupId = getGroupId(group.hierarchy);
    if (!discoveredGroups.has(groupId)) {
      discoveredGroups.set(groupId, group);
      sendEvent({
        eventType: 'testGroupDiscovered',
        payload: {
          groupName: group.name,
          parentNames: group.parentNames
        }
      });
    }
  }
}

/**
 * Send GroupStart event if not already started
 */
function ensureGroupStarted(hierarchy) {
  const groupId = getGroupId(hierarchy);
  if (!groupStarts.has(groupId)) {
    groupStarts.set(groupId, Date.now());
    
    const group = discoveredGroups.get(groupId);
    if (group) {
      sendEvent({
        eventType: 'testGroupStart',
        payload: {
          groupName: group.name,
          parentNames: group.parentNames
        }
      });
    }
  }
}

class ThreePioJestReporter {
  originalStdoutWrite;
  originalStderrWrite;
  currentTestFile = null;
  captureEnabled = false;
  testSuiteStats = new Map(); // Track stats per test suite

  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
  }

  onRunStart() {
    // Collection phase for Jest (Jest doesn't have separate collection)
    sendEvent({
      eventType: 'collectionStart',
      payload: { phase: 'collection' }
    });
  }

  onTestStart(test) {
    this.currentTestFile = test.path;
    
    // Discover the file as a root group
    ensureGroupsDiscovered(test.path, []);
    
    // Start the file group
    ensureGroupStarted([test.path]);
    
    // Store file group info
    fileGroups.set(test.path, {
      startTime: Date.now(),
      tests: []
    });
    
    // Start output capture
    this.startCapture();
  }

  onTestCaseStart(test, testCaseStartInfo) {
    if (testCaseStartInfo?.ancestorTitles && testCaseStartInfo?.title) {
      // Ensure all parent groups are discovered
      ensureGroupsDiscovered(test.path, testCaseStartInfo.ancestorTitles);
      
      // Start all parent groups
      if (testCaseStartInfo.ancestorTitles.length > 0) {
        for (let i = 0; i <= testCaseStartInfo.ancestorTitles.length; i++) {
          const hierarchy = [test.path, ...testCaseStartInfo.ancestorTitles.slice(0, i)];
          ensureGroupStarted(hierarchy);
        }
      }
    }
  }

  onTestCaseResult(test, testCaseResult) {
    if (testCaseResult) {
      const parentNames = [test.path, ...(testCaseResult.ancestorTitles || [])];
      const testName = testCaseResult.title;
      
      let status = 'PASS';
      if (testCaseResult.status === 'failed') {
        status = 'FAIL';
      } else if (testCaseResult.status === 'skipped' || testCaseResult.status === 'pending') {
        status = 'SKIP';
      }
      
      const error = testCaseResult.failureMessages?.join('\n\n');
      
      // Send the test case event with group hierarchy
      const payload = {
        testName: testName,
        parentNames: parentNames,
        status: status,
        duration: testCaseResult.duration
      };

      // Only include error if it exists
      if (error) {
        payload.error = {
          message: error
        };
      }

      sendEvent({
        eventType: 'testCase',
        payload: payload
      });
      
      // Track test in file group
      const fileGroup = fileGroups.get(test.path);
      if (fileGroup) {
        fileGroup.tests.push({
          name: testName,
          status: status,
          duration: testCaseResult.duration
        });
      }
    }
  }

  onTestResult(test, testResult, aggregatedResult) {
    this.stopCapture();
    
    // Send console output as group output
    if (testResult.console && testResult.console.length > 0) {
      for (const log of testResult.console) {
        const chunk = `${log.message}\n`;
        sendEvent({
          eventType: log.type === 'error' ? 'groupStderr' : 'groupStdout',
          payload: {
            groupName: test.path,
            parentNames: [],
            chunk: chunk
          }
        });
      }
    }
    
    // Calculate totals for the file group
    let totals = {
      total: 0,
      passed: 0,
      failed: 0,
      skipped: 0
    };
    
    if (testResult.testResults) {
      for (const testCase of testResult.testResults) {
        totals.total++;
        if (testCase.status === 'passed') {
          totals.passed++;
        } else if (testCase.status === 'failed') {
          totals.failed++;
        } else if (testCase.status === 'skipped' || testCase.status === 'pending') {
          totals.skipped++;
        }
      }
    }
    
    // Send GroupResult for all describe blocks (from deepest to shallowest)
    const processedGroups = new Set();
    
    if (testResult.testResults) {
      // Group tests by their ancestor titles to find describe blocks
      const describeGroups = new Map();
      
      for (const testCase of testResult.testResults) {
        const ancestorPath = testCase.ancestorTitles?.join(':') || '';
        if (!describeGroups.has(ancestorPath)) {
          describeGroups.set(ancestorPath, {
            ancestorTitles: testCase.ancestorTitles || [],
            tests: []
          });
        }
        describeGroups.get(ancestorPath).tests.push(testCase);
      }
      
      // Process describe groups from deepest to shallowest
      const sortedGroups = Array.from(describeGroups.entries())
        .sort((a, b) => b[1].ancestorTitles.length - a[1].ancestorTitles.length);
      
      for (const [ancestorPath, groupInfo] of sortedGroups) {
        if (groupInfo.ancestorTitles.length > 0) {
          const groupTotals = {
            total: groupInfo.tests.length,
            passed: groupInfo.tests.filter(t => t.status === 'passed').length,
            failed: groupInfo.tests.filter(t => t.status === 'failed').length,
            skipped: groupInfo.tests.filter(t => t.status === 'skipped' || t.status === 'pending').length
          };
          
          const groupName = groupInfo.ancestorTitles[groupInfo.ancestorTitles.length - 1];
          const parentNames = [test.path, ...groupInfo.ancestorTitles.slice(0, -1)];
          
          const groupId = getGroupId([...parentNames, groupName]);
          if (!processedGroups.has(groupId)) {
            processedGroups.add(groupId);
            
            let groupStatus = 'PASS';
            if (groupTotals.failed > 0) {
              groupStatus = 'FAIL';
            } else if (groupTotals.passed === 0 && groupTotals.skipped > 0) {
              groupStatus = 'SKIP';
            }
            
            // Calculate duration if we tracked start time
            const startTime = groupStarts.get(groupId);
            const duration = startTime ? Date.now() - startTime : undefined;
            
            sendEvent({
              eventType: 'testGroupResult',
              payload: {
                groupName: groupName,
                parentNames: parentNames,
                status: groupStatus,
                duration: duration,
                totals: groupTotals
              }
            });
          }
        }
      }
    }
    
    // Send GroupResult for the file itself
    const fileGroup = fileGroups.get(test.path);
    const fileStatus = totals.failed > 0 ? 'FAIL' : (totals.passed > 0 ? 'PASS' : 'SKIP');
    const fileDuration = fileGroup?.startTime ? Date.now() - fileGroup.startTime : undefined;
    
    sendEvent({
      eventType: 'testGroupResult',
      payload: {
        groupName: test.path,
        parentNames: [],
        status: fileStatus,
        duration: fileDuration,
        totals: totals
      }
    });
    
    this.currentTestFile = null;
  }

  onRunComplete(testContexts, results) {
    this.stopCapture();
    
    // Send run complete event
    sendEvent({
      eventType: 'runComplete',
      payload: {}
    });
  }

  startCapture() {
    if (this.captureEnabled) return;
    this.captureEnabled = true;
    
    process.stdout.write = (chunk, ...args) => {
      const chunkStr = chunk.toString();
      if (this.currentTestFile) {
        sendEvent({
          eventType: 'groupStdout',
          payload: {
            groupName: this.currentTestFile,
            parentNames: [],
            chunk: chunkStr
          }
        });
      }
      return true;
    };
    
    process.stderr.write = (chunk, ...args) => {
      const chunkStr = chunk.toString();
      if (this.currentTestFile) {
        sendEvent({
          eventType: 'groupStderr',
          payload: {
            groupName: this.currentTestFile,
            parentNames: [],
            chunk: chunkStr
          }
        });
      }
      return true;
    };
  }

  stopCapture() {
    if (!this.captureEnabled) return;
    this.captureEnabled = false;
    process.stdout.write = this.originalStdoutWrite;
    process.stderr.write = this.originalStderrWrite;
  }

  getLastError() {
    // Required by Jest reporter interface
  }
}

module.exports = ThreePioJestReporter;