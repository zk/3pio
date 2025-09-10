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
from io import StringIO

from _pytest.config import Config
from _pytest.reports import TestReport
from _pytest.nodes import Item


# Global reporter instance
_reporter: Optional['ThreepioReporter'] = None


class ThreepioReporter:
    """pytest reporter that sends test events via IPC."""
    
    def __init__(self, ipc_path: str):
        self.ipc_path = ipc_path
        self.test_files = set()
        self.test_results = {}  # Track results per file
        
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


def pytest_configure(config: Config) -> None:
    """Register the 3pio reporter if IPC path is set."""
    global _reporter
    
    ipc_path = os.environ.get('THREEPIO_IPC_PATH')
    
    if ipc_path:
        # Create the reporter instance
        _reporter = ThreepioReporter(ipc_path)
        
        # Store it in config for access in other hooks
        config._threepio_reporter = _reporter
        
        # Note: Output capture is disabled via -s flag added by 3pio CLI
        # This ensures we can capture all print statements from tests
        
        # Reduce verbosity for cleaner output
        if not hasattr(config.option, 'verbose') or config.option.verbose is None:
            config.option.verbose = -1
        if not hasattr(config.option, 'quiet') or not config.option.quiet:
            config.option.quiet = True


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
    
    # Clear the global reporter
    _reporter = None