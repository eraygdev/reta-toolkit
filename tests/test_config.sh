#!/usr/bin/env bash
# Tests for `reta config` and hint toggling.

test_config_defaults_and_set_hints() {
    mkdir -p fakehome
    local out
    out=$(HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-config" hints)
    assert_eq "hints=on" "$out"

    HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-config" hints off >/dev/null
    out=$(HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-config" hints)
    assert_eq "hints=off" "$out"
}

test_config_reset_restores_defaults() {
    mkdir -p fakehome
    HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-config" hints off >/dev/null
    assert_file_exists fakehome/.config/reta/config

    HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-config" --reset >/dev/null
    assert_file_not_exists fakehome/.config/reta/config

    local out
    out=$(HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-config" hints)
    assert_eq "hints=on" "$out"
}

test_config_rejects_unknown_key_and_invalid_value() {
    mkdir -p fakehome
    assert_exit_code 1 env HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-config" unknown_key
    assert_exit_code 2 env HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-config" hints invalid
}

test_config_hints_off_suppresses_next_steps() {
    mkdir -p fakehome
    HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-config" hints off >/dev/null
    echo "APP_KEY=abc" > .env.example
    echo "APP_KEY=abc" > .env

    local out
    out=$(NO_COLOR=1 HOME="$PWD/fakehome" "$RETA_ROOT/libexec/reta-env" check)
    assert_not_contains "$out" "Next steps:"
}

run_test "reta config reads and updates hints"         test_config_defaults_and_set_hints
run_test "reta config --reset restores defaults"       test_config_reset_restores_defaults
run_test "reta config rejects unknown key/value"       test_config_rejects_unknown_key_and_invalid_value
run_test "reta config hints off suppresses Next steps" test_config_hints_off_suppresses_next_steps