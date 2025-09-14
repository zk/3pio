#!/usr/bin/env python3

import os
import platform
import sys
from pathlib import Path
from setuptools import setup
from setuptools.command.install import install
import shutil
import stat

PACKAGE_VERSION = "0.2.0"

# Platform and architecture mapping
PLATFORM_MAPPING = {
    "Darwin": "darwin",
    "Linux": "linux", 
    "Windows": "windows"
}

ARCH_MAPPING = {
    "x86_64": "amd64",
    "aarch64": "arm64",
    "arm64": "arm64",
    "AMD64": "amd64"  # Windows reports this
}

def get_platform_info():
    system = platform.system()
    machine = platform.machine()
    
    platform_name = PLATFORM_MAPPING.get(system)
    if not platform_name:
        raise ValueError(f"Unsupported platform: {system}")
        
    arch_name = ARCH_MAPPING.get(machine)
    if not arch_name:
        raise ValueError(f"Unsupported architecture: {machine}")
        
    return platform_name, arch_name

def make_executable(file_path):
    """Make file executable on Unix systems."""
    if platform.system() != "Windows":
        current_permissions = os.stat(file_path).st_mode
        os.chmod(file_path, current_permissions | stat.S_IEXEC)

class PostInstallCommand(install):
    """Custom install command to set up the correct binary after package installation."""
    
    def run(self):
        install.run(self)
        self.setup_binary()
        
    def setup_binary(self):
        try:
            print("Setting up 3pio binary...")
            
            platform_name, arch_name = get_platform_info()
            print(f"Platform: {platform_name}, Architecture: {arch_name}")
            
            # Get package directory where binaries are installed
            package_dir = Path(self.install_lib) / "threepio"
            bin_dir = package_dir / "bin"
            binaries_dir = package_dir / "binaries"
            
            # Create bin directory if it doesn't exist
            bin_dir.mkdir(parents=True, exist_ok=True)
            
            # Source binary name
            source_binary_name = f"3pio-{platform_name}-{arch_name}"
            if platform_name == "windows":
                source_binary_name += ".exe"
            
            source_path = binaries_dir / source_binary_name
            
            if not source_path.exists():
                raise FileNotFoundError(f"Binary not found for your platform: {source_binary_name}")
            
            # Destination binary name (always "3pio" or "3pio.exe")
            dest_binary_name = "3pio.exe" if platform_name == "windows" else "3pio"
            dest_path = bin_dir / dest_binary_name
            
            # Copy the appropriate binary to bin/3pio
            print(f"Copying {source_binary_name} to bin/{dest_binary_name}")
            shutil.copy2(source_path, dest_path)
            
            # Make it executable
            make_executable(dest_path)
            
            print("3pio binary set up successfully")
            print("")
            print("Installation complete! You can now use the '3pio' command:")
            print("  3pio pytest           # Run pytest tests")
            print("  3pio pytest -v        # Run with verbose output")
            print("  3pio --help           # Show help")
            print("")
            print("Note: Package installed as 'threepio-test-runner', command is '3pio'")
            
        except Exception as e:
            print(f"Failed to set up 3pio binary: {e}")
            print("Please report this issue at https://github.com/zk/3pio/issues")
            sys.exit(1)

# Read README file
def read_readme():
    readme_path = Path(__file__).parent / "README.md"
    if readme_path.exists():
        with open(readme_path, "r", encoding="utf-8") as f:
            return f.read()
    return ""

setup(
    name="threepio-test-runner",
    version=PACKAGE_VERSION,
    description="Context-optimized test runner for coding agents",
    long_description=read_readme(),
    long_description_content_type="text/markdown",
    author="Zachary Kim",
    author_email="zachary@example.com",
    url="https://github.com/zk/3pio",
    project_urls={
        "Bug Reports": "https://github.com/zk/3pio/issues",
        "Source": "https://github.com/zk/3pio",
    },
    packages=["threepio"],
    package_dir={"threepio": "threepio"},
    package_data={
        "threepio": [
            "binaries/3pio-darwin-amd64",
            "binaries/3pio-darwin-arm64",
            "binaries/3pio-linux-amd64",
            "binaries/3pio-linux-arm64",
            "binaries/3pio-windows-amd64.exe",
        ]
    },
    entry_points={
        "console_scripts": [
            "3pio=threepio:main",
        ],
    },
    cmdclass={
        "install": PostInstallCommand,
    },
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9", 
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Software Development :: Testing",
        "Topic :: Software Development :: Libraries",
    ],
    python_requires=">=3.8",
    install_requires=[],
    keywords="test testing pytest ai adapter reporter",
    license="MIT",
)