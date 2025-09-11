# Verify adapter files exist for embedding in Go binary

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$AdapterDir = Join-Path $ProjectRoot "internal\adapters"

Write-Host "Verifying adapters for embedding..."

# Check that adapter files exist
$MissingFiles = @()

$JestAdapter = Join-Path $AdapterDir "jest.js"
if (-not (Test-Path $JestAdapter)) {
    $MissingFiles += "jest.js"
}

$VitestAdapter = Join-Path $AdapterDir "vitest.js"
if (-not (Test-Path $VitestAdapter)) {
    $MissingFiles += "vitest.js"
}

$PytestAdapter = Join-Path $AdapterDir "pytest_adapter.py"
if (-not (Test-Path $PytestAdapter)) {
    $MissingFiles += "pytest_adapter.py"
}

# Report missing files
if ($MissingFiles.Count -gt 0) {
    Write-Host "Missing adapter files:" -ForegroundColor Red
    foreach ($file in $MissingFiles) {
        Write-Host "  - $AdapterDir\$file"
    }
    Write-Host ""
    Write-Host "These files should be committed to the repository."
    exit 1
}

Write-Host "All adapters present for embedding" -ForegroundColor Green
Get-ChildItem -Path $AdapterDir -Filter "*.js" | Select-Object Name, Length, LastWriteTime
Get-ChildItem -Path $AdapterDir -Filter "*.py" | Select-Object Name, Length, LastWriteTime