@echo off
setlocal enabledelayedexpansion

:: GANTRY Installer for Windows (CMD)
:: Gateway for AI Navigation, Telemetry, and Runtime Yield
::
:: Usage:
::   curl -fsSL https://raw.githubusercontent.com/mattabdou/gantry/main/install.cmd -o install.cmd && install.cmd && del install.cmd

set "GITHUB_REPO=mattabdou/gantry"
set "BINARY_NAME=gantry"

echo.
echo ==========================================
echo   GANTRY Installer for Windows
echo   Gateway for AI Navigation, Telemetry,
echo   and Runtime Yield
echo ==========================================
echo.

:: Detect architecture
set "ARCH="
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" set "ARCH=amd64"
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"

if "%ARCH%"=="" (
    echo [ERROR] Unsupported architecture: %PROCESSOR_ARCHITECTURE%
    exit /b 1
)

echo [INFO] Detected architecture: %ARCH%

:: Check for curl
where curl >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] curl is required but not found. Please install curl or use the PowerShell installer.
    exit /b 1
)

:: Fetch latest version from GitHub API
echo [INFO] Fetching latest version...
set "API_URL=https://api.github.com/repos/%GITHUB_REPO%/releases/latest"

:: Create temp file for API response
set "TEMP_FILE=%TEMP%\gantry_version_%RANDOM%.txt"

curl -sL "%API_URL%" -o "%TEMP_FILE%" 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Could not fetch latest version from GitHub.
    del "%TEMP_FILE%" 2>nul
    exit /b 1
)

:: Extract version using findstr (look for "tag_name": "vX.Y.Z")
for /f "tokens=2 delims=:," %%a in ('findstr /C:"\"tag_name\"" "%TEMP_FILE%"') do (
    set "VERSION_RAW=%%a"
)
del "%TEMP_FILE%" 2>nul

:: Clean up version string (remove quotes, spaces, and 'v' prefix)
set "VERSION=%VERSION_RAW:"=%"
set "VERSION=%VERSION: =%"
set "VERSION=%VERSION:v=%"

if "%VERSION%"=="" (
    echo [ERROR] Could not parse version from GitHub API response.
    exit /b 1
)

echo [INFO] Latest version: %VERSION%

:: Set install directory
set "INSTALL_DIR=%LOCALAPPDATA%\Programs\gantry"

:: Create install directory if it doesn't exist
if not exist "%INSTALL_DIR%" (
    echo [INFO] Creating directory: %INSTALL_DIR%
    mkdir "%INSTALL_DIR%"
)

:: Download binary
set "BINARY_FILE=%BINARY_NAME%-windows-%ARCH%.exe"
set "DOWNLOAD_URL=https://github.com/%GITHUB_REPO%/releases/download/v%VERSION%/%BINARY_FILE%"
set "DEST_PATH=%INSTALL_DIR%\%BINARY_NAME%.exe"

echo [INFO] Downloading gantry for windows/%ARCH%...
echo [INFO] URL: %DOWNLOAD_URL%

curl -fsSL "%DOWNLOAD_URL%" -o "%DEST_PATH%"
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Failed to download gantry. Please check your internet connection.
    echo [ERROR] You can manually download from: https://github.com/%GITHUB_REPO%/releases
    exit /b 1
)

echo [SUCCESS] Downloaded gantry to %DEST_PATH%

:: Check if install directory is in PATH
echo %PATH% | findstr /I /C:"%INSTALL_DIR%" >nul
if %ERRORLEVEL% neq 0 (
    echo [INFO] Adding %INSTALL_DIR% to user PATH...

    :: Get current user PATH
    for /f "tokens=2*" %%a in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USER_PATH=%%b"

    :: Add to user PATH if not already there
    if defined USER_PATH (
        setx PATH "%INSTALL_DIR%;!USER_PATH!" >nul 2>nul
    ) else (
        setx PATH "%INSTALL_DIR%" >nul 2>nul
    )

    if %ERRORLEVEL% equ 0 (
        echo [SUCCESS] Added %INSTALL_DIR% to PATH
        set "PATH_ADDED=1"
    ) else (
        echo [WARN] Could not add to PATH automatically.
        echo [WARN] Please add %INSTALL_DIR% to your PATH manually.
        set "PATH_ADDED=1"
    )
) else (
    set "PATH_ADDED=0"
)

:: Initialize config if it doesn't exist
set "CONFIG_FILE=%USERPROFILE%\.gantryrc.json"
set "CONFIG_EXISTED=0"

echo.
if exist "%CONFIG_FILE%" (
    echo [INFO] Using existing configuration file: %CONFIG_FILE%
    set "CONFIG_EXISTED=1"
) else (
    echo [INFO] Initializing configuration...
    :: Try to run gantry init
    "%DEST_PATH%" init >nul 2>nul
)

:: Show post-install instructions
echo.
echo ==========================================
echo   GANTRY Installation Complete!
echo ==========================================
echo.

if "%PATH_ADDED%"=="1" (
    echo [WARN] PATH was updated. Please restart your terminal for changes to take effect.
    echo.
)

echo Next steps:
echo   1. Restart your terminal ^(if PATH was updated^)
echo   2. Edit %USERPROFILE%\.gantryrc.json to configure your OTEL endpoint and credentials
echo   3. Optionally create .gantry.json in your project directories
echo   4. Run 'gantry' instead of 'claude' to launch Claude Code
echo.
echo For more information:
echo   gantry --help
echo   https://github.com/%GITHUB_REPO%
echo.

endlocal
