#!/usr/bin/env bash
set -euo pipefail

echo "Installing UTM via Homebrew..."
if ! command -v brew >/dev/null 2>&1; then
  echo "Homebrew not found. Install from https://brew.sh and re-run." >&2
  exit 1
fi

brew install --cask utm

echo
echo "UTM installed. Launching UTM..."
open -a UTM || true

echo
echo "Next steps:" 
echo "1) In UTM, Create New → Virtualize → Windows → use Gallery to build Windows 11 ARM."
echo "2) After Windows setup, open this repo inside the VM and run:"
echo "     powershell -ExecutionPolicy Bypass -File scripts\\windows\\bootstrap.ps1"
echo "3) See docs/windows-dev.md for full instructions."

