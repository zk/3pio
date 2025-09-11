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
            # For debugging - normally we'd be silent
            pass
    
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
    
    ipc_path = os.environ.get('THREEPIO_IPC_PATH')
    
    if ipc_path:
        # Create the reporter instance
        _reporter = ThreepioReporter(ipc_path)
        
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
        _reporter.send_event("collectionFinish", {
            "collected": session.testscollected if hasattr(session, 'testscollected') else 0
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
    
    # Extract file path from nodeid (format: path/to/test.py::TestClass::test_method)
    file_path = report.nodeid.split('::')[0]
    
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
    
    # Extract test name
    test_parts = report.nodeid.split('::')
    if len(test_parts) > 1:
        test_name = '::'.join(test_parts[1:])
    else:
        test_name = report.nodeid
    
    # Build test case payload
    payload = {
        "filePath": file_path,
        "testName": test_name,
        "status": status,
        "duration": report.duration * 1000 if hasattr(report, 'duration') else 0  # Convert to milliseconds
    }
    
    # Add suite name if test is in a class
    if len(test_parts) > 2:
        payload["suiteName"] = test_parts[1]
    
    # Add error information for failures
    if report.failed:
        if hasattr(report, 'longrepr') and report.longrepr:
            payload["error"] = str(report.longrepr)
        # Track failed test for file result
        _reporter.test_results[file_path]["failed_tests"].append({
            "name": test_name,
            "duration": report.duration * 1000 if hasattr(report, 'duration') else 0
        })
    
    # Send test case event
    _reporter.send_event("testCase", payload)


def pytest_sessionfinish(session, exitstatus: int) -> None:
    """Called after all tests have run."""
    global _reporter
    
    if not _reporter:
        return
    
    # Send testFileResult events for all test files
    for file_path in _reporter.test_files:
        results = _reporter.test_results.get(file_path, {})
        
        # Determine overall file status
        if results.get("failed", 0) > 0:
            status = "FAIL"
        elif results.get("passed", 0) > 0:
            status = "PASS"
        elif results.get("skipped", 0) > 0:
            status = "SKIP"
        else:
            status = "UNKNOWN"
        
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