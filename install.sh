#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob

RETA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="$HOME/.local/bin"

echo "reta installer"
echo "Source: $RETA_DIR"
echo ""

mkdir -p "$TARGET"

# 1. ~/.reta symlink (skipped if the repo already lives there)
if [ "$RETA_DIR" = "$HOME/.reta" ]; then
    echo "OK  already installed at ~/.reta"
else
    if [ -L "$HOME/.reta" ]; then
        rm -f "$HOME/.reta"
    elif [ -e "$HOME/.reta" ]; then
        echo "ERROR: ~/.reta exists but is not a symlink."
        echo "       Move or remove it manually, then re-run."
        exit 1
    fi
    ln -sfn "$RETA_DIR" "$HOME/.reta"
    echo "OK  ~/.reta -> $RETA_DIR"
fi

# 2. Broken symlink cleanup (only links pointing into ~/.reta)
broken=0
for link in "$TARGET"/*; do
    [ -L "$link" ] || continue
    t="$(readlink "$link" 2>/dev/null || true)"
    case "$t" in
        "$HOME/.reta"/*|"$RETA_DIR"/*)
            if [ ! -e "$t" ]; then
                rm -f "$link"
                echo "OK  cleaned broken: $(basename "$link")"
                broken=$((broken+1))
            fi
            ;;
    esac
done

# 3. Set executable bits
chmod +x "$RETA_DIR/bin"/* 2>/dev/null || true
chmod +x "$RETA_DIR/libexec"/* 2>/dev/null || true
chmod +x "$RETA_DIR/lib/export-context" 2>/dev/null || true

# 4. Only 'reta' goes onto PATH
[ -L "$TARGET/reta" ] && rm -f "$TARGET/reta"
ln -sf "$RETA_DIR/bin/reta" "$TARGET/reta"
echo "OK  reta -> $RETA_DIR/bin/reta"

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
echo "Done. Test: reta help"