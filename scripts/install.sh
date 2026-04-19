#!/bin/sh
set -eu

# specd installer
# Usage: curl -sSL https://stackific.com/specd/install.sh | sh

REPO="stackific/specd"
INSTALL_DIR="$HOME/.specd/bin"
BINARY="specd"

main() {
    detect_platform
    get_latest_version
    download_binary
    install_binary
    setup_path
    echo ""
    echo "specd ${VERSION} installed successfully!"
    echo ""
}

detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      echo "Error: unsupported OS: $OS"; exit 1 ;;
    esac

    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)             echo "Error: unsupported architecture: $ARCH"; exit 1 ;;
    esac

    echo "Detected platform: ${OS}/${ARCH}"
}

get_latest_version() {
    if command -v curl > /dev/null 2>&1; then
        VERSION="$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')"
    elif command -v wget > /dev/null 2>&1; then
        VERSION="$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')"
    else
        echo "Error: curl or wget is required"
        exit 1
    fi

    if [ -z "$VERSION" ]; then
        echo "Error: could not determine latest version"
        exit 1
    fi

    echo "Latest version: ${VERSION}"
}

download_binary() {
    FILENAME="${BINARY}-${OS}-${ARCH}"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

    echo "Downloading ${URL}..."

    TMPDIR="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR"' EXIT

    if command -v curl > /dev/null 2>&1; then
        curl -sSL -o "${TMPDIR}/${BINARY}" "$URL"
    else
        wget -qO "${TMPDIR}/${BINARY}" "$URL"
    fi
}

install_binary() {
    mkdir -p "$INSTALL_DIR"
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    chmod +x "${INSTALL_DIR}/${BINARY}"
    echo "Installed to ${INSTALL_DIR}/${BINARY}"
}

setup_path() {
    EXPORT_LINE="export PATH=\"\$HOME/.specd/bin:\$PATH\""

    # Check if already in PATH
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) return ;;
    esac

    echo ""
    echo "Add specd to your PATH by adding this line to your shell config:"
    echo ""

    SHELL_NAME="$(basename "${SHELL:-/bin/sh}")"
    case "$SHELL_NAME" in
        bash)
            RC="$HOME/.bashrc"
            echo "  echo '${EXPORT_LINE}' >> ${RC}"
            ;;
        zsh)
            RC="$HOME/.zshrc"
            echo "  echo '${EXPORT_LINE}' >> ${RC}"
            ;;
        fish)
            echo "  fish_add_path $INSTALL_DIR"
            echo ""
            echo "Or add to ~/.config/fish/config.fish:"
            echo "  set -gx PATH \$HOME/.specd/bin \$PATH"
            ;;
        *)
            echo "  ${EXPORT_LINE}"
            echo ""
            echo "Add the above to your shell's config file."
            ;;
    esac

    echo ""
    echo "Then reload your terminal or run:"
    case "$SHELL_NAME" in
        bash) echo "  source ${RC}" ;;
        zsh)  echo "  source ${RC}" ;;
        fish) echo "  exec fish" ;;
        *)    echo "  source your shell config file" ;;
    esac
}

main
