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
from _pytest.terminal import TerminalReporter


# Global reporter instance
_reporter: Optional['ThreepioReporter'] = None


class ThreepioReporter:
    """pytest reporter that sends test events via IPC."""

    def __init__(self, ipc_path: str):
        self.ipc_path = ipc_path
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
            # Write to IPC file (append mode, create if doesn't exist)
            with open(self.ipc_path, 'a') as f:
                f.write(json.dumps(event) + '\n')
                f.flush()  # Ensure immediate write
        except Exception as e:
            # Log error to debug log but stay silent in console
            self._log_error(f"Failed to send IPC event: {e}")
    
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
        self._log("INFO", "==================================")
    
    def _log_info(self, message: str) -> None:
        """Log an info message."""
        self._log("INFO", message)
    
    def _log_error(self, message: str) -> None:
        """Log an error message."""
        self._log("ERROR", message)
    
    def _log_debug(self, message: str) -> None:
        """Log a debug message."""
        if os.environ.get("THREEPIO_DEBUG") == "1":
            self._log("DEBUG", message)

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
                    event_type = "stdoutChunk" if self.stream_type == "stdout" else "stderrChunk"
                    try:
                        self.reporter.send_event(event_type, {
                            "filePath": self.reporter.current_test_file,
                            "chunk": text
                        })
                    except:
                        pass  # Silently ignore IPC errors
                
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
    global _reporter
    
    ipc_path = #__IPC_PATH__#"WILL_BE_REPLACED"#__IPC_PATH__#
    
    if True:  # IPC path will always be present after injection
        # Create the reporter instance
        _reporter = ThreepioReporter(ipc_path)
        _reporter._log_info("Plugin initialized in pytest_configure")
        
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
    global _reporter
    
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
        
        # Also send as stderr to capture the error
        if hasattr(report, 'longrepr') and report.longrepr:
            _reporter.send_event("stderrChunk", {
                "filePath": file_path,
                "chunk": f"Collection Error:\n{str(report.longrepr)}\n"
            })


def pytest_collection_finish(session) -> None:
    """Called after collection is completed."""
    global _reporter
    
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
    global _reporter

    if _reporter:
        file_path = _reporter.get_file_path(item)

        # Send testFileStart event if this is a new file
        if file_path not in _reporter.test_files:
            _reporter.test_files.add(file_path)
            _reporter.send_event("testFileStart", {"filePath": file_path})
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


def pytest_runtest_logreport(report: TestReport) -> None:
    """Process test reports."""
    global _reporter
    
    if not _reporter:
        return
    
    # Only process the 'call' phase (actual test execution)
    if report.when != 'call':
        return
    
    # Parse the test hierarchy from nodeid
    file_path, suite_chain, test_name = _reporter.parse_test_hierarchy(report.nodeid)

    # Initialize results for file if needed
    if file_path not in _reporter.test_results:
        _reporter.test_results[file_path] = {"passed": 0, "failed": 0, "skipped": 0, "failed_tests": []}

    # Determine test status
    if report.passed:
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
    global _reporter
    
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
            'total': results.get("passed", 0) + results.get("failed", 0) + results.get("skipped", 0),
            'passed': results.get("passed", 0),
            'failed': results.get("failed", 0),
            'skipped': results.get("skipped", 0)
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

        # Keep legacy testFileResult for compatibility
        payload = {
            "filePath": file_path,
            "status": status
        }

        # Add failed test details for FAIL status
        if status == "FAIL" and results.get("failed_tests"):
            payload["failedTests"] = results["failed_tests"]

        _reporter.send_event("testFileResult", payload)


def pytest_unconfigure(config: Config) -> None:
    """Clean up when pytest is done."""
    global _reporter
    
    # Note: We don't restore stdout/stderr since we manage the entire test run
    # This prevents any buffered output from appearing after the tests complete
    
    # Clear the global reporter
    _reporter = None