package port

import (
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func TestCollectPortsAndPIDsLiveListener(t *testing.T) {
	testutil.SilenceStdio(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind loopback listener: %v", err)
	}
	defer ln.Close()

	portStr := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	entries, err := collectPorts()
	if err != nil {
		t.Skipf("port inspection tool unavailable in this environment: %v", err)
	}

	var found *portEntry
	for i := range entries {
		if entries[i].Port == portStr {
			found = &entries[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected live listener on port %s to appear in collectPorts()", portStr)
	}

	selfPID := strconv.Itoa(os.Getpid())
	if found.PID != "-" && found.PID != selfPID {
		t.Fatalf("port %s PID = %q, want %q", portStr, found.PID, selfPID)
	}

	pids := collectPIDsForPort(portStr)
	if len(pids) > 0 {
		matched := false
		for _, p := range pids {
			if p == selfPID {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("collectPIDsForPort(%s) = %v, expected %s", portStr, pids, selfPID)
		}
	}

	if code := cmdList([]string{portStr}); code != 0 {
		t.Fatalf("cmdList(%s) = %d, want 0", portStr, code)
	}
}

func TestKillHelperProcess(t *testing.T) {
	if os.Getenv("RETA_KILL_HELPER") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

func TestKillPIDsTerminatesSubprocess(t *testing.T) {
	testutil.SilenceStdio(t)

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestKillHelperProcess$")
	cmd.Env = append(os.Environ(), "RETA_KILL_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper subprocess: %v", err)
	}
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})

	time.Sleep(150 * time.Millisecond)
	pidStr := strconv.Itoa(cmd.Process.Pid)
	if killed := killPIDs([]string{pidStr}); killed != 1 {
		t.Fatalf("killPIDs([%s]) = %d, want 1", pidStr, killed)
	}

	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
		t.Fatal("helper subprocess did not exit after killPIDs")
	}
}
