#!/usr/bin/env bash
set -euo pipefail

RETA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="$HOME/.local/bin"

echo "reta installer (Go)"
echo "Source: $RETA_DIR"
echo ""

if ! command -v go >/dev/null 2>&1; then
    echo "Error: 'go' is required to build from source." >&2
    echo "Or install prebuilt binary via: curl -fsSL https://raw.githubusercontent.com/eraygdev/reta-toolkit/main/scripts/install.sh | sh" >&2
    exit 1
fi

mkdir -p "$TARGET"

if [ "$RETA_DIR" = "$HOME/.reta" ]; then
    echo "  OK    already at ~/.reta"
else
    if [ -L "$HOME/.reta" ]; then
        rm -f "$HOME/.reta"
    elif [ -e "$HOME/.reta" ]; then
        echo "  ERR   ~/.reta exists but is not a symlink." >&2
        exit 1
    fi
    ln -sfn "$RETA_DIR" "$HOME/.reta"
    echo "  OK    ~/.reta -> $RETA_DIR"
fi

echo "Building Go binary..."
rm -f "$TARGET/reta"
(cd "$RETA_DIR" && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$TARGET/reta" ./cmd/reta)
chmod +x "$TARGET/reta"
echo "  OK    installed $TARGET/reta"

case ":$PATH:" in
    *":$TARGET:"*)
        echo "  OK    $TARGET is on PATH"
        ;;
    *)
        echo ""
        echo "  WARN  $TARGET is not on PATH. Add to ~/.zshrc or ~/.bashrc:"
        echo "        export PATH=\"\$HOME/.local/bin:\$PATH\""
        ;;
esac

echo ""
echo "Shell completion:"
BASH_COMPLETION_DIR="$HOME/.local/share/bash-completion/completions"
mkdir -p "$BASH_COMPLETION_DIR"
rm -f "$BASH_COMPLETION_DIR/reta"
"$TARGET/reta" completion bash > "$BASH_COMPLETION_DIR/reta"
echo "  OK    bash: $BASH_COMPLETION_DIR/reta"

if command -v zsh >/dev/null 2>&1; then
    ZSH_COMPLETION_DIR="$HOME/.zfunc"
    mkdir -p "$ZSH_COMPLETION_DIR"
    rm -f "$ZSH_COMPLETION_DIR/_reta"
    "$TARGET/reta" completion zsh > "$ZSH_COMPLETION_DIR/_reta"
    echo "  OK    zsh:  $ZSH_COMPLETION_DIR/_reta"
    if ! grep -qs "fpath.*\.zfunc" "$HOME/.zshrc" 2>/dev/null; then
        echo "  WARN  zsh: add 'fpath+=~/.zfunc' and 'autoload -Uz compinit && compinit' to ~/.zshrc"
    fi
fi

RCD_START="# ===RETA*==="
RCD_END="# ==========="
RCD_BLOCK='# ===RETA*===
command -v reta >/dev/null 2>&1 && eval "$(reta shell-init)"
# ==========='

for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
    [ -f "$rc" ] || continue
    if grep -Fqsx "$RCD_START" "$rc" 2>/dev/null; then
        tmp="$(mktemp)"
        awk -v start="$RCD_START" -v end="$RCD_END" -v block="$RCD_BLOCK" '
            $0 == start { print block; skip = 1; next }
            skip && $0 == end { skip = 0; next }
            !skip { print }
        ' "$rc" > "$tmp"
        if ! cmp -s "$tmp" "$rc"; then
            cat "$tmp" > "$rc"
            echo "  OK    updated rcd block in $rc"
        else
            echo "  OK    rcd block up to date in $rc"
        fi
        rm -f "$tmp"
    else
        printf '\n%s\n' "$RCD_BLOCK" >> "$rc"
        echo "  OK    added rcd block to $rc"
    fi
done

echo ""
echo "  OK    reta installed"
"$TARGET/reta" doctor || true