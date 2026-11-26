#!/usr/bin/env bash
# Tests for `lib/adopt-makefile` and `reta init --convert`.

test_adopt_unit_suite() {
    (cd "$RETA_ROOT" && go test ./internal/adopt)
}

test_adopt_dry_run_does_not_modify() {
    cat > Makefile <<'EOF'
build:
	echo building
EOF
    local out
    out=$(NO_COLOR=1 "$RETA_BIN" adopt-makefile Makefile --dry-run)
    assert_contains "$out" "include \$(RETA_BASE)"
    assert_file_not_exists Makefile.bak
    local current
    current=$(cat Makefile)
    assert_not_contains "$current" "RETA_BASE"
}

test_adopt_creates_backup_and_converts() {
    cat > Makefile <<'EOF'
help:
	@echo "  make run   Start service"

run:
	echo running
EOF
    NO_COLOR=1 "$RETA_BIN" adopt-makefile Makefile >/dev/null
    assert_file_exists Makefile.bak

    local orig conv
    orig=$(cat Makefile.bak)
    conv=$(cat Makefile)
    assert_not_contains "$orig" "RETA_BASE"
    assert_contains "$conv" "include \$(RETA_BASE)"
    assert_contains "$conv" "run:  ## Start service"
    assert_contains "$conv" "# help:"
}

test_adopt_no_backup_flag() {
    cat > Makefile <<'EOF'
build:
	echo ok
EOF
    NO_COLOR=1 "$RETA_BIN" adopt-makefile Makefile --no-backup >/dev/null
    assert_file_not_exists Makefile.bak
    local conv
    conv=$(cat Makefile)
    assert_contains "$conv" "build:  ## EDIT_HERE: add description"
}

test_adopt_skips_already_converted() {
    cat > Makefile <<'EOF'
RETA_BASE := $(HOME)/.reta/lib/base.mk
include $(RETA_BASE)

build:  ## Build
	echo ok
EOF
    local out
    out=$(NO_COLOR=1 "$RETA_BIN" adopt-makefile Makefile)
    assert_contains "$out" "skip"
    assert_file_not_exists Makefile.bak
}

test_adopt_missing_file_exits_1() {
    assert_exit_code 1 "$RETA_BIN" adopt-makefile nonexistent.mk
}

test_init_convert_flag_adopts_and_copies_other_templates() {
    cat > Makefile <<'EOF'
# --- BUILD ---
compile: # Compile app
	echo compiling
EOF
    NO_COLOR=1 "$RETA_BIN" init . --convert >/dev/null
    assert_file_exists Makefile.bak
    assert_file_exists .env.example
    assert_file_exists .gitignore
    assert_file_exists README.md

    local conv
    conv=$(cat Makefile)
    assert_contains "$conv" "# === BUILD ==="
    assert_contains "$conv" "compile:  ## Compile app"
}

test_converted_makefile_runs_with_make() {
    command -v make >/dev/null 2>&1 || return 0

    mkdir -p fakehome
    ln -sfn "$RETA_ROOT" fakehome/.reta

    cat > Makefile <<'EOF'
help:
	@echo "  make status   Check status"

status:
	@echo "STATUS_OK"

# --- DATA ---
migrate:
	@echo "MIGRATED"
EOF
    NO_COLOR=1 "$RETA_BIN" adopt-makefile Makefile --no-backup >/dev/null

    local help_out run_out
    help_out=$(HOME="$PWD/fakehome" make --no-print-directory help)
    assert_contains "$help_out" "Project-specific targets"
    assert_contains "$help_out" "status"
    assert_contains "$help_out" "Check status"
    assert_contains "$help_out" "DATA"
    assert_contains "$help_out" "migrate"
    assert_contains "$help_out" "EDIT_HERE: add description"

    # Verify blank line between 'Project-specific targets' and preserved targets
    python3 -c '
import sys
lines = sys.stdin.read().splitlines()
idx = next(i for i, l in enumerate(lines) if "Project-specific targets" in l)
assert lines[idx + 1].strip() == "", f"expected blank line after header, got {lines[idx + 1]!r}"
assert "hi" in lines[idx + 2], f"expected hi target first, got {lines[idx + 2]!r}"
assert "status" in lines[idx + 3], f"expected status target second, got {lines[idx + 3]!r}"
' <<<"$help_out"

    local conv_mk hi_out
    conv_mk=$(cat Makefile)
    assert_contains "$conv_mk" "hi:  ## Say hello"
    hi_out=$(HOME="$PWD/fakehome" make --no-print-directory hi)
    assert_eq "Hi👋" "$hi_out"

    run_out=$(HOME="$PWD/fakehome" make --no-print-directory status)
    assert_eq "STATUS_OK" "$run_out"
}

test_adopt_skips_greeting_hi_when_target_exists() {
    cat > Makefile <<'EOF'
hi:  ## Custom greeting
	@echo "CUSTOM_HI"
EOF
    NO_COLOR=1 "$RETA_BIN" adopt-makefile Makefile --no-backup >/dev/null
    local conv
    conv=$(cat Makefile)
    assert_contains "$conv" "Custom greeting"
    assert_not_contains "$conv" "Say hello"
}

run_test "adopt-makefile unit test suite"           test_adopt_unit_suite
run_test "adopt-makefile --dry-run keeps original"  test_adopt_dry_run_does_not_modify
run_test "adopt-makefile creates backup & converts" test_adopt_creates_backup_and_converts
run_test "adopt-makefile --no-backup skips backup"  test_adopt_no_backup_flag
run_test "adopt-makefile skips already converted"   test_adopt_skips_already_converted
run_test "adopt-makefile exits 1 on missing file"   test_adopt_missing_file_exits_1
run_test "adopt-makefile keeps existing hi target"  test_adopt_skips_greeting_hi_when_target_exists
run_test "reta init --convert adopts Makefile"      test_init_convert_flag_adopts_and_copies_other_templates
run_test "converted Makefile works with GNU make"   test_converted_makefile_runs_with_make