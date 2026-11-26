//go:build darwin

package port

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

func collectPorts() ([]portEntry, error) {
	if _, err := exec.LookPath("lsof"); err != nil {
		return nil, fmt.Errorf("lsof is not available")
	}
	out, err := exec.Command("lsof", "-i", "-P", "-n").Output()
	if err != nil {
		return nil, nil
	}
	return parseLsofListen(string(out), dockerPortMap()), nil
}

func collectPIDsForPort(port string) []string {
	var pids []string
	seen := make(map[string]bool)
	for _, useSudo := range []bool{false, true} {
		var cmd *exec.Cmd
		if useSudo {
			cmd = exec.Command("sudo", "-n", "lsof", "-i", ":"+port, "-P", "-n", "-s", "TCP:LISTEN", "-t")
		} else {
			cmd = exec.Command("lsof", "-i", ":"+port, "-P", "-n", "-s", "TCP:LISTEN", "-t")
		}
		if out, err := cmd.Output(); err == nil {
			for _, line := range strings.Fields(string(out)) {
				if !seen[line] {
					seen[line] = true
					pids = append(pids, line)
				}
			}
			if len(pids) > 0 {
				return pids
			}
		}
	}
	return pids
}

func killPIDs(pids []string) int {
	c := ui.Colors()
	killed := 0
	var parsed []int
	for _, s := range pids {
		pid, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			continue
		}
		parsed = append(parsed, pid)
		if err := syscall.Kill(pid, syscall.SIGTERM); err == nil {
			killed++
		}
	}
	time.Sleep(1 * time.Second)
	for _, pid := range parsed {
		if err := syscall.Kill(pid, 0); err == nil {
			fmt.Printf("  %sWARN%s  PID %d did not exit, sending SIGKILL...\n", c.Yellow, c.Reset, pid)
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	return killed
}
