#!/bin/bash
#
# GANTRY Installer
# Gateway for AI Navigation, Telemetry, and Runtime Yield
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/mattabdou/gantry/main/install.sh | bash
#
# Or download and run locally:
#   chmod +x install.sh && ./install.sh
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="mattabdou/gantry"
BINARY_NAME="gantry"
VERSION="1.1.0"

# Print colored output
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     OS="linux" ;;
        Darwin*)    OS="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
        *)          error "Unsupported operating system: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   ARCH="amd64" ;;
        arm64|aarch64)  ARCH="arm64" ;;
        *)              error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Determine install directory
determine_install_dir() {
    # Try /usr/local/bin first (requires sudo)
    if [ -w /usr/local/bin ]; then
        INSTALL_DIR="/usr/local/bin"
        USE_SUDO=""
    elif command -v sudo &> /dev/null && sudo -n true 2>/dev/null; then
        INSTALL_DIR="/usr/local/bin"
        USE_SUDO="sudo"
    elif [ -d "$HOME/.local/bin" ]; then
        INSTALL_DIR="$HOME/.local/bin"
        USE_SUDO=""
    else
        # Create ~/.local/bin if it doesn't exist
        mkdir -p "$HOME/.local/bin"
        INSTALL_DIR="$HOME/.local/bin"
        USE_SUDO=""
    fi
}

# Check if ~/.local/bin is in PATH
check_path() {
    if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
        if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
            warn "~/.local/bin is not in your PATH"
            echo ""
            echo "Add it to your shell profile:"
            echo ""
            echo "  # For bash (~/.bashrc):"
            echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
            echo ""
            echo "  # For zsh (~/.zshrc):"
            echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
            echo ""
            PATH_WARNING=true
        fi
    fi
}

# Download binary from GitHub releases
download_binary() {
    local url="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${BINARY_NAME}-${OS}-${ARCH}"

    if [ "$OS" = "windows" ]; then
        url="${url}.exe"
    fi

    info "Downloading gantry for ${OS}/${ARCH}..."

    # Check if we're running from the repo with pre-built binaries
    local local_binary="build/${BINARY_NAME}-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        local_binary="${local_binary}.exe"
    fi

    if [ -f "$local_binary" ]; then
        info "Using local binary: $local_binary"
        BINARY_PATH="$local_binary"
        return
    fi

    # Download from GitHub
    local tmp_file=$(mktemp)

    if command -v curl &> /dev/null; then
        if ! curl -fsSL "$url" -o "$tmp_file" 2>/dev/null; then
            # If release doesn't exist, try to build from source
            rm -f "$tmp_file"
            build_from_source
            return
        fi
    elif command -v wget &> /dev/null; then
        if ! wget -q "$url" -O "$tmp_file" 2>/dev/null; then
            rm -f "$tmp_file"
            build_from_source
            return
        fi
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    BINARY_PATH="$tmp_file"
}

# Build from source if download fails
build_from_source() {
    if ! command -v go &> /dev/null; then
        error "Could not download binary and Go is not installed. Please install Go 1.21+ or download a release from https://github.com/${GITHUB_REPO}/releases"
    fi

    info "Building from source..."

    local tmp_dir=$(mktemp -d)
    local current_dir=$(pwd)

    # Check if we're in the gantry repo
    if [ -f "go.mod" ] && grep -q "github.com/mattabdou/gantry" go.mod 2>/dev/null; then
        info "Building from current directory..."
        go build -ldflags "-s -w" -o "$tmp_dir/gantry" .
    else
        # Clone and build
        info "Cloning repository..."
        git clone --depth 1 "https://github.com/${GITHUB_REPO}.git" "$tmp_dir/src"
        cd "$tmp_dir/src"
        go build -ldflags "-s -w" -o "$tmp_dir/gantry" .
        cd "$current_dir"
    fi

    BINARY_PATH="$tmp_dir/gantry"
}

# Install the binary
install_binary() {
    local dest="${INSTALL_DIR}/${BINARY_NAME}"

    info "Installing to ${dest}..."

    if [ -n "$USE_SUDO" ]; then
        sudo cp "$BINARY_PATH" "$dest"
        sudo chmod +x "$dest"
    else
        cp "$BINARY_PATH" "$dest"
        chmod +x "$dest"
    fi

    # Clean up temp file if it was a download
    if [[ "$BINARY_PATH" == /tmp/* ]]; then
        rm -f "$BINARY_PATH"
    fi

    success "Installed gantry to ${dest}"
}

# Verify installation
verify_installation() {
    if command -v gantry &> /dev/null; then
        local version=$(gantry --version 2>/dev/null || echo "unknown")
        success "Verified: $version"
    elif [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        local version=$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>/dev/null || echo "unknown")
        success "Verified: $version"
    else
        warn "Installation completed but gantry is not in PATH yet"
    fi
}

# Initialize configuration
initialize_config() {
    local config_file="$HOME/.gantryrc.json"

    echo ""
    if [ -f "$config_file" ]; then
        info "Using existing configuration file: $config_file"
    else
        info "Initializing configuration..."

        # Run gantry init
        if command -v gantry &> /dev/null; then
            gantry init
        elif [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
            "${INSTALL_DIR}/${BINARY_NAME}" init
        fi
    fi
}

# Show post-install instructions
show_instructions() {
    echo ""
    echo "=========================================="
    echo "  GANTRY Installation Complete!"
    echo "=========================================="
    echo ""

    if [ "${PATH_WARNING:-false}" = true ]; then
        warn "Don't forget to add ~/.local/bin to your PATH (see above)"
        echo ""
    fi

    # Check for GANTRY_USERNAME
    if [ -z "$GANTRY_USERNAME" ]; then
        warn "GANTRY_USERNAME environment variable is not set"
        echo ""
        echo "You must set GANTRY_USERNAME in your shell profile:"
        echo ""
        echo "  # For bash (~/.bashrc):"
        echo "  export GANTRY_USERNAME=\"your.username\""
        echo ""
        echo "  # For zsh (~/.zshrc):"
        echo "  export GANTRY_USERNAME=\"your.username\""
        echo ""
    else
        success "GANTRY_USERNAME is set to: $GANTRY_USERNAME"
    fi

    echo "Next steps:"
    echo "  1. Edit ~/.gantryrc.json to configure your OTEL endpoint and credentials"
    echo "  2. Optionally create .gantry.json in your project directories"
    echo "  3. Run 'gantry' instead of 'claude' to launch Claude Code"
    echo ""
    echo "For more information:"
    echo "  gantry --help"
    echo "  https://github.com/${GITHUB_REPO}"
    echo ""
}

# Main installation flow
main() {
    echo ""
    echo "=========================================="
    echo "  GANTRY Installer"
    echo "  Gateway for AI Navigation, Telemetry,"
    echo "  and Runtime Yield"
    echo "=========================================="
    echo ""

    detect_os
    detect_arch
    info "Detected: ${OS}/${ARCH}"

    determine_install_dir
    info "Install directory: ${INSTALL_DIR}"

    check_path
    download_binary
    install_binary
    verify_installation
    initialize_config
    show_instructions
}

# Run main
main "$@"
