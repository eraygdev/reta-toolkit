#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob

RETA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="$HOME/.local/bin"

# shellcheck source=lib/ui.sh
source "$RETA_DIR/lib/ui.sh"

echo "reta installer"
echo "Source: $RETA_DIR"
echo ""

mkdir -p "$TARGET"

# 1. ~/.reta symlink (skipped if the repo already lives there)
if [ "$RETA_DIR" = "$HOME/.reta" ]; then
    ui_ok "already installed at ~/.reta"
else
    if [ -L "$HOME/.reta" ]; then
        rm -f "$HOME/.reta"
    elif [ -e "$HOME/.reta" ]; then
        ui_err "~/.reta exists but is not a symlink."
        echo "       Move or remove it manually, then re-run." >&2
        exit 1
    fi
    ln -sfn "$RETA_DIR" "$HOME/.reta"
    ui_ok "~/.reta -> $RETA_DIR"
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
                ui_ok "cleaned broken: $(basename "$link")"
                broken=$((broken+1))
            fi
            ;;
    esac
done

# 3. Set executable bits
chmod +x "$RETA_DIR/bin"/* 2>/dev/null || true
chmod +x "$RETA_DIR/libexec"/* 2>/dev/null || true
chmod +x "$RETA_DIR/lib/export-context" 2>/dev/null || true
chmod +x "$RETA_DIR/lib/check-secrets" 2>/dev/null || true
chmod +x "$RETA_DIR/lib/adopt-makefile" 2>/dev/null || true
chmod +x "$RETA_DIR/tests/run.sh" 2>/dev/null || true

# 4. Only 'reta' goes onto PATH
[ -L "$TARGET/reta" ] && rm -f "$TARGET/reta"
ln -sf "$RETA_DIR/bin/reta" "$TARGET/reta"
ui_ok "reta -> $RETA_DIR/bin/reta"

# 5. PATH check
case ":$PATH:" in
    *":$TARGET:"*)
        ui_ok "$TARGET is on PATH"
        ;;
    *)
        echo ""
        ui_warn "$TARGET is not on PATH. Add to ~/.zshrc or ~/.bashrc:"
        echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo "Then: source ~/.zshrc"
        ;;
esac

# 6. Shell completion (best-effort, never aborts install)
echo ""
echo "Shell completion:"
BASH_COMPLETION_DIR="$HOME/.local/share/bash-completion/completions"
ZSH_COMPLETION_DIR="$HOME/.zfunc"

if [ -f "$RETA_DIR/lib/completion.bash" ]; then
    mkdir -p "$BASH_COMPLETION_DIR"
    ln -sfn "$RETA_DIR/lib/completion.bash" "$BASH_COMPLETION_DIR/reta"
    ui_ok "bash: $BASH_COMPLETION_DIR/reta"
fi

if [ -f "$RETA_DIR/lib/completion.zsh" ] && command -v zsh >/dev/null 2>&1; then
    mkdir -p "$ZSH_COMPLETION_DIR"
    ln -sfn "$RETA_DIR/lib/completion.zsh" "$ZSH_COMPLETION_DIR/_reta"
    ui_ok "zsh:  $ZSH_COMPLETION_DIR/_reta"
    if ! grep -qs "fpath.*\.zfunc" "$HOME/.zshrc" 2>/dev/null; then
        ui_warn "zsh: add to ~/.zshrc:"
        echo "      fpath+=~/.zfunc"
        echo "      autoload -Uz compinit && compinit"
    fi
fi

echo ""
ui_ok "reta installed"
ui_next_steps \
    "reta doctor       Verify installation health" \
    "reta help         List all available commands" \
    "reta init         Scaffold templates into a new project" \
    "reta init-env     Generate .env from .env.example"