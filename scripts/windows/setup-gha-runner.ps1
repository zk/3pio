<#
Purpose: Install and register a GitHub Actions self-hosted runner on Windows (ARM Mac VM).

Usage (Admin recommended for service install):
  powershell -ExecutionPolicy Bypass -File scripts\windows\setup-gha-runner.ps1 -Repo "owner/repo" -Token "<registration token>"

Notes
- Generate a registration token from: GitHub → Repo → Settings → Actions → Runners → New self-hosted runner → Windows → x64 → Copy token.
- Token is short-lived; do not commit it. You can also set it via env var GHA_RUNNER_TOKEN.
- On Windows ARM, we use the win-x64 runner (works via x64 emulation).
#>

param(
  [Parameter(Mandatory=$true)][string]$Repo,
  [string]$Token = $env:GHA_RUNNER_TOKEN,
  [string]$RunnerName = "$($env:COMPUTERNAME)-local",
  [string]$Labels = 'self-hosted,windows,local',
  [string]$WorkDir = "$PSScriptRoot\_work"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Ensure-AdminOption {
  $id = [Security.Principal.WindowsIdentity]::GetCurrent()
  $p = New-Object Security.Principal.WindowsPrincipal($id)
  return $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not $Token) { throw 'Token is required (pass -Token or set env:GHA_RUNNER_TOKEN).' }

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$root = Join-Path $PSScriptRoot 'gha-runner'
New-Item -ItemType Directory -Force -Path $root | Out-Null
Push-Location $root

try {
  Write-Host 'Fetching latest Actions runner release metadata...' -ForegroundColor Cyan
  $headers = @{ 'User-Agent' = '3pio-setup' }
  $release = Invoke-RestMethod -Headers $headers -Uri 'https://api.github.com/repos/actions/runner/releases/latest'
  $asset = $release.assets | Where-Object { $_.name -like '*win-x64*.zip' } | Select-Object -First 1
  if (-not $asset) { throw 'Could not find win-x64 runner asset.' }

  $zip = Join-Path $root $asset.name
  if (-not (Test-Path $zip)) {
    Write-Host ('Downloading {0}...' -f $asset.name) -ForegroundColor Cyan
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zip
  }

  Write-Host 'Extracting runner...' -ForegroundColor Cyan
  Expand-Archive -Path $zip -DestinationPath $root -Force

  $cfgArgs = @(
    'config.cmd',
    '--url', "https://github.com/$Repo",
    '--token', $Token,
    '--name', $RunnerName,
    '--labels', $Labels,
    '--work', $WorkDir,
    '--unattended'
  )

  Write-Host 'Configuring runner...' -ForegroundColor Cyan
  & .\config.cmd @($cfgArgs[1..$cfgArgs.Length])

  if (Ensure-AdminOption) {
    Write-Host 'Installing runner as a Windows service...' -ForegroundColor Cyan
    .\svc install
    .\svc start
    Write-Host 'Runner service installed and started.' -ForegroundColor Green
  } else {
    Write-Warning 'Not elevated. The runner will need to be started manually in this window with .\run.cmd, or re-run this script as Admin to install the service.'
    Write-Host 'Starting interactive runner (press Ctrl+C to stop)...' -ForegroundColor Yellow
    .\run.cmd
  }

  Write-Host 'Done. Verify the runner appears in GitHub → Settings → Actions → Runners.' -ForegroundColor Green
}
finally {
  Pop-Location
}

