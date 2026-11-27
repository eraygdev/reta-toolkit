#!/usr/bin/env bash
# Tests for `reta init`.

test_init_copies_templates() {
    mkdir -p fakehome
    HOME="$PWD/fakehome" "$RETA_BIN" init . >/dev/null
    assert_file_exists Makefile
    assert_file_exists .env.example
    assert_file_exists .gitignore
    assert_file_exists README.md
    assert_file_exists .retasign
    assert_file_exists AGENTS.md

    local sig
    sig=$(cat .retasign)
    python3 -c '
import re, sys
sig = sys.stdin.read().strip()
assert re.match(r"^RETA\*rid[0-9]+r$", sig), f"unexpected signature: {sig!r}"
' <<<"$sig"
}

test_init_agents_file_variants() {
    mkdir -p fakehome proj_tpl proj_empty proj_off
    printf "Custom agent prompt\n" > prompt.md

    HOME="$PWD/fakehome" "$RETA_BIN" config agents-file CLAUDE.md >/dev/null
    HOME="$PWD/fakehome" "$RETA_BIN" config agents-template prompt.md >/dev/null
    HOME="$PWD/fakehome" "$RETA_BIN" init proj_tpl >/dev/null
    assert_file_exists proj_tpl/CLAUDE.md
    assert_file_not_exists proj_tpl/AGENTS.md
    assert_eq "Custom agent prompt" "$(cat proj_tpl/CLAUDE.md)"

    HOME="$PWD/fakehome" "$RETA_BIN" config agents-template "" >/dev/null
    HOME="$PWD/fakehome" "$RETA_BIN" init proj_empty >/dev/null
    assert_file_exists proj_empty/CLAUDE.md
    assert_eq "" "$(cat proj_empty/CLAUDE.md)"

    HOME="$PWD/fakehome" "$RETA_BIN" config agents-file "" >/dev/null
    HOME="$PWD/fakehome" "$RETA_BIN" init proj_off >/dev/null
    assert_file_not_exists proj_off/CLAUDE.md
    assert_file_not_exists proj_off/AGENTS.md
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
run_test "reta init handles agents config"   test_init_agents_file_variants
run_test "reta init does not overwrite"      test_init_does_not_overwrite
run_test "reta init rejects missing target"  test_init_rejects_missing_target