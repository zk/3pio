#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Platform and architecture mapping
const PLATFORM_MAPPING = {
  darwin: 'darwin',
  linux: 'linux', 
  win32: 'windows'
};

const ARCH_MAPPING = {
  x64: 'amd64',
  arm64: 'arm64'
};

function getPlatformInfo() {
  const platform = PLATFORM_MAPPING[process.platform];
  const arch = ARCH_MAPPING[process.arch];
  
  if (!platform) {
    throw new Error(`Unsupported platform: ${process.platform}`);
  }
  
  if (!arch) {
    throw new Error(`Unsupported architecture: ${process.arch}`);
  }
  
  return { platform, arch };
}

function makeExecutable(filePath) {
  if (process.platform !== 'win32') {
    fs.chmodSync(filePath, '755');
  }
}

function install() {
  try {
    const { platform, arch } = getPlatformInfo();
    
    // Source binary in the binaries directory
    const sourceBinaryName = `3pio-${platform}-${arch}${platform === 'windows' ? '.exe' : ''}`;
    const sourcePath = path.join(__dirname, 'binaries', sourceBinaryName);
    
    if (!fs.existsSync(sourcePath)) {
      throw new Error(`Binary not found for your platform: ${sourceBinaryName}`);
    }
    
    // Destination - replace the placeholder file with actual binary
    const destPath = path.join(__dirname, 'bin', '3pio');
    
    // Copy the appropriate binary to bin/3pio
    fs.copyFileSync(sourcePath, destPath);
    
    // Make it executable
    makeExecutable(destPath);
    
  } catch (error) {
    console.error('Failed to set up 3pio binary:', error.message);
    console.error('Please report this issue at https://github.com/zk/3pio/issues');
    process.exit(1);
  }
}

// Run the installation
install();