#!/usr/bin/env python3
import os
import sys

# Add dist to path
sys.path.insert(0, '/Users/zk/code/3pio/dist')

# Set IPC path
os.environ['THREEPIO_IPC_PATH'] = '/tmp/test_debug.jsonl'

# Import pytest
import pytest

# Run pytest with our adapter
result = pytest.main([
    '-p', 'pytest_adapter',  # Load our adapter
    '-v',  # Verbose
    '--tb=short',  # Short traceback
    '/Users/zk/code/3pio/tests/fixtures/basic-pytest/test_math.py::TestMathOperations::test_add_numbers_correctly'
])

print(f"Exit code: {result}")

# Check if IPC file was created
if os.path.exists('/tmp/test_debug.jsonl'):
    print("IPC file created!")
    with open('/tmp/test_debug.jsonl', 'r') as f:
        print("Contents:")
        print(f.read())
else:
    print("IPC file NOT created")