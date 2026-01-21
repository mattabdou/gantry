#Requires -Version 5.1
<#
.SYNOPSIS
    GANTRY Installer for Windows
    Gateway for AI Navigation, Telemetry, and Runtime Yield

.DESCRIPTION
    Downloads and installs GANTRY, a launcher for Claude Code that configures
    environment and telemetry.

.EXAMPLE
    irm https://raw.githubusercontent.com/mattabdou/gantry/main/install.ps1 | iex

.EXAMPLE
    .\install.ps1

.LINK
    https://github.com/mattabdou/gantry
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

# Configuration
$GithubRepo = "mattabdou/gantry"
$BinaryName = "gantry"
$Version = "1.1.3"

# Colors
function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Blue }
function Write-Success { Write-Host "[SUCCESS] $args" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Error { Write-Host "[ERROR] $args" -ForegroundColor Red }

function Get-Architecture {
    # Use PROCESSOR_ARCHITECTURE env var for maximum compatibility
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Get-InstallDirectory {
    # Try common locations in order of preference
    $locations = @(
        "$env:LOCALAPPDATA\Programs\gantry"
        "$env:USERPROFILE\.local\bin"
        "$env:USERPROFILE\bin"
    )

    foreach ($loc in $locations) {
        if (Test-Path $loc) {
            return $loc
        }
    }

    # Default to first option, will create it
    return $locations[0]
}

function Test-InPath {
    param([string]$Directory)

    $paths = $env:PATH -split ';'
    return $paths -contains $Directory
}

function Add-ToUserPath {
    param([string]$Directory)

    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$Directory*") {
        $newPath = "$Directory;$currentPath"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        $env:PATH = "$Directory;$env:PATH"
        return $true
    }
    return $false
}

function Get-Binary {
    param(
        [string]$Arch,
        [string]$DestDir
    )

    $binaryFileName = "$BinaryName-windows-$Arch.exe"
    $url = "https://github.com/$GithubRepo/releases/download/v$Version/$binaryFileName"
    $destPath = Join-Path $DestDir "$BinaryName.exe"

    # Check if we're running from repo with pre-built binaries
    $localBinary = Join-Path $PWD "build\$binaryFileName"
    if (Test-Path $localBinary) {
        Write-Info "Using local binary: $localBinary"
        Copy-Item $localBinary $destPath -Force
        return $destPath
    }

    Write-Info "Downloading gantry for windows/$Arch..."

    try {
        # Try to download from GitHub releases
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($url, $destPath)
        return $destPath
    }
    catch {
        Write-Warn "Could not download from GitHub releases: $_"

        # Try to build from source if Go is available
        if (Get-Command go -ErrorAction SilentlyContinue) {
            Write-Info "Attempting to build from source..."
            return Build-FromSource -DestDir $DestDir
        }

        throw "Could not download binary and Go is not installed. Please download manually from https://github.com/$GithubRepo/releases"
    }
}

function Build-FromSource {
    param([string]$DestDir)

    $destPath = Join-Path $DestDir "$BinaryName.exe"

    # Check if we're in the gantry repo
    if (Test-Path "go.mod") {
        $goMod = Get-Content "go.mod" -Raw
        if ($goMod -like "*github.com/mattabdou/gantry*") {
            Write-Info "Building from current directory..."
            go build -ldflags "-s -w" -o $destPath .
            return $destPath
        }
    }

    # Clone and build
    $tempDir = Join-Path $env:TEMP "gantry-build-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

    try {
        Write-Info "Cloning repository..."
        git clone --depth 1 "https://github.com/$GithubRepo.git" $tempDir
        Push-Location $tempDir
        go build -ldflags "-s -w" -o $destPath .
        Pop-Location
        return $destPath
    }
    finally {
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Initialize-Config {
    $configFile = Join-Path $env:USERPROFILE ".gantryrc.json"

    Write-Host ""
    if (Test-Path $configFile) {
        Write-Info "Using existing configuration file: $configFile"
        return $true  # Config already existed
    }
    else {
        Write-Info "Initializing configuration..."

        # Run gantry init
        $gantryPath = Get-Command gantry -ErrorAction SilentlyContinue
        if ($gantryPath) {
            & gantry init
        }
        return $false  # Config was newly created
    }
}

function Show-PostInstallInstructions {
    param(
        [bool]$PathAdded,
        [string]$InstallDir,
        [bool]$ConfigExisted
    )

    Write-Host ""
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host "  GANTRY Installation Complete!" -ForegroundColor Cyan
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host ""

    if ($PathAdded) {
        Write-Warn "Added $InstallDir to your PATH"
        Write-Host "  You may need to restart your terminal for PATH changes to take effect."
        Write-Host ""
    }

    # Check for GANTRY_USERNAME (only for new installations)
    # Skip this check if config already existed, as username is likely already configured
    if (-not $ConfigExisted) {
        if (-not $env:GANTRY_USERNAME) {
            Write-Warn "GANTRY_USERNAME environment variable is not set"
            Write-Host ""
            Write-Host "You can set your username in ~/.gantryrc.json (gantry.username)"
            Write-Host "or set GANTRY_USERNAME. Choose one method:"
            Write-Host ""
            Write-Host "  # PowerShell (current session):"
            Write-Host '  $env:GANTRY_USERNAME = "your.username"'
            Write-Host ""
            Write-Host "  # PowerShell (permanent - add to $PROFILE):"
            Write-Host '  $env:GANTRY_USERNAME = "your.username"'
            Write-Host ""
            Write-Host "  # System Environment Variables (permanent):"
            Write-Host '  [Environment]::SetEnvironmentVariable("GANTRY_USERNAME", "your.username", "User")'
            Write-Host ""
            Write-Host "  # Or use: Settings > System > About > Advanced system settings > Environment Variables"
            Write-Host ""
        }
        else {
            Write-Success "GANTRY_USERNAME is set to: $env:GANTRY_USERNAME"
        }
    }

    Write-Host "Next steps:"
    Write-Host "  1. Edit $env:USERPROFILE\.gantryrc.json to configure your OTEL endpoint and credentials"
    Write-Host "  2. Optionally create .gantry.json in your project directories"
    Write-Host "  3. Run 'gantry' instead of 'claude' to launch Claude Code"
    Write-Host ""
    Write-Host "For more information:"
    Write-Host "  gantry --help"
    Write-Host "  https://github.com/$GithubRepo"
    Write-Host ""
}

function Main {
    Write-Host ""
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host "  GANTRY Installer for Windows" -ForegroundColor Cyan
    Write-Host "  Gateway for AI Navigation, Telemetry," -ForegroundColor Cyan
    Write-Host "  and Runtime Yield" -ForegroundColor Cyan
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host ""

    # Detect architecture
    $arch = Get-Architecture
    Write-Info "Detected architecture: $arch"

    # Determine install directory
    $installDir = Get-InstallDirectory
    Write-Info "Install directory: $installDir"

    # Create install directory if it doesn't exist
    if (-not (Test-Path $installDir)) {
        Write-Info "Creating directory: $installDir"
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    # Download/copy binary
    $binaryPath = Get-Binary -Arch $arch -DestDir $installDir
    Write-Success "Installed gantry to $binaryPath"

    # Add to PATH if needed
    $pathAdded = $false
    if (-not (Test-InPath $installDir)) {
        Write-Info "Adding $installDir to PATH..."
        $pathAdded = Add-ToUserPath $installDir
    }

    # Show installation result
    Write-Success "Installed to: $binaryPath"
    if ($pathAdded -or -not (Test-InPath $installDir)) {
        Write-Warn "Restart your terminal to use 'gantry' command"
    }

    # Initialize config
    $configExisted = Initialize-Config

    # Show instructions
    Show-PostInstallInstructions -PathAdded $pathAdded -InstallDir $installDir -ConfigExisted $configExisted
}

# Run main
Main
