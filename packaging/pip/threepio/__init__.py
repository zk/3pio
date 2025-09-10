#!/usr/bin/env python3
"""
3pio - A context-competent test runner for coding agents.

This package provides a native Go binary with zero runtime dependencies
that acts as a protocol droid for test frameworks like Jest, Vitest, and pytest.
"""

import os
import subprocess
import sys
from pathlib import Path

__version__ = "0.0.1"

def get_binary_path():
    """Get the path to the 3pio binary."""
    package_dir = Path(__file__).parent
    
    # Check for binary in the package bin directory
    if sys.platform == "win32":
        binary_name = "3pio.exe"
    else:
        binary_name = "3pio"
        
    binary_path = package_dir / "bin" / binary_name
    
    if binary_path.exists():
        return str(binary_path)
        
    # Fallback: check PATH
    import shutil
    binary_in_path = shutil.which("3pio")
    if binary_in_path:
        return binary_in_path
        
    raise RuntimeError(
        "3pio binary not found. Please ensure the package was installed correctly."
    )

def main():
    """Main entry point for the 3pio command."""
    try:
        binary_path = get_binary_path()
        
        # Pass all arguments to the binary
        args = [binary_path] + sys.argv[1:]
        
        # Execute the binary with the same environment
        result = subprocess.run(args, env=os.environ.copy())
        
        # Exit with the same code as the binary
        sys.exit(result.returncode)
        
    except FileNotFoundError:
        print("❌ 3pio binary not found", file=sys.stderr)
        print("Please ensure 3pio is installed correctly with: pip install 3pio", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)  # Standard exit code for Ctrl+C
    except Exception as e:
        print(f"❌ Error running 3pio: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()