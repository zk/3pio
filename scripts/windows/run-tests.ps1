<#
Run 3pio build + Go unit & integration tests on Windows and write results.

Usage:
  powershell -ExecutionPolicy Bypass -File scripts\windows\run-tests.ps1 -Repo "C:\Users\you\code\3pio" -OutDir "C:\Users\you\code\3pio\.3pio\windows-run"

Outputs:
  - unit.json: go test -json ./...
  - integration.json: go test -json ./tests/integration_go/...
  - summary.json: exit codes and timestamps
  - raw logs are captured within the JSON streams
#>

[CmdletBinding()]
param(
  [Parameter(Mandatory=$true)][string]$Repo,
  [Parameter(Mandatory=$true)][string]$OutDir
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

function Ensure-Dir([string]$p) {
  if (-not (Test-Path -Path $p)) { New-Item -ItemType Directory -Force -Path $p | Out-Null }
}

Ensure-Dir -p $OutDir

Push-Location $Repo
try {
  $ts = Get-Date -Format 'yyyy-MM-ddTHH-mm-ss'
  $unitLog = Join-Path $OutDir "unit-$ts.json"
  $intLog  = Join-Path $OutDir "integration-$ts.json"
  $summary = Join-Path $OutDir "summary-$ts.json"

  Write-Host "Building 3pio..." -ForegroundColor Cyan
  & go build -o "build\3pio.exe" "cmd\3pio\main.go"
  $codeBuild = $LASTEXITCODE

  Write-Host "Running unit tests (JSON)..." -ForegroundColor Cyan
  & go test -json ./... *> $unitLog
  $codeUnit = $LASTEXITCODE

  Write-Host "Running integration tests (JSON)..." -ForegroundColor Cyan
  & go test -json ./tests/integration_go/... *> $intLog
  $codeInt = $LASTEXITCODE

  $result = [pscustomobject]@{
    timestamp     = $ts
    buildExitCode = $codeBuild
    unitExitCode  = $codeUnit
    intExitCode   = $codeInt
    outDir        = $OutDir
    repo          = $Repo
  }
  $result | ConvertTo-Json | Out-File -Encoding UTF8 -FilePath $summary

  $exit = if ($codeBuild -ne 0 -or $codeUnit -ne 0 -or $codeInt -ne 0) { 1 } else { 0 }
  exit $exit
}
finally {
  Pop-Location
}

