#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob

DEV_KIT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="$HOME/.local/bin"

echo "dev-kit installer"
echo "Source: $DEV_KIT_DIR"
echo ""

mkdir -p "$TARGET"

# 1. ~/.dev-kit symlink
if [ -L "$HOME/.dev-kit" ]; then
    rm -f "$HOME/.dev-kit"
elif [ -e "$HOME/.dev-kit" ]; then
    echo "ERROR: ~/.dev-kit exists but is not a symlink."
    echo "       Move or remove it manually, then re-run."
    exit 1
fi
ln -sfn "$DEV_KIT_DIR" "$HOME/.dev-kit"
echo "OK  ~/.dev-kit -> $DEV_KIT_DIR"

# 2. Kırık symlink temizliği (sadece dev-kit'e işaret edenler)
broken=0
for link in "$TARGET"/*; do
    [ -L "$link" ] || continue
    t="$(readlink "$link" 2>/dev/null || true)"
    case "$t" in
        "$HOME/.dev-kit"/*|"$DEV_KIT_DIR"/*)
            if [ ! -e "$t" ]; then
                rm -f "$link"
                echo "OK  cleaned broken: $(basename "$link")"
                broken=$((broken+1))
            fi
            ;;
    esac
done

# 3. Executable bit'leri ayarla
chmod +x "$DEV_KIT_DIR/bin"/* 2>/dev/null || true
chmod +x "$DEV_KIT_DIR/libexec"/* 2>/dev/null || true
chmod +x "$DEV_KIT_DIR/lib/export-context" 2>/dev/null || true

# 4. Sadece 'devkit' PATH'e girer
[ -L "$TARGET/devkit" ] && rm -f "$TARGET/devkit"
ln -sf "$DEV_KIT_DIR/bin/devkit" "$TARGET/devkit"
echo "OK  devkit -> $DEV_KIT_DIR/bin/devkit"

# 5. PATH kontrolü
case ":$PATH:" in
    *":$TARGET:"*)
        echo "OK  $TARGET is on PATH"
        ;;
    *)
        echo ""
        echo "WARN: $TARGET is not on PATH. Add to ~/.zshrc or ~/.bashrc:"
        echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo "Then: source ~/.zshrc"
        ;;
esac

echo ""
echo "Done. Test: devkit help"