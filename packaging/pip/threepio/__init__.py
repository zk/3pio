#!/usr/bin/env python3
"""
3pio - A context-friendly test runner for coding agents.

See https://github.com/zk/3pio

"""

import os
import subprocess
import sys
from pathlib import Path

__version__ = "0.0.2"

def setup_binary():
    """Set up the platform-specific binary on first run."""
    import platform
    import shutil
    
    package_dir = Path(__file__).parent
    bin_dir = package_dir / "bin"
    binaries_dir = package_dir / "binaries"
    
    # Determine platform
    system = platform.system()
    machine = platform.machine()
    
    platform_map = {"Darwin": "darwin", "Linux": "linux", "Windows": "windows"}
    arch_map = {"x86_64": "amd64", "aarch64": "arm64", "arm64": "arm64", "AMD64": "amd64"}
    
    platform_name = platform_map.get(system)
    arch_name = arch_map.get(machine)
    
    if not platform_name or not arch_name:
        raise RuntimeError(f"Unsupported platform: {system} {machine}")
    
    # Source and destination paths
    source_name = f"3pio-{platform_name}-{arch_name}"
    if platform_name == "windows":
        source_name += ".exe"
    
    source_path = binaries_dir / source_name
    dest_name = "3pio.exe" if platform_name == "windows" else "3pio"
    dest_path = bin_dir / dest_name
    
    # Create bin directory and copy binary if not exists
    if not dest_path.exists():
        bin_dir.mkdir(parents=True, exist_ok=True)
        if source_path.exists():
            shutil.copy2(source_path, dest_path)
            # Make executable on Unix
            if platform_name != "windows":
                import stat
                current = dest_path.stat().st_mode
                dest_path.chmod(current | stat.S_IEXEC)
            
            # Print setup message
            print("3pio binary set up successfully")
            print("You can now use the '3pio' command")
    
    return str(dest_path)

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
    
    # Try to set up the binary
    try:
        return setup_binary()
    except Exception:
        pass
        
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
        print("3pio binary not found", file=sys.stderr)
        print("Please ensure 3pio is installed correctly with: pip install threepio-test-runner", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)  # Standard exit code for Ctrl+C
    except Exception as e:
        print(f"Error running 3pio: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
