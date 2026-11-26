#!/usr/bin/env bash
# Smoke and integration tests for Go reta subcommands.

test_all_subcommands_support_help() {
    local name out
    for name in init init-env env config hook update uninstall clean search check-secrets export adopt-makefile port completion shell-init doctor; do
        out=$(NO_COLOR=1 "$RETA_BIN" help "$name" 2>&1)
        if [ -z "$out" ]; then
            echo "    expected non-empty help output for $name"
            return 1
        fi
    done
}

test_hook_works_without_makefile_and_blocks_staged_secret() {
    command -v git >/dev/null 2>&1 || return 0

    mkdir -p fakehome
    ln -sfn "$RETA_ROOT" fakehome/.reta

    git init -q
    git config user.email "test@example.com"
    git config user.name "Test"
    echo ".env" > .gitignore
    echo "print('ok')" > app.py
    git add .gitignore app.py

    HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" hook install >/dev/null
    assert_file_exists .git/hooks/pre-commit

    local status_out
    status_out=$(HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" hook status)
    assert_contains "$status_out" "Active"

    # Clean commit without a Makefile must succeed
    HOME="$PWD/fakehome" git commit -q -m "clean commit"

    # Unstaged secret on disk must NOT block a clean staged file
    local p1="sk-proj-"
    local p2="abcdefghijklmnopqrstuvwxyz1234"
    echo "KEY = \"${p1}${p2}\"" > secret_unstaged.py
    echo "print('clean')" > clean2.py
    git add clean2.py
    HOME="$PWD/fakehome" git commit -q -m "clean staged commit"

    # Staged secret MUST be blocked
    git add secret_unstaged.py
    assert_exit_code 1 env HOME="$PWD/fakehome" git commit -q -m "should fail"

    HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" hook remove >/dev/null
    assert_file_not_exists .git/hooks/pre-commit
}

test_search_filters_dead_dirs_and_jumps() {
    mkdir -p fakehome valid_proj not_a_proj deleted_proj
    echo "RETA*rid111111111111r" > valid_proj/.retasign
    rmdir deleted_proj

    mkdir -p fakehome/.config/reta
    cat > fakehome/.config/reta/projects.txt <<EOF
$PWD/valid_proj
$PWD/not_a_proj
$PWD/deleted_proj
EOF

    local out
    out=$(printf "n\n" | HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" search valid_proj)
    assert_contains "$out" "Jumping to"
    assert_file_exists fakehome/.cache/reta/last_path

    local jumped
    jumped=$(cat fakehome/.cache/reta/last_path)
    assert_eq "$PWD/valid_proj" "$jumped"

    local reg
    reg=$(cat fakehome/.config/reta/projects.txt)
    assert_contains "$reg" "valid_proj"
    assert_not_contains "$reg" "not_a_proj"
    assert_not_contains "$reg" "deleted_proj"
}

test_search_prompts_to_add_missing_signature() {
    mkdir -p fakehome missing_sig_proj
    mkdir -p fakehome/.config/reta
    echo "$PWD/missing_sig_proj" > fakehome/.config/reta/projects.txt

    printf "y\nq\n" | HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" search >/dev/null 2>&1
    assert_file_exists missing_sig_proj/.retasign
    local reg
    reg=$(cat fakehome/.config/reta/projects.txt)
    assert_contains "$reg" "missing_sig_proj"
}

test_search_discovers_depth_and_handles_name_collision() {
    mkdir -p fakehome/a/b/c/d/deep_proj fakehome/team1/app fakehome/team2/app
    echo "RETA*rid222222222222r" > fakehome/team1/app/.retasign
    echo "RETA*rid333333333333r" > fakehome/team2/app/.retasign
    echo "RETA*rid444444444444r" > fakehome/a/b/c/d/deep_proj/.retasign

    local list_out
    list_out=$(printf "q\n" | HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" search 2>&1)
    assert_contains "$list_out" "apprid222222222222r"
    assert_contains "$list_out" "apprid333333333333r"
    assert_not_contains "$list_out" "deep_proj"
    assert_contains "$list_out" "reta config search-depth"

    # Exact name collision picks the first match
    local jump_out jumped
    jump_out=$(HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" search app)
    assert_contains "$jump_out" "Jumping to"
    jumped=$(cat fakehome/.cache/reta/last_path)
    assert_eq "$PWD/fakehome/team1/app" "$jumped"

    # Increasing search-depth to 5 discovers deep_proj
    HOME="$PWD/fakehome" "$RETA_BIN" config search-depth 5 >/dev/null
    jump_out=$(HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" search deep_proj)
    assert_contains "$jump_out" "Jumping to"
    jumped=$(cat fakehome/.cache/reta/last_path)
    assert_eq "$PWD/fakehome/a/b/c/d/deep_proj" "$jumped"
}

test_init_env_non_tty_handles_empty_keys() {
    cat > .env.example <<'EOF'
APP_KEY=GENERATE_SECRET # auto
EMPTY_KEY=
export STATIC_VAR=hello
EOF

    NO_COLOR=1 "$RETA_BIN" init-env </dev/null >/dev/null
    assert_file_exists .env

    local content
    content=$(cat .env)
    assert_not_contains "$content" "GENERATE_SECRET"
    assert_contains "$content" "EMPTY_KEY="
    assert_contains "$content" "export STATIC_VAR=hello"
}

test_doctor_runs_without_installed_home_reta() {
    mkdir -p fakehome
    local out
    set +e
    out=$(HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" doctor 2>&1)
    set -e
    assert_contains "$out" "Summary:"
    assert_contains "$out" "~/.reta missing"
}

test_uninstall_cleans_symlinks_and_completions() {
    mkdir -p fakehome/.local/bin fakehome/.local/share/bash-completion/completions fakehome/.zfunc
    ln -sfn "$RETA_ROOT" fakehome/.reta
    cp "$RETA_BIN" fakehome/.local/bin/reta
    touch fakehome/.local/share/bash-completion/completions/reta
    touch fakehome/.zfunc/_reta

    HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" uninstall -y >/dev/null
    assert_file_not_exists fakehome/.local/bin/reta
    assert_file_not_exists fakehome/.local/share/bash-completion/completions/reta
    assert_file_not_exists fakehome/.zfunc/_reta
    assert_file_not_exists fakehome/.reta
}

test_hook_and_makefile_report_not_installed() {
    if command -v git >/dev/null 2>&1; then
        mkdir -p fakehome
        ln -sfn "$RETA_ROOT" fakehome/.reta

        git init -q
        git config user.email "test@example.com"
        git config user.name "Test"
        echo "print('ok')" > app.py
        git add app.py

        HOME="$PWD/fakehome" NO_COLOR=1 "$RETA_BIN" hook install >/dev/null
        rm -f fakehome/.reta

        local git_dir clean_path hook_out hook_rc
        git_dir="$(dirname "$(command -v git)")"
        clean_path="${git_dir}:/usr/bin:/bin"
        set +e
        hook_out=$(PATH="$clean_path" HOME="$PWD/fakehome" git commit -q -m "should fail" 2>&1)
        hook_rc=$?
        set -e

        assert_eq "1" "$hook_rc" "pre-commit should fail when ~/.reta is missing"
        assert_contains "$hook_out" "reta toolkit is not installed."
        assert_contains "$hook_out" "git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta && ~/.reta/install.sh"
    fi

    if command -v make >/dev/null 2>&1; then
        mkdir -p fakehome_mk
        cp "$RETA_ROOT/templates/Makefile" Makefile

        local mk_out mk_rc
        set +e
        mk_out=$(HOME="$PWD/fakehome_mk" make help 2>&1)
        mk_rc=$?
        set -e

        if [ "$mk_rc" -eq 0 ]; then
            echo "    expected make to fail when ~/.reta is missing"
            return 1
        fi
        assert_contains "$mk_out" "reta toolkit is not installed."
        assert_contains "$mk_out" "git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta && ~/.reta/install.sh"
    fi
}

test_hook_works_from_git_subdirectory() {
    command -v git >/dev/null 2>&1 || return 0

    mkdir -p fakehome sub/nested
    ln -sfn "$RETA_ROOT" fakehome/.reta

    git init -q
    (
        cd sub/nested
        HOME="$PWD/../../fakehome" NO_COLOR=1 "$RETA_BIN" hook install >/dev/null
    )
    assert_file_exists .git/hooks/pre-commit

    local status_out
    status_out=$(cd sub/nested && HOME="$PWD/../../fakehome" NO_COLOR=1 "$RETA_BIN" hook status)
    assert_contains "$status_out" "Active"
}

run_test "all subcommands support --help"                test_all_subcommands_support_help
run_test "reta hook works without Makefile (staged)"     test_hook_works_without_makefile_and_blocks_staged_secret
run_test "hook and Makefile report missing reta"         test_hook_and_makefile_report_not_installed
run_test "reta hook works from git subdirectory"         test_hook_works_from_git_subdirectory
run_test "reta search filters dead dirs & jumps"         test_search_filters_dead_dirs_and_jumps
run_test "reta search adds missing signature on prompt"  test_search_prompts_to_add_missing_signature
run_test "reta search depth & name collision handling"   test_search_discovers_depth_and_handles_name_collision
run_test "reta init-env handles non-TTY empty keys"      test_init_env_non_tty_handles_empty_keys
run_test "reta doctor diagnoses missing ~/.reta"         test_doctor_runs_without_installed_home_reta
run_test "reta uninstall removes bin & completion links" test_uninstall_cleans_symlinks_and_completions