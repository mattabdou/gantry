#Requires -Version 5.1
<#
.SYNOPSIS
    GANTRY Uninstaller for Windows
    Gateway for AI Navigation, Telemetry, and Runtime Yield

.DESCRIPTION
    Removes GANTRY from your system, including the binary and optionally the config file.

.EXAMPLE
    irm https://raw.githubusercontent.com/mattabdou/gantry/main/uninstall.ps1 | iex

.EXAMPLE
    .\uninstall.ps1

.LINK
    https://github.com/mattabdou/gantry
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

# Colors
function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Blue }
function Write-Success { Write-Host "[SUCCESS] $args" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Error { Write-Host "[ERROR] $args" -ForegroundColor Red }

function Find-Gantry {
    # Check common locations
    $locations = @(
        "$env:LOCALAPPDATA\Programs\gantry\gantry.exe"
        "$env:USERPROFILE\.local\bin\gantry.exe"
        "$env:USERPROFILE\bin\gantry.exe"
    )

    foreach ($loc in $locations) {
        if (Test-Path $loc) {
            return $loc
        }
    }

    # Try Get-Command
    $cmd = Get-Command gantry -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }

    return $null
}

function Get-InstallDirectory {
    param([string]$BinaryPath)
    return Split-Path -Parent $BinaryPath
}

function Remove-FromUserPath {
    param([string]$Directory)

    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -like "*$Directory*") {
        $paths = $currentPath -split ';' | Where-Object { $_ -ne $Directory -and $_ -ne "" }
        $newPath = $paths -join ';'
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        return $true
    }
    return $false
}

function Remove-GantryBinary {
    param([string]$BinaryPath)

    if (-not $BinaryPath) {
        Write-Warn "Gantry binary not found"
        return $null
    }

    Write-Info "Found gantry at: $BinaryPath"

    $installDir = Get-InstallDirectory $BinaryPath

    # Remove the binary
    Remove-Item $BinaryPath -Force
    Write-Success "Removed gantry binary"

    # Remove install directory if empty
    if (Test-Path $installDir) {
        $remaining = Get-ChildItem $installDir -ErrorAction SilentlyContinue
        if (-not $remaining) {
            Remove-Item $installDir -Force
            Write-Info "Removed empty install directory: $installDir"
        }
    }

    return $installDir
}

function Remove-GantryFromPath {
    param([string]$InstallDir)

    if (-not $InstallDir) {
        return
    }

    if (Remove-FromUserPath $InstallDir) {
        Write-Success "Removed $InstallDir from PATH"
        Write-Warn "You may need to restart your terminal for PATH changes to take effect"
    }
}

function Remove-GantryConfig {
    $configFile = Join-Path $env:USERPROFILE ".gantryrc.json"

    if (Test-Path $configFile) {
        Write-Host ""
        $response = Read-Host "Remove global config file ($configFile)? [y/N]"
        if ($response -match '^[Yy]') {
            Remove-Item $configFile -Force
            Write-Success "Removed $configFile"
        }
        else {
            Write-Info "Keeping $configFile"
        }
    }
    else {
        Write-Info "No global config file found"
    }
}

function Show-Completion {
    Write-Host ""
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host "  GANTRY Uninstallation Complete!" -ForegroundColor Cyan
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Note: Any .gantry.json project config files were not removed."
    Write-Host "You can manually delete them from your project directories if needed."
    Write-Host ""
}

function Main {
    Write-Host ""
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host "  GANTRY Uninstaller for Windows" -ForegroundColor Cyan
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host ""

    # Find and remove gantry
    $gantryPath = Find-Gantry
    $installDir = Remove-GantryBinary $gantryPath

    # Remove from PATH
    Remove-GantryFromPath $installDir

    # Ask about config removal
    Remove-GantryConfig

    # Show completion
    Show-Completion
}

# Run main
Main
