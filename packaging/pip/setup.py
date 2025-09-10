#!/usr/bin/env python3

import os
import platform
import subprocess
import sys
from pathlib import Path
from setuptools import setup
from setuptools.command.install import install
import urllib.request
import tarfile
import zipfile
import stat

PACKAGE_VERSION = "0.0.1"
GITHUB_OWNER = "zk"  
GITHUB_REPO = "3pio"

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

def get_binary_url(version, platform_name, arch_name):
    base_url = f"https://github.com/{GITHUB_OWNER}/{GITHUB_REPO}/releases/download"
    extension = "zip" if platform_name == "windows" else "tar.gz"
    filename = f"3pio-{platform_name}-{arch_name}.{extension}"
    return f"{base_url}/v{version}/{filename}"

def download_file(url, destination):
    """Download file with progress indication."""
    print(f"Downloading from: {url}")
    urllib.request.urlretrieve(url, destination)
    print("Download completed")

def extract_archive(source_path, dest_dir):
    """Extract tar.gz or zip archive."""
    if source_path.suffix == '.gz':
        with tarfile.open(source_path, 'r:gz') as tar:
            tar.extractall(dest_dir)
    elif source_path.suffix == '.zip':
        with zipfile.ZipFile(source_path, 'r') as zip_file:
            zip_file.extractall(dest_dir)
    else:
        raise ValueError(f"Unsupported archive format: {source_path.suffix}")

def make_executable(file_path):
    """Make file executable on Unix systems."""
    if platform.system() != "Windows":
        current_permissions = os.stat(file_path).st_mode
        os.chmod(file_path, current_permissions | stat.S_IEXEC)

class PostInstallCommand(install):
    """Custom install command to download binary after package installation."""
    
    def run(self):
        install.run(self)
        self.download_binary()
        
    def download_binary(self):
        try:
            print("Installing 3pio binary...")
            
            platform_name, arch_name = get_platform_info()
            print(f"Platform: {platform_name}, Architecture: {arch_name}")
            
            # Create bin directory in package location
            package_dir = Path(self.install_lib) / "threepio"
            bin_dir = package_dir / "bin"
            bin_dir.mkdir(parents=True, exist_ok=True)
            
            url = get_binary_url(PACKAGE_VERSION, platform_name, arch_name)
            
            # Download archive
            extension = "zip" if platform_name == "windows" else "tar.gz"
            archive_name = f"3pio-{platform_name}-{arch_name}.{extension}"
            archive_path = bin_dir / archive_name
            
            download_file(url, archive_path)
            
            # Extract binary
            print("Extracting binary...")
            extract_archive(archive_path, bin_dir)
            
            # Find and setup the binary
            binary_name = "3pio.exe" if platform_name == "windows" else "3pio"
            binary_path = bin_dir / binary_name
            
            if not binary_path.exists():
                raise FileNotFoundError(f"Binary not found after extraction: {binary_path}")
                
            make_executable(binary_path)
            
            # Clean up archive
            archive_path.unlink()
            
            print("✅ 3pio binary installed successfully")
            
        except Exception as e:
            print(f"❌ Failed to install 3pio binary: {e}")
            sys.exit(1)

# Read README file
def read_readme():
    readme_path = Path(__file__).parent / "README.md"
    if readme_path.exists():
        with open(readme_path, "r", encoding="utf-8") as f:
            return f.read()
    return ""

setup(
    name="3pio",
    version=PACKAGE_VERSION,
    description="A context-competent test runner for coding agents",
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
    keywords="test testing jest vitest pytest ai adapter reporter go binary",
    license="MIT",
)