#!/usr/bin/env bash
# Tests for `lib/check-secrets`.

test_check_secrets_clean_repo_exits_0() {
    echo ".env" > .gitignore
    echo "print('hello')" > app.py

    assert_exit_code 0 "$RETA_ROOT/lib/check-secrets"
}

test_check_secrets_detects_leaked_key_exits_1() {
    echo ".env" > .gitignore
    local p1="sk-proj-"
    local p2="abcdefghijklmnopqrstuvwxyz1234"
    echo "API_KEY = \"${p1}${p2}\"" > app.py

    local out
    set +e
    out=$(NO_COLOR=1 "$RETA_ROOT/lib/check-secrets" 2>&1)
    local rc=$?
    set -e

    assert_eq "1" "$rc" "expected exit 1 on leaked key"
    assert_contains "$out" "app.py:1"
    assert_contains "$out" "OpenAI API key"
}

test_check_secrets_respects_ignore_marker() {
    echo ".env" > .gitignore
    local p1="AKIA"
    local p2="IOSFODNN7EXAMPLE"
    echo "KEY = \"${p1}${p2}\"  # check-secrets: ignore" > app.py

    assert_exit_code 0 "$RETA_ROOT/lib/check-secrets"
}

test_check_secrets_skips_env_example() {
    echo ".env" > .gitignore
    local p1="AIza"
    local p2="SyDummyKeyForExampleTemplate1234567890"
    echo "GEMINI_API_KEY=${p1}${p2}" > .env.example

    assert_exit_code 0 "$RETA_ROOT/lib/check-secrets"
}

test_check_secrets_warns_when_env_missing_from_gitignore() {
    echo "node_modules/" > .gitignore
    echo "print('ok')" > app.py

    local out
    out=$(NO_COLOR=1 "$RETA_ROOT/lib/check-secrets" 2>&1)
    assert_contains "$out" "WARN"
    assert_contains "$out" ".env is not listed in .gitignore"
}

test_check_secrets_fails_when_env_tracked_by_git() {
    command -v git >/dev/null 2>&1 || return 0

    git init -q
    echo ".env" > .gitignore
    echo "SECRET=123" > .env
    git add -f .env

    local out
    set +e
    out=$(NO_COLOR=1 "$RETA_ROOT/lib/check-secrets" 2>&1)
    local rc=$?
    set -e

    assert_eq "1" "$rc" "expected exit 1 when .env is tracked in git"
    assert_contains "$out" "Sensitive file(s) tracked by git"
}

run_test "check-secrets passes on clean project"        test_check_secrets_clean_repo_exits_0
run_test "check-secrets catches hardcoded secret"      test_check_secrets_detects_leaked_key_exits_1
run_test "check-secrets honors inline ignore marker"   test_check_secrets_respects_ignore_marker
run_test "check-secrets ignores .env.example"          test_check_secrets_skips_env_example
run_test "check-secrets warns on missing .gitignore"   test_check_secrets_warns_when_env_missing_from_gitignore
run_test "check-secrets fails on tracked .env in git"  test_check_secrets_fails_when_env_tracked_by_git