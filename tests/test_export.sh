#!/usr/bin/env bash
# Tests for `lib/export-context` and `base.mk` export targets with DIR=.

test_export_defaults_to_exports_dir() {
    mkdir -p proj
    echo "print('hello')" > proj/app.py

    "$RETA_BIN" export -r proj >/dev/null
    assert_file_exists proj/exports/project_export.md
    assert_file_not_exists proj/project_export.md
}

test_export_dir_dot_writes_to_root() {
    mkdir -p proj
    echo "print('hello')" > proj/app.py

    "$RETA_BIN" export -r proj --dir . >/dev/null
    assert_file_exists proj/project_export.md
    assert_file_not_exists proj/exports/project_export.md
}

test_clean_regenerates_export_in_custom_dir() {
    mkdir -p proj/custom_out
    echo "OLD_CONTENT" > proj/custom_out/project_export.md
    echo "print('new')" > proj/app.py

    "$RETA_BIN" export -r proj --dir custom_out --clean >/dev/null
    assert_file_exists proj/custom_out/project_export.md
    local content
    content=$(cat proj/custom_out/project_export.md)
    assert_not_contains "$content" "OLD_CONTENT"
    assert_contains "$content" "print('new')"
}

test_clean_exports_refuses_non_exports_dir() {
    mkdir -p proj/custom_out
    echo "keep" > proj/custom_out/project_export.md
    echo "print('hi')" > proj/app.py

    assert_exit_code 1 "$RETA_BIN" export -r proj --dir custom_out --clean-exports
    assert_file_exists proj/custom_out/project_export.md
}

test_clean_exports_refuses_when_project_root_named_exports() {
    mkdir -p exports/exports
    echo "keep" > exports/exports/project_export.md
    echo "print('hi')" > exports/app.py

    assert_exit_code 1 "$RETA_BIN" export -r exports --dir exports --clean-exports
    assert_file_exists exports/exports/project_export.md
}

test_clean_exports_clears_files_and_keeps_dir() {
    mkdir -p proj/exports/sub
    echo "a" > proj/exports/project_export.md
    echo "b" > proj/exports/sub/extra.txt
    echo "print('hi')" > proj/app.py

    local out
    out=$(printf "y\n" | "$RETA_BIN" export -r proj --clean-exports 2>&1)
    assert_contains "$out" "2 file(s)"
    assert_file_not_exists proj/exports/project_export.md
    assert_file_not_exists proj/exports/sub/extra.txt
    if [ ! -d proj/exports ]; then
        echo "expected proj/exports directory to remain"
        return 1
    fi
}

test_make_help_displays_dynamic_dir() {
    command -v make >/dev/null 2>&1 || return 0

    mkdir -p fakehome
    ln -sfn "$RETA_ROOT" fakehome/.reta
    cat > Makefile <<'EOF'
RETA_BASE := $(HOME)/.reta/lib/base.mk
include $(RETA_BASE)

DIR := custom_exports
EOF

    local out_proj out_cli
    out_proj=$(HOME="$PWD/fakehome" make --no-print-directory help)
    assert_contains "$out_proj" "Export source for AI context (DIR=custom_exports)"

    out_cli=$(HOME="$PWD/fakehome" make --no-print-directory help DIR=.)
    assert_contains "$out_cli" "Export source for AI context (DIR=.)"
}

run_test "export defaults to exports/ dir"                   test_export_defaults_to_exports_dir
run_test "export --dir . writes to project root"             test_export_dir_dot_writes_to_root
run_test "export --clean regenerates in custom dir"          test_clean_regenerates_export_in_custom_dir
run_test "clean-exports refuses non-exports dir"             test_clean_exports_refuses_non_exports_dir
run_test "clean-exports refuses project root named exports"  test_clean_exports_refuses_when_project_root_named_exports
run_test "clean-exports removes contents and keeps folder"   test_clean_exports_clears_files_and_keeps_dir
test_make_export_src_writes_to_project_exports_dir() {
    command -v make >/dev/null 2>&1 || return 0

    mkdir -p fakehome src other
    ln -sfn "$RETA_ROOT" fakehome/.reta
    echo "print('src')" > src/main.py
    echo "print('other')" > other/ignored.py
    cat > Makefile <<'EOF'
RETA_BASE := $(HOME)/.reta/lib/base.mk
include $(RETA_BASE)
EOF

    HOME="$PWD/fakehome" make --no-print-directory export SRC=./src >/dev/null
    assert_file_exists exports/project_export.md
    assert_file_not_exists src/exports/project_export.md
    local content
    content=$(cat exports/project_export.md)
    assert_contains "$content" "main.py"
    assert_not_contains "$content" "ignored.py"
}

run_test "make help reflects DIR dynamically"                test_make_help_displays_dynamic_dir
run_test "make export SRC=./src writes to project exports"   test_make_export_src_writes_to_project_exports_dir