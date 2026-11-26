#!/usr/bin/env bash
# Minimal test helpers. Zero dependencies, pure bash.

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
CURRENT_TEST=""
TEST_TMP=""

if [ -z "${NO_COLOR:-}" ] && [ -t 1 ]; then
    C_BOLD=$'\033[1m'
    C_DIM=$'\033[2m'
    C_GREEN=$'\033[32m'
    C_RED=$'\033[31m'
    C_YELLOW=$'\033[1;33m'
    C_CYAN=$'\033[36m'
    C_RESET=$'\033[0m'
else
    C_BOLD=""
    C_DIM=""
    C_GREEN=""
    C_RED=""
    C_YELLOW=""
    C_CYAN=""
    C_RESET=""
fi

assert_eq() {
    local expected="$1"
    local actual="$2"
    local message="${3:-assertion failed}"

    if [ "$expected" = "$actual" ]; then
        return 0
    fi
    echo "    expected: '$expected'"
    echo "    actual:   '$actual'"
    echo "    context:  $message"
    return 1
}

assert_exit_code() {
    local expected="$1"
    shift
    local actual
    set +e
    "$@" >/dev/null 2>&1
    actual=$?
    set -e
    if [ "$expected" = "$actual" ]; then
        return 0
    fi
    echo "    expected exit code: $expected"
    echo "    actual:             $actual"
    echo "    command:            $*"
    return 1
}

assert_file_exists() {
    local path="$1"
    if [ -f "$path" ]; then
        return 0
    fi
    echo "    expected file to exist: $path"
    return 1
}

assert_file_not_exists() {
    local path="$1"
    if [ ! -e "$path" ]; then
        return 0
    fi
    echo "    expected file to NOT exist: $path"
    return 1
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    if [[ "$haystack" == *"$needle"* ]]; then
        return 0
    fi
    echo "    expected to contain: '$needle'"
    echo "    actual:              '$haystack'"
    return 1
}

assert_not_contains() {
    local haystack="$1"
    local needle="$2"
    if [[ "$haystack" != *"$needle"* ]]; then
        return 0
    fi
    echo "    expected NOT to contain: '$needle'"
    echo "    actual:                  '$haystack'"
    return 1
}

run_test() {
    local name="$1"
    local fn="$2"

    TESTS_RUN=$((TESTS_RUN + 1))
    CURRENT_TEST="$name"

    TEST_TMP=$(mktemp -d)

    local output rc
    output=$( ( set -e; cd "$TEST_TMP"; "$fn" ) 2>&1 )
    rc=$?

    if [ "$rc" -eq 0 ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        printf "  %sPASS%s  %s\n" "$C_GREEN" "$C_RESET" "$name"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        printf "  %sFAIL%s  %s %s(rc=%d)%s\n" "$C_RED" "$C_RESET" "$name" "$C_DIM" "$rc" "$C_RESET"
        [ -n "$output" ] && printf "%s\n" "$output" | sed 's/^/    /'
    fi

    rm -rf "$TEST_TMP"
    TEST_TMP=""
}

report() {
    echo ""
    if [ "$TESTS_FAILED" -eq 0 ]; then
        printf "%sResults:%s %s%d passed%s, %d failed, %d total\n" \
            "$C_BOLD" "$C_RESET" "$C_GREEN" "$TESTS_PASSED" "$C_RESET" "$TESTS_FAILED" "$TESTS_RUN"
    else
        printf "%sResults:%s %s%d passed%s, %s%d failed%s, %d total\n" \
            "$C_BOLD" "$C_RESET" "$C_GREEN" "$TESTS_PASSED" "$C_RESET" "$C_RED" "$TESTS_FAILED" "$C_RESET" "$TESTS_RUN"
    fi
    [ "$TESTS_FAILED" -eq 0 ]
}