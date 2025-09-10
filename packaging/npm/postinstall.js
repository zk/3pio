#!/usr/bin/env node

const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');
const { pipeline } = require('stream');
const { promisify } = require('util');
const zlib = require('zlib');
const { execSync } = require('child_process');

const pipelineAsync = promisify(pipeline);

// Package version (should match the npm package version)
const PACKAGE_VERSION = require('./package.json').version;

// GitHub repository details
const GITHUB_OWNER = 'zk';
const GITHUB_REPO = '3pio';

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

function getBinaryUrl(version, platform, arch) {
  const baseUrl = `https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download`;
  const extension = platform === 'windows' ? 'zip' : 'tar.gz';
  const filename = `3pio-${platform}-${arch}.${extension}`;
  return `${baseUrl}/v${version}/${filename}`;
}

async function downloadFile(url, destination) {
  return new Promise((resolve, reject) => {
    const protocol = url.startsWith('https:') ? https : http;
    
    protocol.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirects
        return downloadFile(response.headers.location, destination).then(resolve, reject);
      }
      
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download: ${response.statusCode} ${response.statusMessage}`));
        return;
      }
      
      const fileStream = fs.createWriteStream(destination);
      response.pipe(fileStream);
      
      fileStream.on('finish', () => {
        fileStream.close();
        resolve();
      });
      
      fileStream.on('error', reject);
    }).on('error', reject);
  });
}

async function extractTarGz(source, destination) {
  try {
    // Use system tar command for reliable extraction
    execSync(`tar -xzf "${source}" -C "${destination}"`, { stdio: 'pipe' });
  } catch (error) {
    throw new Error(`Failed to extract tar.gz: ${error.message}`);
  }
}

async function extractZip(source, destination) {
  try {
    if (process.platform === 'win32') {
      // Use PowerShell on Windows
      execSync(`powershell -command "Expand-Archive -Path '${source}' -DestinationPath '${destination}' -Force"`, { stdio: 'pipe' });
    } else {
      // Use unzip on Unix systems
      execSync(`unzip -o "${source}" -d "${destination}"`, { stdio: 'pipe' });
    }
  } catch (error) {
    throw new Error(`Failed to extract zip: ${error.message}`);
  }
}

async function makeExecutable(filePath) {
  if (process.platform !== 'win32') {
    fs.chmodSync(filePath, '755');
  }
}

async function install() {
  try {
    console.log('Installing 3pio binary...');
    
    const { platform, arch } = getPlatformInfo();
    console.log(`Platform: ${platform}, Architecture: ${arch}`);
    
    // Create bin directory
    const binDir = path.join(__dirname, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }
    
    const url = getBinaryUrl(PACKAGE_VERSION, platform, arch);
    console.log(`Downloading from: ${url}`);
    
    const extension = platform === 'windows' ? 'zip' : 'tar.gz';
    const downloadPath = path.join(__dirname, `3pio-${platform}-${arch}.${extension}`);
    
    // Download the archive
    await downloadFile(url, downloadPath);
    console.log('Download completed');
    
    // Extract the archive
    console.log('Extracting binary...');
    if (extension === 'tar.gz') {
      await extractTarGz(downloadPath, binDir);
    } else {
      await extractZip(downloadPath, binDir);
    }
    
    // Make the binary executable
    const binaryName = platform === 'windows' ? '3pio.exe' : '3pio';
    const binaryPath = path.join(binDir, binaryName);
    
    if (!fs.existsSync(binaryPath)) {
      throw new Error(`Binary not found after extraction: ${binaryPath}`);
    }
    
    await makeExecutable(binaryPath);
    
    // Clean up downloaded archive
    fs.unlinkSync(downloadPath);
    
    console.log('✅ 3pio binary installed successfully');
    
  } catch (error) {
    console.error('❌ Failed to install 3pio binary:', error.message);
    process.exit(1);
  }
}

// Run the installation
install();