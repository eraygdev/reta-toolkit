#!/usr/bin/env sh
set -eu

REPO="eraygdev/reta-toolkit"
INSTALL_DIR="${HOME}/.local/bin"
BIN_PATH="${INSTALL_DIR}/reta"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
    linux|darwin) ;;
    *)
        echo "Unsupported OS: $OS" >&2
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

ASSET="reta-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

mkdir -p "$INSTALL_DIR"
echo "Downloading ${ASSET}..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$URL" -o "$BIN_PATH"
elif command -v wget >/dev/null 2>&1; then
    wget -qO "$BIN_PATH" "$URL"
else
    echo "Error: curl or wget is required." >&2
    exit 1
fi
chmod +x "$BIN_PATH"

"$BIN_PATH" doctor --fix >/dev/null 2>&1 || true

BLOCK_START="# ===RETA*==="
BLOCK_END="# ==========="
for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
    [ -f "$rc" ] || continue
    if ! grep -Fqsx "$BLOCK_START" "$rc" 2>/dev/null; then
        printf '\n%s\ncommand -v reta >/dev/null 2>&1 && eval "$(reta shell-init)"\n%s\n' "$BLOCK_START" "$BLOCK_END" >> "$rc"
        echo "Added rcd integration to $rc"
    fi
done

echo "Installed reta to $BIN_PATH"
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "Add to PATH: export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
esac