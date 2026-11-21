#!/usr/bin/env bash
# Test runner. Sources every test_*.sh and executes registered tests.

set -uo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RETA_ROOT="$(cd "$TESTS_DIR/.." && pwd)"
export RETA_ROOT

# shellcheck source=lib.sh
source "$TESTS_DIR/lib.sh"

echo "reta test suite"
echo "Root: $RETA_ROOT"
echo ""

for file in "$TESTS_DIR"/test_*.sh; do
    [ -f "$file" ] || continue
    echo "== $(basename "$file" .sh) =="
    # shellcheck source=/dev/null
    source "$file"
    echo ""
done

report