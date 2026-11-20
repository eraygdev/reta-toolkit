# shellcheck shell=bash
# Shared UI constants (colors) for bash scripts.
# Source from any bash script:
#   source "$HOME/.reta/lib/ui.sh"
#
# Colors are disabled automatically when:
#   - stdout is not a TTY (piped/redirected)
#   - NO_COLOR environment variable is set

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    BOLD=$'\033[1m'
    DIM=$'\033[2m'
    CYAN=$'\033[36m'
    GREEN=$'\033[32m'
    YELLOW=$'\033[1;33m'
    MAGENTA=$'\033[1;35m'
    RED=$'\033[31m'
    RESET=$'\033[0m'
else
    BOLD='' DIM='' CYAN='' GREEN='' YELLOW='' MAGENTA='' RED='' RESET=''
fi

# Optional semantic helpers — consistent OK/WARN/ERR formatting
ui_ok()     { printf "  %sOK%s    %s\n"   "$GREEN"  "$RESET" "$*"; }
ui_warn()   { printf "  %sWARN%s  %s\n"   "$YELLOW" "$RESET" "$*"; }
ui_err()    { printf "  %sERR%s   %s\n"   "$RED"    "$RESET" "$*" >&2; }
ui_fail()   { printf "  %sFAIL%s  %s\n"   "$RED"    "$RESET" "$*"; }
ui_dim()    { printf "  %s%s%s\n"         "$DIM"    "$*"     "$RESET"; }

_reta_hints_enabled() {
    case "${RETA_HINTS:-}" in
        off|false|0|no) return 1 ;;
    esac
    local cfg="$HOME/.config/reta/config"
    if [ -f "$cfg" ]; then
        local val
        val="$(sed -n 's/^hints=//p' "$cfg" 2>/dev/null | tail -1)"
        [ "$val" = "off" ] && return 1
    fi
    return 0
}

ui_next_steps() {
    _reta_hints_enabled || return 0
    [ $# -eq 0 ] && return 0
    printf "\n%sNext steps:%s\n" "$BOLD" "$RESET"
    local step
    for step in "$@"; do
        printf "  → %s\n" "$step"
    done
}