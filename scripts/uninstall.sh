#!/bin/sh
set -eu

# specd uninstaller — Stackific Inc. All rights reserved.
# https://stackific.com/specd
# Usage: curl -sSL https://stackific.com/specd/uninstall.sh | sh

PRODUCT="specd"
INSTALL_DIR="$HOME/.specd"
BIN_DIR="$INSTALL_DIR/bin"

main() {
    if [ ! -d "$BIN_DIR" ]; then
        echo "${PRODUCT} is not installed (${BIN_DIR} not found)."
        exit 0
    fi

    echo "Removing ${BIN_DIR}..."
    rm -rf "$BIN_DIR"
    echo "Removed ${BIN_DIR}"

    echo ""
    echo "Note: ${INSTALL_DIR}/ (config, cache, skills) was kept."
    echo "To remove everything: rm -rf ${INSTALL_DIR}"
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
