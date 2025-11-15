# install.ps1 - FabricX Runtime installer for Windows

param(
    [string]$Version = "0.1.0",
    [string]$InstallDir = "$env:LOCALAPPDATA\FabricX"
)

$ErrorActionPreference = "Stop"

Write-Host "🚀 FabricX Runtime Installer v$Version" -ForegroundColor Green
Write-Host ""

# Check Docker
Write-Host "🐳 Checking Docker..." -ForegroundColor Cyan
try {
    docker ps | Out-Null
    Write-Host "✓ Docker is available" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker is not running" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please start Docker Desktop and try again"
    exit 1
}

# Download
$BinaryName = "fabricx-runtime-$Version-windows-amd64.zip"
$DownloadUrl = "https://github.com/temmyjay001/fabricx/releases/download/v$Version/$BinaryName"

Write-Host ""
Write-Host "📥 Downloading FabricX Runtime..." -ForegroundColor Cyan
Write-Host "   URL: $DownloadUrl"

$TempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
$ZipPath = Join-Path $TempDir $BinaryName

try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath
} catch {
    Write-Host "❌ Failed to download binary" -ForegroundColor Red
    Write-Host "Please check if version $Version exists"
    exit 1
}

# Extract
Write-Host "📦 Extracting..." -ForegroundColor Cyan
Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force

# Install
Write-Host "📂 Installing to $InstallDir..." -ForegroundColor Cyan
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

$BinaryFile = Get-ChildItem -Path $TempDir -Filter "fabricx-runtime-*.exe" | Select-Object -First 1
Copy-Item -Path $BinaryFile.FullName -Destination "$InstallDir\fabricx-runtime.exe" -Force

# Add to PATH if not already there
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    Write-Host "➕ Adding to PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

# Cleanup
Remove-Item -Path $TempDir -Recurse -Force

# Verify
Write-Host ""
Write-Host "✅ Installation successful!" -ForegroundColor Green
Write-Host ""
Write-Host "🎉 You can now start the runtime:"
Write-Host "   fabricx-runtime" -ForegroundColor Yellow
Write-Host ""
Write-Host "Or run in background:"
Write-Host "   Start-Process fabricx-runtime -WindowStyle Hidden" -ForegroundColor Yellow