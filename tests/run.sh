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

echo "reta test suite"
echo "Root: $RETA_ROOT"
echo "Building Go binary..."
(cd "$RETA_ROOT" && go build -o "$RETA_BIN" ./cmd/reta) || exit 1
echo ""

echo "== go unit tests =="
if ! (cd "$RETA_ROOT" && go test ./...); then
    echo "FAIL: go test ./..."
    exit 1
fi
echo ""

for file in "$TESTS_DIR"/test_*.sh; do
    [ -f "$file" ] || continue
    echo "== $(basename "$file" .sh) =="
    # shellcheck source=/dev/null
    source "$file"
    echo ""
done

report