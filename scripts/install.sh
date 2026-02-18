#!/bin/bash
set -euo pipefail

# headline-goat installer
# Usage: curl -sSL https://raw.githubusercontent.com/gkobilansky/headline-goat/main/scripts/install.sh | bash

REPO="gkobilansky/headline-goat"
INSTALL_DIR="${HLG_INSTALL_DIR:-/usr/local/bin}"

detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux) echo "linux" ;;
        darwin) echo "darwin" ;;
        *) echo "unsupported" ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unsupported" ;;
    esac
}

get_binary_name() {
    local os="${1:-$(detect_os)}"
    local arch="${2:-$(detect_arch)}"
    echo "hlg-${os}-${arch}"
}

get_latest_version() {
    curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" |
        grep '"tag_name":' |
        sed -E 's/.*"([^"]+)".*/\1/'
}

download_binary() {
    local version="$1"
    local binary_name="$2"
    local dest="$3"

    local url="https://github.com/${REPO}/releases/download/${version}/${binary_name}"

    echo "Downloading ${binary_name} ${version}..."

    if command -v curl &>/dev/null; then
        curl -sSL -o "$dest" "$url"
    elif command -v wget &>/dev/null; then
        wget -q -O "$dest" "$url"
    else
        echo "Error: curl or wget required"
        return 1
    fi
}

install_binary() {
    local src="$1"
    local dest_dir="$2"
    local binary_name="hlg"

    chmod +x "$src"

    # Try to install to dest_dir, fall back to ~/.local/bin
    if [ -w "$dest_dir" ]; then
        mv "$src" "${dest_dir}/${binary_name}"
        echo "Installed to ${dest_dir}/${binary_name}"
    elif [ -w "$HOME/.local/bin" ]; then
        mkdir -p "$HOME/.local/bin"
        mv "$src" "$HOME/.local/bin/${binary_name}"
        echo "Installed to $HOME/.local/bin/${binary_name}"
        echo "Make sure $HOME/.local/bin is in your PATH"
    else
        echo "Installing to ${dest_dir} (requires sudo)..."
        sudo mv "$src" "${dest_dir}/${binary_name}"
        echo "Installed to ${dest_dir}/${binary_name}"
    fi
}

main() {
    echo "headline-goat installer"
    echo "======================"
    echo ""

    local os arch binary_name version tmp_file

    os="$(detect_os)"
    arch="$(detect_arch)"

    if [ "$os" = "unsupported" ]; then
        echo "Error: Unsupported operating system: $(uname -s)"
        exit 1
    fi

    if [ "$arch" = "unsupported" ]; then
        echo "Error: Unsupported architecture: $(uname -m)"
        exit 1
    fi

    echo "Detected: ${os}/${arch}"

    binary_name="$(get_binary_name "$os" "$arch")"

    version="$(get_latest_version)"
    if [ -z "$version" ]; then
        echo "Error: Could not determine latest version"
        exit 1
    fi

    echo "Latest version: ${version}"

    tmp_file="$(mktemp)"
    trap 'rm -f "$tmp_file"' EXIT

    download_binary "$version" "$binary_name" "$tmp_file"
    install_binary "$tmp_file" "$INSTALL_DIR"

    echo ""
    echo "Installation complete! Run 'hlg' to get started."
}

# Allow sourcing for testing
if [ "${1:-}" = "--source-only" ]; then
    return 0 2>/dev/null || exit 0
fi

main "$@"
