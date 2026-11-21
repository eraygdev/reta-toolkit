#!/usr/bin/env bash
# Tests for reta dispatcher (help, version).

test_help_lists_all_subcommands() {
    local out
    out=$("$RETA_ROOT/bin/reta" help 2>&1)
    assert_contains "$out" "doctor"
    assert_contains "$out" "init"
    assert_contains "$out" "port"
    assert_contains "$out" "env"
}

test_help_subcommand_delegates() {
    local out
    out=$("$RETA_ROOT/bin/reta" help port 2>&1)
    assert_contains "$out" "list"
    assert_contains "$out" "kill"
}

test_help_unknown_subcommand() {
    assert_exit_code 1 "$RETA_ROOT/bin/reta" help nonexistent
}

test_version_flag() {
    local out expected
    out=$("$RETA_ROOT/bin/reta" --version 2>&1)
    expected=$(cat "$RETA_ROOT/VERSION")
    assert_eq "$expected" "$out"
}

test_help_respects_setup_order() {
    local out
    out=$(NO_COLOR=1 "$RETA_ROOT/bin/reta" help 2>&1)
    python3 -c '
import sys
out = sys.stdin.read()
i_init = out.index("  init ")
i_ienv = out.index("  init-env ")
i_upd = out.index("  update ")
i_uninst = out.index("  uninstall ")
assert i_init < i_ienv < i_upd < i_uninst, "unexpected Setup command order"
' <<<"$out"
}

run_test "reta help lists subcommands"   test_help_lists_all_subcommands
run_test "reta help respects order"      test_help_respects_setup_order
run_test "reta help <cmd> delegates"     test_help_subcommand_delegates
run_test "reta help rejects unknown"     test_help_unknown_subcommand
run_test "reta --version prints VERSION" test_version_flag