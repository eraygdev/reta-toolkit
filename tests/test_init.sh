#!/usr/bin/env bash
# Tests for `reta init`.

test_init_copies_templates() {
    "$RETA_BIN" init . >/dev/null
    assert_file_exists Makefile
    assert_file_exists .env.example
    assert_file_exists .gitignore
    assert_file_exists README.md
    assert_file_exists .retasign

    local sig
    sig=$(cat .retasign)
    python3 -c '
import re, sys
sig = sys.stdin.read().strip()
assert re.match(r"^RETA\*rid[0-9]+r$", sig), f"unexpected signature: {sig!r}"
' <<<"$sig"
}

test_init_does_not_overwrite() {
    echo "custom-content" > Makefile
    "$RETA_BIN" init . >/dev/null
    local content
    content=$(cat Makefile)
    assert_eq "custom-content" "$content" "Makefile must not be overwritten"
}

test_init_rejects_missing_target() {
    assert_exit_code 1 "$RETA_BIN" init /nonexistent/path
}

run_test "reta init copies all templates"    test_init_copies_templates
run_test "reta init does not overwrite"      test_init_does_not_overwrite
run_test "reta init rejects missing target"  test_init_rejects_missing_target