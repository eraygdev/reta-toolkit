#!/usr/bin/env bash
# Tests for `reta config` and hint toggling.

test_config_defaults_and_set_hints() {
    mkdir -p fakehome
    local out
    out=$(HOME="$PWD/fakehome" "$RETA_BIN" config hints)
    assert_eq "hints=on" "$out"

    HOME="$PWD/fakehome" "$RETA_BIN" config hints off >/dev/null
    out=$(HOME="$PWD/fakehome" "$RETA_BIN" config hints)
    assert_eq "hints=off" "$out"
}

test_config_reset_restores_defaults() {
    mkdir -p fakehome
    HOME="$PWD/fakehome" "$RETA_BIN" config hints off >/dev/null
    assert_file_exists fakehome/.config/reta/config

    HOME="$PWD/fakehome" "$RETA_BIN" config --reset >/dev/null
    assert_file_not_exists fakehome/.config/reta/config

    local out
    out=$(HOME="$PWD/fakehome" "$RETA_BIN" config hints)
    assert_eq "hints=on" "$out"
}

test_config_rejects_unknown_key_and_invalid_value() {
    mkdir -p fakehome
    assert_exit_code 1 env HOME="$PWD/fakehome" "$RETA_BIN" config unknown_key
    assert_exit_code 2 env HOME="$PWD/fakehome" "$RETA_BIN" config hints invalid
}

test_config_hints_off_suppresses_next_steps() {
    mkdir -p fakehome
    HOME="$PWD/fakehome" "$RETA_BIN" config hints off >/dev/null
    echo "APP_KEY=abc" > .env.example
    echo "APP_KEY=abc" > .env

    local out
    out=$(NO_COLOR=1 HOME="$PWD/fakehome" "$RETA_BIN" env check)
    assert_not_contains "$out" "Next steps:"
}

test_config_learn_mode_toggle() {
    mkdir -p fakehome
    local out
    out=$(HOME="$PWD/fakehome" "$RETA_BIN" config learn)
    assert_eq "learn=off" "$out"

    HOME="$PWD/fakehome" "$RETA_BIN" config learn on >/dev/null
    out=$(HOME="$PWD/fakehome" "$RETA_BIN" config learn)
    assert_eq "learn=on" "$out"

    HOME="$PWD/fakehome" "$RETA_BIN" config --reset >/dev/null
    out=$(HOME="$PWD/fakehome" "$RETA_BIN" config learn)
    assert_eq "learn=off" "$out"
}

test_config_search_depth_setting() {
    mkdir -p fakehome
    local out
    out=$(NO_COLOR=1 HOME="$PWD/fakehome" "$RETA_BIN" config search-depth)
    assert_contains "$out" "search-depth=4"
    assert_contains "$out" "recommended: 4"

    HOME="$PWD/fakehome" "$RETA_BIN" config search-depth 6 >/dev/null
    out=$(NO_COLOR=1 HOME="$PWD/fakehome" "$RETA_BIN" config search-depth)
    assert_contains "$out" "search-depth=6"

    out=$(NO_COLOR=1 HOME="$PWD/fakehome" "$RETA_BIN" config)
    assert_contains "$out" "recommended: 4"

    assert_exit_code 2 env HOME="$PWD/fakehome" "$RETA_BIN" config search-depth 0
    assert_exit_code 2 env HOME="$PWD/fakehome" "$RETA_BIN" config search-depth abc
}

run_test "reta config reads and updates hints"         test_config_defaults_and_set_hints
run_test "reta config reads and updates learn mode"    test_config_learn_mode_toggle
run_test "reta config manages search-depth"            test_config_search_depth_setting
run_test "reta config --reset restores defaults"       test_config_reset_restores_defaults
run_test "reta config rejects unknown key/value"       test_config_rejects_unknown_key_and_invalid_value
run_test "reta config hints off suppresses Next steps" test_config_hints_off_suppresses_next_steps