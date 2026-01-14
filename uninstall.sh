#!/bin/bash
#
# GANTRY Uninstaller
# Gateway for AI Navigation, Telemetry, and Runtime Yield
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/mattabdou/gantry/main/uninstall.sh | bash
#
# Or download and run locally:
#   chmod +x uninstall.sh && ./uninstall.sh
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Find gantry binary location
find_gantry() {
    # Check common locations
    local locations=(
        "/usr/local/bin/gantry"
        "$HOME/.local/bin/gantry"
        "$HOME/bin/gantry"
    )

    for loc in "${locations[@]}"; do
        if [ -f "$loc" ]; then
            GANTRY_PATH="$loc"
            return 0
        fi
    done

    # Try which command
    if command -v gantry &> /dev/null; then
        GANTRY_PATH=$(which gantry)
        return 0
    fi

    return 1
}

# Remove gantry binary
remove_binary() {
    if [ -z "$GANTRY_PATH" ]; then
        warn "Gantry binary not found"
        return
    fi

    info "Found gantry at: $GANTRY_PATH"

    # Check if we need sudo
    if [ -w "$GANTRY_PATH" ]; then
        rm -f "$GANTRY_PATH"
    elif command -v sudo &> /dev/null; then
        info "Removing requires sudo..."
        sudo rm -f "$GANTRY_PATH"
    else
        error "Cannot remove $GANTRY_PATH - permission denied"
    fi

    success "Removed gantry binary"
}

# Remove config files
remove_config() {
    local config_file="$HOME/.gantryrc.json"

    if [ -f "$config_file" ]; then
        echo ""
        read -p "Remove global config file ($config_file)? [y/N] " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -f "$config_file"
            success "Removed $config_file"
        else
            info "Keeping $config_file"
        fi
    else
        info "No global config file found"
    fi
}

# Show completion message
show_completion() {
    echo ""
    echo "=========================================="
    echo "  GANTRY Uninstallation Complete!"
    echo "=========================================="
    echo ""
    echo "Note: Any .gantry.json project config files were not removed."
    echo "You can manually delete them from your project directories if needed."
    echo ""
}

# Main uninstallation flow
main() {
    echo ""
    echo "=========================================="
    echo "  GANTRY Uninstaller"
    echo "=========================================="
    echo ""

    # Find gantry
    if find_gantry; then
        remove_binary
    else
        warn "Gantry binary not found in common locations"
    fi

    # Ask about config removal
    remove_config

    # Show completion
    show_completion
}

# Run main
main "$@"
