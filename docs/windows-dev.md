Windows on ARM (UTM) – Local Windows CI Repro on Apple Silicon

Overview
- Goal: Run Windows locally on an Apple Silicon Mac to reproduce CI failures and run 3pio tests.
- Tooling: UTM (free; QEMU/Apple Virtualization UI) virtualizing Windows 11 ARM.
- Result: A Windows 11 ARM VM with Go/Node/Python/pnpm and Git configured for cross‑platform dev.

Prereqs (macOS)
- Apple Silicon Mac (M1/M2/M3), 60+ GB free disk recommended.
- Homebrew (optional): https://brew.sh

Step 1 — Install UTM
- Easiest: `brew install --cask utm` (or download from https://mac.getutm.app)

Step 2 — Create the Windows 11 ARM VM
1) Launch UTM → Create New → Virtualize → Windows.
2) Use the Gallery option (recommended). Pick Windows 11 ARM and follow prompts. UTM will download/build an ARM64 ISO for you (via UUP) and set sane defaults.
   - If Gallery is unavailable, supply a Windows 11 ARM64 ISO obtained via UUP dump. Allocate: 4–8 vCPUs, 8–16 GB RAM, 100+ GB disk.
3) Start the VM and complete Windows setup. Install Windows updates until none remain.

Notes
- Performance: Windows ARM virtualized is fast for ARM; x64 apps run via Microsoft’s emulation (slower than native).
- Graphics: Limited acceleration; OK for dev/CI.
- Licensing: Activate Windows 11 with a valid license key (per Microsoft terms).

Step 3 — Networking & File Sharing
Option A (Recommended): SMB share from macOS
- macOS: System Settings → General → Sharing → File Sharing → enable. Add your project folder; click “i” → Options → enable SMB, check your user.
- Windows: Press Win+R → enter `\\<your-mac-hostname>.local` → authenticate → right‑click the shared folder → Map network drive (assign a letter).

Option B: Clone inside the VM
- Use Git inside Windows to clone the repo into `C:\Users\<you>\code\3pio` for simpler CRLF handling.

Step 4 — Bootstrap the Windows Dev Environment
Run the provided script as Administrator inside the VM:

1) Open PowerShell as Administrator.
2) From your repo directory (or after copying the script), run:
   `powershell -ExecutionPolicy Bypass -File scripts\windows\bootstrap.ps1`

What it does
- Enables long paths, installs Git, Go, Node.js LTS, Python 3.11, pnpm via winget.
- Sets Git to LF line endings on commit (`autocrlf=input`).
- Verifies versions and prints a short summary.

Optional — Add Make
- We avoid Make by default on Windows. If you want `make`, install MSYS2 via winget, then use pacman to add `make`. See comments inside `bootstrap.ps1` for commands and caveats.

Step 5 — Build and Test 3pio on Windows
Without Make
- Build: `go build -o build\3pio.exe cmd\3pio\main.go`
- Unit tests: `go test ./...`
- Integration tests: `go test ./tests/integration_go/...`

With Make (if installed)
- `make test-all` or `make test-integration`

Adapter usage (preferred)
- Use 3pio to wrap downstream tests, matching CI: e.g., `build\3pio.exe pnpm test`, `build\3pio.exe pytest`.

CI Parity Tips
- Match versions with your CI matrix (Go/Node/Python). `windows-latest` in GitHub Actions is Windows Server 2022 x64; Windows 11 ARM runs x64 apps via emulation, which is generally sufficient for logic bugs.
- Normalize newlines in your tests if asserting output (`\r\n` vs `\n`).
- Quote paths containing spaces. Prefer PowerShell scripts for Windows steps.

Optional — Self-hosted GitHub Actions Runner
- You can run your actual workflow locally in this VM. Use `scripts\windows\setup-gha-runner.ps1` and provide a GitHub registration token (never commit tokens). The script guides you through registration.

Troubleshooting
- Winget missing: Install “App Installer” from Microsoft Store, then retry bootstrap.
- `LongPathsEnabled`: If tools still fail on long paths, reboot the VM after bootstrap.
- SMB share slow: Clone the repo locally inside the VM instead.

Enable SSH for Host Automation (optional but recommended)
- Install and enable OpenSSH Server on Windows to allow the macOS host to start tests remotely and fetch results.
- PowerShell (Admin):
  - Install: `Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0`
  - Start now: `Start-Service sshd`
  - Auto-start: `Set-Service -Name sshd -StartupType Automatic`
  - Firewall: `New-NetFirewallRule -Name sshd -DisplayName "OpenSSH Server" -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22`
  - Find Windows username: `whoami`

Run Windows tests from macOS
- Use `scripts/macos/run-windows-tests.sh` to start the UTM VM (by bundle path or name), wait for SSH, run Go unit/integration tests inside Windows via PowerShell, and copy results back to `.3pio/windows-run/` on your Mac.
- See script `--help` for required flags (Windows host/IP, username, repo path in Windows).
