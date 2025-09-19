<#
Purpose: Provision a Windows 11 ARM dev VM for 3pio work.
Actions:
- Enable long paths
- Install Git, Go, Node.js LTS, Python 3.11, pnpm via winget
- Configure Git for LF line endings on commit
- Print versions and basic sanity checks

Run as: PowerShell (Admin)
  powershell -ExecutionPolicy Bypass -File scripts\windows\bootstrap.ps1
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Ensure-Admin {
  $currentIdentity = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = New-Object Security.Principal.WindowsPrincipal($currentIdentity)
  if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host 'Elevating to Administrator...' -ForegroundColor Yellow
    Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs
    exit 0
  }
}

function Set-LongPathsEnabled {
  Write-Host 'Enabling NTFS long paths (requires reboot to fully apply in some cases)...' -ForegroundColor Cyan
  New-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem' -Name 'LongPathsEnabled' -PropertyType DWord -Value 1 -Force | Out-Null
}

function Ensure-Winget {
  if (Get-Command winget -ErrorAction SilentlyContinue) { return }
  Write-Warning 'winget not found. Install "App Installer" from Microsoft Store, then re-run this script.'
  throw 'winget missing'
}

function Install-WithWinget {
  param([Parameter(Mandatory=$true)][string]$Id)
  Write-Host ("Installing {0} via winget..." -f $Id) -ForegroundColor Cyan
  # -h to suppress installer UI, accept agreements
  winget install -e --id $Id --accept-package-agreements --accept-source-agreements -h | Out-Null
}

function Configure-Git {
  Write-Host 'Configuring Git line endings (autocrlf=input)...' -ForegroundColor Cyan
  git config --global core.autocrlf input
  git config --global core.safecrlf false
}

function Print-Versions {
  Write-Host "\nInstalled tool versions:" -ForegroundColor Green
  try { git --version } catch { Write-Warning 'git missing from PATH' }
  try { go version } catch { Write-Warning 'go missing from PATH' }
  try { node -v } catch { Write-Warning 'node missing from PATH' }
  try { npm -v } catch { Write-Warning 'npm missing from PATH' }
  try { pnpm -v } catch { Write-Warning 'pnpm missing from PATH' }
  try { python --version } catch { Write-Warning 'python missing from PATH' }
}

function Main {
  Ensure-Admin
  Set-LongPathsEnabled
  Ensure-Winget

  # Core tooling
  $ids = @(
    'Git.Git',
    'GoLang.Go',
    'OpenJS.NodeJS.LTS',
    'Python.Python.3.11',
    'pnpm.pnpm'
  )
  foreach ($id in $ids) { Install-WithWinget -Id $id }

  # Optional: MSYS2 for make (commented by default)
  <#
  Install-WithWinget -Id 'MSYS2.MSYS2'
  $msysBash = 'C:\\msys64\\usr\\bin\\bash.exe'
  if (Test-Path $msysBash) {
    Write-Host 'Installing make via MSYS2 pacman (this may take several minutes and might require reruns after a full MSYS2 update)...' -ForegroundColor Cyan
    & $msysBash -lc 'pacman -Sy --noconfirm && pacman -S --needed --noconfirm make'
    Write-Host 'MSYS2 make installed. Use: "C:\\msys64\\usr\\bin\\make.exe"' -ForegroundColor Green
  }
  #>

  Configure-Git
  Print-Versions

  Write-Host "\nDone. If tools report path/permissions issues, reboot the VM and retry." -ForegroundColor Green
}

Main

