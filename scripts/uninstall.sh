#!/bin/sh
set -eu

# specd uninstaller — Stackific Inc. All rights reserved.
# https://stackific.com/specd
# Usage: curl -sSL https://stackific.com/specd/uninstall.sh | sh

COMPANY="Stackific Inc."
PRODUCT="specd"
HOMEPAGE="https://stackific.com/specd"
INSTALL_DIR="$HOME/.specd"
BIN_DIR="$INSTALL_DIR/bin"
BINARY="specd"

main() {
    if [ ! -d "$INSTALL_DIR" ]; then
        echo "${PRODUCT} is not installed (${INSTALL_DIR} not found)."
        exit 0
    fi

    echo "Removing ${INSTALL_DIR}..."
    rm -rf "$INSTALL_DIR"
    echo "Removed ${INSTALL_DIR}"

    echo ""
    clean_path
    echo ""
    echo "${PRODUCT} has been uninstalled."
}

clean_path() {
    SHELL_NAME="$(basename "${SHELL:-/bin/sh}")"

    echo "Remove the specd PATH entry from your shell config:"
    echo ""

    case "$SHELL_NAME" in
        bash)
            echo "  Edit ~/.bashrc and remove the line:"
            echo "    export PATH=\"\$HOME/.specd/bin:\$PATH\""
            echo ""
            echo "  Then run: source ~/.bashrc"
            ;;
        zsh)
            echo "  Edit ~/.zshrc and remove the line:"
            echo "    export PATH=\"\$HOME/.specd/bin:\$PATH\""
            echo ""
            echo "  Then run: source ~/.zshrc"
            ;;
        fish)
            echo "  Run: set -e fish_user_paths[(contains -i $BIN_DIR \$fish_user_paths)]"
            echo ""
            echo "  Or edit ~/.config/fish/config.fish and remove the specd PATH line."
            echo "  Then run: exec fish"
            ;;
        *)
            echo "  Remove this line from your shell config:"
            echo "    export PATH=\"\$HOME/.specd/bin:\$PATH\""
            echo ""
            echo "  Then reload your terminal."
            ;;
    esac
}

main
