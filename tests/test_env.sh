#!/usr/bin/env bash
# Tests for `reta env`.

test_env_check_missing_key_exits_1() {
    cat > .env.example <<'EOF'
APP_KEY=abc
MISSING_KEY=xyz
EOF
    echo "APP_KEY=abc" > .env

    assert_exit_code 1 "$RETA_BIN" env check
}

test_env_check_in_sync_exits_0() {
    echo "APP_KEY=abc" > .env.example
    echo "APP_KEY=abc" > .env

    assert_exit_code 0 "$RETA_BIN" env check
}

test_env_check_missing_files_exits_1() {
    assert_exit_code 1 "$RETA_BIN" env check
}

test_env_diff_detects_extra_keys() {
    cat > .env.example <<'EOF'
APP_KEY=abc
EOF
    cat > .env <<'EOF'
APP_KEY=abc
EXTRA=value
EOF

    local out
    out=$("$RETA_BIN" env diff 2>&1) || true
    assert_contains "$out" "EXTRA"
}

test_env_sync_appends_missing_and_preserves_existing() {
    cat > .env.example <<'EOF'
APP_KEY=GENERATE_SECRET
REDIS_URL=redis://localhost:6379/0
JWT_SECRET=GENERATE_SECRET
EOF
    echo "APP_KEY=keep_existing_value" > .env

    NO_COLOR=1 "$RETA_BIN" env sync >/dev/null
    local content
    content=$(cat .env)
    assert_contains "$content" "APP_KEY=keep_existing_value"
    assert_contains "$content" "REDIS_URL=redis://localhost:6379/0"
    assert_not_contains "$content" "GENERATE_SECRET"

    local jwt
    jwt=$(grep '^JWT_SECRET=' .env | cut -d= -f2)
    assert_eq "64" "${#jwt}" "JWT_SECRET should be 64 hex chars"
}

test_env_sync_dry_run_does_not_modify() {
    cat > .env.example <<'EOF'
APP_KEY=abc
NEW_KEY=xyz
EOF
    echo "APP_KEY=abc" > .env

    local out
    out=$(NO_COLOR=1 "$RETA_BIN" env sync --dry-run)
    assert_contains "$out" "NEW_KEY=xyz"
    local content
    content=$(cat .env)
    assert_not_contains "$content" "NEW_KEY"
}

test_env_sync_handles_inline_comments_and_quotes() {
    cat > .env.example <<'EOF'
APP_KEY=abc
EMPTY_WITH_COMMENT= # fill later
QUOTED_VAL="hello world"
EOF
    echo "APP_KEY=abc" > .env

    NO_COLOR=1 "$RETA_BIN" env sync </dev/null >/dev/null
    local content
    content=$(cat .env)
    assert_contains "$content" 'QUOTED_VAL="hello world"'
    assert_contains "$content" "EMPTY_WITH_COMMENT="
    assert_not_contains "$content" "# fill later"
}

run_test "reta env check exits 1 when missing"      test_env_check_missing_key_exits_1
run_test "reta env check exits 0 when in sync"      test_env_check_in_sync_exits_0
run_test "reta env check exits 1 when no files"     test_env_check_missing_files_exits_1
run_test "reta env diff detects extra keys"         test_env_diff_detects_extra_keys
run_test "reta env sync appends missing keys"       test_env_sync_appends_missing_and_preserves_existing
run_test "reta env sync --dry-run keeps original"   test_env_sync_dry_run_does_not_modify
run_test "reta env sync handles comments & quotes"  test_env_sync_handles_inline_comments_and_quotes