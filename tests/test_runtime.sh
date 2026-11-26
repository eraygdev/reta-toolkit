#!/usr/bin/env bash
# Live OS runtime environment tests (Linux / macOS).

test_runtime_rcd_jumps_in_live_bash() {
    mkdir -p fakehome/work/alpha fakehome/work/beta
    echo "RETA*rid101010101010r" > fakehome/work/alpha/.retasign
    echo "RETA*rid202020202020r" > fakehome/work/beta/.retasign

    local landed
    landed=$(HOME="$PWD/fakehome" NO_COLOR=1 bash -c '
        set -e
        eval "$("'"$RETA_BIN"'" shell-init bash)"
        rcd beta >/dev/null
        pwd
    ')
    assert_eq "$PWD/fakehome/work/beta" "$landed" "rcd should change directory in live bash shell"
}

test_runtime_port_list_detects_live_socket() {
    command -v python3 >/dev/null 2>&1 || return 0
    if ! command -v ss >/dev/null 2>&1 && ! command -v lsof >/dev/null 2>&1; then
        return 0
    fi

    python3 -c '
import socket, time
s = socket.socket()
s.bind(("127.0.0.1", 0))
s.listen(1)
with open("port.txt", "w") as f:
    f.write(str(s.getsockname()[1]))
time.sleep(15)
' &
    local pid=$!
    trap 'kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true' RETURN

    local i=0
    while [ ! -s port.txt ] && [ "$i" -lt 30 ]; do
        sleep 0.05
        i=$((i + 1))
    done
    [ -s port.txt ] || return 1

    local port out
    port=$(cat port.txt)
    out=$(NO_COLOR=1 "$RETA_BIN" port list "$port")
    assert_contains "$out" "$port"
}

test_runtime_doctor_fix_repairs_base_mk() {
    mkdir -p fakehome/.local/bin
    cp "$RETA_BIN" fakehome/.local/bin/reta

    HOME="$PWD/fakehome" PATH="$PWD/fakehome/.local/bin:$PATH" NO_COLOR=1 "$RETA_BIN" doctor --fix >/dev/null
    assert_file_exists fakehome/.reta/lib/base.mk
    cmp -s "$RETA_ROOT/lib/base.mk" fakehome/.reta/lib/base.mk
}

test_runtime_cross_os_test_compile() {
    mkdir -p xbuild
    local os_arch goos goarch pkg
    for os_arch in "windows amd64" "darwin arm64"; do
        goos="${os_arch% *}"
        goarch="${os_arch#* }"
        (
            cd "$RETA_ROOT"
            CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -o "$PWD/xbuild/reta-${goos}" ./cmd/reta
            for pkg in ./cmd/reta ./internal/clean ./internal/port ./internal/doctor ./internal/shell; do
                CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go test -c -o "$PWD/xbuild/$(basename "$pkg")-${goos}.test" "$pkg"
            done
        )
    done
    rm -rf xbuild
}

run_test "runtime: rcd jumps inside live bash shell"    test_runtime_rcd_jumps_in_live_bash
run_test "runtime: port list detects live TCP socket" test_runtime_port_list_detects_live_socket
run_test "runtime: doctor --fix repairs lib/base.mk"  test_runtime_doctor_fix_repairs_base_mk
run_test "runtime: cross-OS compile & test-compile"   test_runtime_cross_os_test_compile