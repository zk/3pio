// Test adapter to debug Jest reporter lifecycle
class TestReporter {
  constructor() {
    console.error('[TestReporter] Constructor called');
  }

  onRunStart() {
    console.error('[TestReporter] onRunStart called');
  }

  onTestStart(test) {
    console.error(`[TestReporter] onTestStart called for ${test.path}`);
  }

  onTestResult(test, testResult) {
    console.error(`[TestReporter] onTestResult called for ${test.path}, status: ${testResult.numFailingTests > 0 ? 'FAIL' : 'PASS'}`);
  }

  onRunComplete() {
    console.error('[TestReporter] onRunComplete called');
  }

  getLastError() {
    // Required by interface
  }
}

module.exports = TestReporter;