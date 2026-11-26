#!/usr/bin/env bash
# Test runner. Sources every test_*.sh and executes registered tests.

set -uo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RETA_ROOT="$(cd "$TESTS_DIR/.." && pwd)"
export RETA_ROOT

# shellcheck source=lib.sh
source "$TESTS_DIR/lib.sh"

TEST_BIN_DIR="$(mktemp -d)"
cleanup_test_bin() {
    rm -rf "$TEST_BIN_DIR"
}
trap cleanup_test_bin EXIT

RETA_BIN="$TEST_BIN_DIR/reta"
export RETA_BIN
export PATH="$TEST_BIN_DIR:$PATH"

VERBOSE=0
for arg in "$@"; do
    case "$arg" in
        -v|--verbose) VERBOSE=1 ;;
    esac
done

START_SEC=$SECONDS
START_MS=$(python3 -c 'import time; print(int(time.monotonic()*1000))' 2>/dev/null || echo "")

colorize_go_test() {
    if [ -z "$C_RESET" ]; then
        cat
        return
    fi
    awk -v g="$C_GREEN" -v r="$C_RED" -v y="$C_YELLOW" -v c="$C_CYAN" -v d="$C_DIM" -v b="$C_BOLD" -v rst="$C_RESET" '
        /^[[:space:]]*=== (RUN|PAUSE|CONT)/ { printf "%s%s%s\n", c, $0, rst; fflush(); next }
        /^[[:space:]]*--- PASS:/ { printf "%s%s%s\n", g, $0, rst; fflush(); next }
        /^[[:space:]]*--- FAIL:/ { printf "%s%s%s\n", r, $0, rst; fflush(); next }
        /^[[:space:]]*--- SKIP:/ { printf "%s%s%s\n", y, $0, rst; fflush(); next }
        /^PASS$/ { printf "%s%s%s%s\n", b, g, $0, rst; fflush(); next }
        /^FAIL$/ || /^FAIL\t/ { printf "%s%s%s%s\n", b, r, $0, rst; fflush(); next }
        /^ok[[:space:]]+/ { sub(/^ok/, g "ok" rst); print; fflush(); next }
        /^\?[[:space:]]+/ { printf "%s%s%s\n", d, $0, rst; fflush(); next }
        /_test\.go:[0-9]+:/ { printf "%s%s%s\n", r, $0, rst; fflush(); next }
        { printf "%s%s%s\n", d, $0, rst; fflush() }
    '
}

printf "%sreta test suite%s\n" "$C_BOLD" "$C_RESET"
printf "%sRoot:%s %s\n" "$C_DIM" "$C_RESET" "$RETA_ROOT"
echo "Building Go binary..."
(cd "$RETA_ROOT" && go build -o "$RETA_BIN" ./cmd/reta) || exit 1
echo ""

printf "%s== go unit tests ==%s\n" "$C_YELLOW" "$C_RESET"
if [ "$VERBOSE" -eq 1 ]; then
    if ! (cd "$RETA_ROOT" && go test -v ./... 2>&1) | colorize_go_test; then
        printf "%sFAIL: go test -v ./...%s\n" "$C_RED" "$C_RESET"
        exit 1
    fi
else
    if ! (cd "$RETA_ROOT" && go test ./... 2>&1) | colorize_go_test; then
        printf "%sFAIL: go test ./...%s\n" "$C_RED" "$C_RESET"
        exit 1
    fi
fi
echo ""

for file in "$TESTS_DIR"/test_*.sh; do
    [ -f "$file" ] || continue
    printf "%s== %s ==%s\n" "$C_YELLOW" "$(basename "$file" .sh)" "$C_RESET"
    # shellcheck source=/dev/null
    source "$file"
    echo ""
done

rc=0
report || rc=$?
END_MS=""
if [ -n "$START_MS" ]; then
    END_MS=$(python3 -c 'import time; print(int(time.monotonic()*1000))' 2>/dev/null || echo "")
fi
if [ -n "$START_MS" ] && [ -n "$END_MS" ]; then
    ELAPSED=$(awk -v s="$START_MS" -v e="$END_MS" 'BEGIN { printf "%.2fs", (e - s) / 1000 }')
    printf "%sDuration:%s %s\n" "$C_DIM" "$C_RESET" "$ELAPSED"
else
    printf "%sDuration:%s %ds\n" "$C_DIM" "$C_RESET" "$((SECONDS - START_SEC))"
fi
exit "$rc"