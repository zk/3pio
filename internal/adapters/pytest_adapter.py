#!/usr/bin/env python
"""
3pio pytest adapter - Reports test results via IPC for AI-optimized reporting.

This plugin hooks into pytest execution and sends test events to a JSON Lines file
specified by the THREEPIO_IPC_PATH environment variable.
"""

import os
import sys
import json
import time
from pathlib import Path
from typing import Optional, Dict, Any
from io import StringIO, TextIOBase
from datetime import datetime

from _pytest.config import Config
from _pytest.reports import TestReport, CollectReport
from _pytest.nodes import Item

# Log level will be replaced at runtime
LOG_LEVEL = #__LOG_LEVEL__#"WARN"#__LOG_LEVEL__#
from _pytest.terminal import TerminalReporter


# Global reporter instance
_reporter: Optional['ThreepioReporter'] = None

# Worker detection flag
_is_worker = False


def is_xdist_worker() -> bool:
    """Detect if running in an xdist worker process.

    Returns True if this is an xdist worker process that should stay silent.
    Returns False if this is a standalone run or xdist controller that should report.
    """
    # Primary detection: check for PYTEST_XDIST_WORKER environment variable
    # This is present only in worker processes, not in the controller
    return os.environ.get('PYTEST_XDIST_WORKER') is not None


class ThreepioReporter:
    """pytest reporter that sends test events via IPC."""

    def __init__(self, ipc_path: str):
        # Store the absolute path to the IPC file to handle directory changes
        self.ipc_path = os.path.abspath(ipc_path)
        self.test_files = set()
        self.test_results = {}  # Track results per file
        self.current_test_file = None
        self.original_stdout = sys.stdout
        self.original_stderr = sys.stderr
        self.capture_enabled = False
        self.debug_log_path = Path.cwd() / ".3pio" / "debug.log"

        # Group tracking for universal abstractions
        self.discovered_groups = set()
        self.group_starts = set()
        self.file_groups = {}

        # Track processed skips to avoid duplicates
        self.processed_skips = set()  # Track (file_path, test_name) tuples

        self._ensure_debug_log_dir()
        self._log_startup()
        
    def send_event(self, event_type: str, payload: Dict[str, Any]) -> None:
        """Send an event to the IPC file."""
        event = {
            "eventType": event_type,
            "payload": payload,
            "timestamp": time.time()
        }

        try:
            # Write to IPC file (using absolute path stored in __init__)
            with open(self.ipc_path, 'a') as f:
                f.write(json.dumps(event) + '\n')
                f.flush()  # Ensure immediate write
        except Exception as e:
            # Log error to debug log but stay silent in console
            self._log_error(f"Failed to send IPC event to {self.ipc_path}: {e}")
    
    def _ensure_debug_log_dir(self) -> None:
        """Ensure the debug log directory exists."""
        try:
            self.debug_log_path.parent.mkdir(parents=True, exist_ok=True)
        except Exception:
            pass
    
    def _log(self, level: str, message: str) -> None:
        """Write a log message to the debug log file."""
        try:
            timestamp = datetime.now().isoformat()
            log_line = f"{timestamp} {level:5} | [pytest-adapter] {message}\n"
            with open(self.debug_log_path, 'a') as f:
                f.write(log_line)
                f.flush()
        except Exception:
            pass
    
    def _log_startup(self) -> None:
        """Log startup information."""
        self._log("INFO", "==================================")
        self._log("INFO", "3pio pytest Adapter v0.0.1")
        self._log("INFO", "Configuration:")
        self._log("INFO", f"  - IPC Path: {self.ipc_path}")
        self._log("INFO", f"  - Process ID: {os.getpid()}")

        # Add comprehensive startup logging
        self._log("INFO", "Path Analysis:")
        self._log("INFO", f"  - Current working directory: {os.getcwd()}")
        self._log("INFO", f"  - IPC path is absolute: {os.path.isabs(self.ipc_path)}")

        # Check if IPC file exists
        if os.path.exists(self.ipc_path):
            self._log("INFO", f"  - IPC file exists: YES")
            try:
                # Check if we can write to it
                with open(self.ipc_path, 'a') as f:
                    f.write("")  # Try empty write
                self._log("INFO", f"  - IPC file writable: YES")
            except Exception as e:
                self._log("ERROR", f"  - IPC file writable: NO - {e}")
        else:
            self._log("INFO", f"  - IPC file exists: NO")

            # Check if parent directory exists
            parent_dir = os.path.dirname(self.ipc_path)
            if os.path.exists(parent_dir):
                self._log("INFO", f"  - Parent directory exists: YES ({parent_dir})")
            else:
                self._log("INFO", f"  - Parent directory exists: NO ({parent_dir})")

        # Try to resolve the absolute path
        try:
            abs_path = os.path.abspath(self.ipc_path)
            self._log("INFO", f"  - Absolute IPC path: {abs_path}")
        except Exception as e:
            self._log("ERROR", f"  - Failed to resolve absolute path: {e}")

        self._log("INFO", "==================================")
    
    def _log_info(self, message: str) -> None:
        """Log an info message."""
        self._log("INFO", message)
    
    def _log_error(self, message: str) -> None:
        """Log an error message."""
        self._log("ERROR", message)
    
    def _log_debug(self, message: str) -> None:
        """Log a debug message."""
        # Skip debug logging for production performance
        pass

    def get_group_id(self, hierarchy):
        """Generate a unique ID for a group path."""
        return ':'.join(hierarchy)

    def parse_test_hierarchy(self, nodeid: str):
        """Parse pytest nodeid into hierarchy components."""
        parts = nodeid.split('::')
        file_path = parts[0]

        if len(parts) == 2:
            # Simple function test: test_file.py::test_function
            return file_path, [], parts[1]
        elif len(parts) == 3:
            # Class-based test: test_file.py::TestClass::test_method
            class_name = parts[1]
            test_name = parts[2]
            return file_path, [class_name], test_name
        else:
            # Fallback for other formats
            test_name = parts[-1]
            suite_chain = parts[1:-1] if len(parts) > 2 else []
            return file_path, suite_chain, test_name

    def build_hierarchy_from_file(self, file_path: str, suite_chain=None):
        """Build complete hierarchy from file and suite structure."""
        if suite_chain is None:
            suite_chain = []
        hierarchy = [file_path]
        if suite_chain:
            hierarchy.extend(suite_chain)
        return hierarchy

    def discover_groups(self, file_path: str, suite_chain=None):
        """Discover all groups in a hierarchy."""
        if suite_chain is None:
            suite_chain = []

        groups = []

        # First, the file itself is a group
        groups.append({
            'hierarchy': [file_path],
            'name': file_path,
            'parent_names': []
        })

        # Then each level of suites creates a nested group
        if suite_chain:
            for i in range(len(suite_chain)):
                parent_names = [file_path] + suite_chain[:i]
                group_name = suite_chain[i]
                groups.append({
                    'hierarchy': parent_names + [group_name],
                    'name': group_name,
                    'parent_names': parent_names
                })

        return groups

    def ensure_groups_discovered(self, file_path: str, suite_chain=None):
        """Send GroupDiscovered events for new groups."""
        if suite_chain is None:
            suite_chain = []

        groups = self.discover_groups(file_path, suite_chain)

        for group in groups:
            group_id = self.get_group_id(group['hierarchy'])
            if group_id not in self.discovered_groups:
                self.discovered_groups.add(group_id)
                self._log_debug(f"Discovering group: {group['name']} (parents: {group['parent_names']})")
                self.send_event('testGroupDiscovered', {
                    'groupName': group['name'],
                    'parentNames': group['parent_names']
                })

    def ensure_group_started(self, hierarchy):
        """Send GroupStart event if not already started."""
        group_id = self.get_group_id(hierarchy)
        if group_id not in self.group_starts:
            self.group_starts.add(group_id)

            # Find the group info from discovered groups
            groups = self.discover_groups(hierarchy[0], hierarchy[1:])
            for group in groups:
                if self.get_group_id(group['hierarchy']) == group_id:
                    self._log_debug(f"Starting group: {group['name']}")
                    self.send_event('testGroupStart', {
                        'groupName': group['name'],
                        'parentNames': group['parent_names']
                    })
                    break
    
    def get_file_path(self, item: Item) -> str:
        """Extract the test file path from a test item."""
        # Get the file path relative to the current directory
        file_path = str(item.fspath)
        
        # Try to make it relative to current working directory
        try:
            cwd = Path.cwd()
            file_path_obj = Path(file_path).resolve()
            file_path = str(file_path_obj.relative_to(cwd))
        except (ValueError, OSError):
            # If can't make relative, use absolute path
            pass
        
        return file_path
    
    def start_capture(self, file_path: str) -> None:
        """Start capturing stdout/stderr for a test file."""
        if self.capture_enabled:
            return
            
        self.current_test_file = file_path
        self.capture_enabled = True
        
        # Create a silent stream that only sends to IPC, doesn't write to terminal
        class SilentIPCStream:
            def __init__(self, reporter, original_stream, stream_type):
                self.reporter = reporter
                self.original = original_stream
                self.stream_type = stream_type
                # Copy essential attributes from original stream
                self.encoding = getattr(original_stream, 'encoding', 'utf-8')
                self.errors = getattr(original_stream, 'errors', 'strict')
                self.newlines = getattr(original_stream, 'newlines', None)
            
            def write(self, text):
                # Send to IPC if we have a current test file
                if self.reporter.current_test_file and text:
                    # Old stdout/stderr chunk events removed - using group events instead
                    # Output is now captured by group events (groupStdout/groupStderr)
                    pass
                
                # Return length of text to indicate success, but DON'T write to terminal
                # This makes the adapter completely silent like Jest/Vitest
                return len(text) if text else 0
            
            def flush(self):
                # No-op since we're not writing to a real stream
                pass
            
            def isatty(self):
                return False
            
            def readable(self):
                return False
            
            def writable(self):
                return True
            
            def seekable(self):
                return False
            
            def __getattr__(self, name):
                # For any other attributes, try to get from original
                return getattr(self.original, name)
        
        # Replace stdout and stderr with our silent streams
        sys.stdout = SilentIPCStream(self, self.original_stdout, "stdout")
        sys.stderr = SilentIPCStream(self, self.original_stderr, "stderr")
    
    def stop_capture(self) -> None:
        """Stop capturing stdout/stderr."""
        if not self.capture_enabled:
            return
            
        self.capture_enabled = False
        self.current_test_file = None
        
        # Don't restore original streams - we manage the entire test run
        # This prevents any buffered output from appearing after capture stops


def pytest_configure(config: Config) -> None:
    """Register the 3pio reporter if IPC path is set."""
    global _reporter, _is_worker

    # Detect if we're running in a worker process
    _is_worker = is_xdist_worker()

    ipc_path = #__IPC_PATH__#"WILL_BE_REPLACED"#__IPC_PATH__#

    # Only initialize reporter in non-worker processes
    if _is_worker:
        # Log detection for debugging (to stderr since we're not initializing reporter)
        try:
            debug_log_path = Path.cwd() / ".3pio" / "debug.log"
            debug_log_path.parent.mkdir(parents=True, exist_ok=True)
            with open(debug_log_path, 'a') as f:
                timestamp = datetime.now().isoformat()
                worker_id = os.environ.get('PYTEST_XDIST_WORKER', 'unknown')
                f.write(f"{timestamp} INFO  | [pytest-adapter] Worker {worker_id} detected - staying silent\n")
                f.flush()
        except:
            pass
        return  # Don't initialize reporter in worker mode

    if True:  # IPC path will always be present after injection
        # Create the reporter instance
        _reporter = ThreepioReporter(ipc_path)
        _reporter._log_info("Plugin initialized in pytest_configure (non-worker mode)")
        _reporter._log_info(f"PYTEST_XDIST_WORKER env var: {os.environ.get('PYTEST_XDIST_WORKER', 'not set')}")

        # Store it in config for access in other hooks
        config._threepio_reporter = _reporter
        
        # Start capturing immediately to catch collection errors
        # Use a special file path for collection phase
        _reporter.start_capture("__collection__")
        
        # Send an event to indicate collection is starting
        _reporter.send_event("collectionStart", {"phase": "collection"})
        
        # Note: Output capture is disabled via -s flag added by 3pio CLI
        # This ensures we can capture all print statements from tests
        
        # Reduce verbosity for cleaner output
        if not hasattr(config.option, 'verbose') or config.option.verbose is None:
            config.option.verbose = -1
        if not hasattr(config.option, 'quiet') or not config.option.quiet:
            config.option.quiet = True


def pytest_collectreport(report: CollectReport) -> None:
    """Handle collection errors."""
    global _reporter, _is_worker

    # Skip in worker mode
    if _is_worker:
        return

    if not _reporter:
        return
    
    # Check if there was a collection error
    if report.failed:
        # Extract the file path if available
        file_path = str(report.nodeid) if report.nodeid else "__collection__"
        
        # Send collection error event
        payload = {
            "filePath": file_path,
            "error": str(report.longrepr) if hasattr(report, 'longrepr') else "Collection failed",
            "phase": "collection"
        }
        
        _reporter.send_event("collectionError", payload)
        
        # Collection error details are captured in the collectionError event


def pytest_collection_finish(session) -> None:
    """Called after collection is completed."""
    global _reporter, _is_worker

    # Skip in worker mode
    if _is_worker:
        return

    if _reporter:
        # Send event to indicate collection finished
        # Get count of collected items
        collected_count = len(session.items) if hasattr(session, 'items') else 0
        _reporter.send_event("collectionFinish", {
            "collected": collected_count
        })
        
        # If no tests were collected, we might still be in collection phase capture
        # Keep capturing for any subsequent errors


def pytest_runtest_protocol(item: Item, nextitem: Optional[Item]) -> None:
    """Called when running a single test."""
    global _reporter, _is_worker

    # Skip in worker mode
    if _is_worker:
        return

    if _reporter:
        file_path = _reporter.get_file_path(item)

        # testFileStart event removed - using group events instead
        if file_path not in _reporter.test_files:
            _reporter.test_files.add(file_path)
            # File discovery and group management handled by group events
            _reporter.test_results[file_path] = {"passed": 0, "failed": 0, "skipped": 0, "failed_tests": []}

            # Discover the file as a root group and start it
            _reporter.ensure_groups_discovered(file_path, [])
            _reporter.ensure_group_started([file_path])

            # Store file group info
            _reporter.file_groups[file_path] = {
                'start_time': time.time(),
                'tests': []
            }

            # Update the file path for capturing (capture already started in pytest_configure)
            _reporter.current_test_file = file_path
        elif _reporter.current_test_file != file_path:
            # Switch to capturing for a different test file
            _reporter.current_test_file = file_path


def _extract_skip_reason(report: TestReport) -> str:
    """Extract skip reason from pytest report object."""
    if hasattr(report, 'longrepr'):
        if isinstance(report.longrepr, tuple) and len(report.longrepr) >= 3:
            # Format: (category, condition, reason)
            reason = str(report.longrepr[2])
            # Remove 'Skipped: ' prefix if present
            if reason.startswith('Skipped: '):
                return reason[9:]
            return reason
        elif isinstance(report.longrepr, str):
            return report.longrepr

    return "Test skipped"


def pytest_runtest_logreport(report: TestReport) -> None:
    """Process test reports."""
    global _reporter, _is_worker

    # Skip in worker mode
    if _is_worker:
        return

    if not _reporter:
        return
    
    # Parse test hierarchy from nodeid (needed for all phases)
    file_path, suite_chain, test_name = _reporter.parse_test_hierarchy(report.nodeid)

    # Initialize results for file if needed
    if file_path not in _reporter.test_results:
        _reporter.test_results[file_path] = {"passed": 0, "failed": 0, "skipped": 0, "xfailed": 0, "xpassed": 0, "failed_tests": []}

    # Check if this is an xfail case (must check before skip handling)
    has_xfail = hasattr(report, 'wasxfail')

    # HANDLE SKIPPED TESTS IN ANY PHASE (but not xfail)
    if report.skipped and report.when in ('setup', 'call') and not has_xfail:
        # Check for duplicate processing
        skip_key = (file_path, test_name)
        if skip_key in _reporter.processed_skips:
            return
        _reporter.processed_skips.add(skip_key)

        # Extract skip reason
        skip_reason = _extract_skip_reason(report)

        # Update skip count
        _reporter.test_results[file_path]["skipped"] += 1

        # Ensure all parent groups are discovered and started
        _reporter.ensure_groups_discovered(file_path, suite_chain)

        # Start all parent groups in the hierarchy
        for i in range(len(suite_chain) + 1):
            hierarchy = [file_path] + suite_chain[:i]
            _reporter.ensure_group_started(hierarchy)

        # Build complete hierarchy for this test case
        parent_names = _reporter.build_hierarchy_from_file(file_path, suite_chain)

        # Send IPC event for skipped test
        _reporter.send_event("testCase", {
            "testName": test_name,
            "parentNames": parent_names,
            "status": "SKIP",
            "skipReason": skip_reason,
            "skipPhase": report.when  # 'setup' or 'call'
        })

        # Track test in file group
        if file_path in _reporter.file_groups:
            _reporter.file_groups[file_path]['tests'].append({
                'name': test_name,
                'status': 'SKIP',
                'duration': 0
            })

        return

    # Only process non-skip events from call phase
    if report.when != 'call':
        return

    # Determine test status

    if has_xfail:
        # Handle xfail/xpass cases
        if report.passed:
            status = "XPASS"  # Test passed unexpectedly
            _reporter.test_results[file_path]["xpassed"] = _reporter.test_results[file_path].get("xpassed", 0) + 1
        else:  # report.skipped is True for xfail
            status = "XFAIL"  # Test failed as expected
            _reporter.test_results[file_path]["xfailed"] = _reporter.test_results[file_path].get("xfailed", 0) + 1
    elif report.passed:
        status = "PASS"
        _reporter.test_results[file_path]["passed"] += 1
    elif report.failed:
        status = "FAIL"
        _reporter.test_results[file_path]["failed"] += 1
    elif report.skipped:
        status = "SKIP"
        _reporter.test_results[file_path]["skipped"] += 1
    else:
        status = "UNKNOWN"

    # Ensure all parent groups are discovered and started
    _reporter.ensure_groups_discovered(file_path, suite_chain)

    # Start all parent groups in the hierarchy
    for i in range(len(suite_chain) + 1):
        hierarchy = [file_path] + suite_chain[:i]
        _reporter.ensure_group_started(hierarchy)

    # Build complete hierarchy for this test case
    parent_names = _reporter.build_hierarchy_from_file(file_path, suite_chain)

    # Build test case payload with group hierarchy
    payload = {
        "testName": test_name,
        "parentNames": parent_names,
        "status": status,
        "duration": report.duration * 1000 if hasattr(report, 'duration') else 0  # Convert to milliseconds
    }

    # Add xfail reason if available
    if has_xfail:
        payload["xfailReason"] = str(report.wasxfail)

    # Add error information for failures
    if report.failed:
        if hasattr(report, 'longrepr') and report.longrepr:
            payload["error"] = str(report.longrepr)
        # Track failed test for file result
        _reporter.test_results[file_path]["failed_tests"].append({
            "name": test_name,
            "duration": report.duration * 1000 if hasattr(report, 'duration') else 0
        })

    # Track test in file group
    if file_path in _reporter.file_groups:
        _reporter.file_groups[file_path]['tests'].append({
            'name': test_name,
            'status': status,
            'duration': report.duration * 1000 if hasattr(report, 'duration') else 0
        })

    _reporter._log_debug(f"Sending test case: {test_name} with parents: {parent_names}")

    # Send test case event with group hierarchy
    _reporter.send_event("testCase", payload)


def pytest_sessionfinish(session, exitstatus: int) -> None:
    """Called after all tests have run."""
    global _reporter, _is_worker

    # Skip in worker mode
    if _is_worker:
        return

    if not _reporter:
        return
    
    _reporter._log_info(f"Session finished with exit status: {exitstatus}")
    
    # Send group result events for all test files
    for file_path in _reporter.test_files:
        results = _reporter.test_results.get(file_path, {})
        file_group = _reporter.file_groups.get(file_path, {})

        # Determine overall file status
        if results.get("failed", 0) > 0:
            status = "FAIL"
        elif results.get("passed", 0) > 0:
            status = "PASS"
        elif results.get("skipped", 0) > 0:
            status = "SKIP"
        else:
            status = "UNKNOWN"

        # Calculate duration
        duration = None
        if 'start_time' in file_group:
            duration = (time.time() - file_group['start_time']) * 1000  # Convert to milliseconds

        # Calculate totals for the file group
        totals = {
            'total': (results.get("passed", 0) + results.get("failed", 0) +
                     results.get("skipped", 0) + results.get("xfailed", 0) +
                     results.get("xpassed", 0)),
            'passed': results.get("passed", 0),
            'failed': results.get("failed", 0),
            'skipped': results.get("skipped", 0),
            'xfailed': results.get("xfailed", 0),
            'xpassed': results.get("xpassed", 0)
        }

        _reporter._log_debug(f"Sending group result for file: {file_path} (status: {status})")

        # Send GroupResult for the file
        _reporter.send_event("testGroupResult", {
            "groupName": file_path,
            "parentNames": [],
            "status": status,
            "duration": duration,
            "totals": totals
        })

        # testFileResult event removed - using group events instead


def pytest_unconfigure(config: Config) -> None:
    """Clean up when pytest is done."""
    global _reporter, _is_worker

    # Skip in worker mode
    if _is_worker:
        return

    # Note: We don't restore stdout/stderr since we manage the entire test run
    # This prevents any buffered output from appearing after the tests complete

    # Clear the global reporter
    _reporter = None