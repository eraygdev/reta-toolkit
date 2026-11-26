#!/usr/bin/env bash
# Tests for `reta init-env`.

test_init_env_generates_secrets() {
    cat > .env.example <<'EOF'
APP_KEY=GENERATE_SECRET
JWT_SECRET=GENERATE_SECRET
EOF

    "$RETA_BIN" init-env >/dev/null

    assert_file_exists .env
    local content
    content=$(cat .env)
    assert_not_contains "$content" "GENERATE_SECRET"

    local key
    key=$(grep '^APP_KEY=' .env | cut -d= -f2)
    assert_eq "64" "${#key}" "APP_KEY should be 64 hex chars"
}

test_init_env_refuses_overwrite() {
    echo "APP_KEY=x" > .env.example
    echo "APP_KEY=existing" > .env

    assert_exit_code 1 "$RETA_BIN" init-env

    local content
    content=$(cat .env)
    assert_eq "APP_KEY=existing" "$content"
}

test_init_env_force_overwrites() {
    echo "APP_KEY=x" > .env.example
    echo "APP_KEY=existing" > .env

    "$RETA_BIN" init-env --force >/dev/null

    local content
    content=$(cat .env)
    assert_not_contains "$content" "existing"
}

test_init_env_missing_template() {
    assert_exit_code 1 "$RETA_BIN" init-env
}

run_test "reta init-env generates secrets"    test_init_env_generates_secrets
run_test "reta init-env refuses overwrite"    test_init_env_refuses_overwrite
run_test "reta init-env --force overwrites"   test_init_env_force_overwrites
run_test "reta init-env missing template"     test_init_env_missing_template