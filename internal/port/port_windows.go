//go:build windows

package port

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

func windowsTaskMap() map[string]string {
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return make(map[string]string)
	}
	return parseWindowsTasklistCSV(string(out))
}

func collectPorts() ([]portEntry, error) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run netstat: %w", err)
	}
	return parseWindowsNetstat(string(out), windowsTaskMap(), dockerPortMap()), nil
}

func collectPIDsForPort(port string) []string {
	entries, err := collectPorts()
	if err != nil {
		return nil
	}
	var pids []string
	seen := make(map[string]bool)
	for _, e := range entries {
		if e.Port == port && e.PID != "" && e.PID != "-" && e.PID != "docker" && !seen[e.PID] {
			seen[e.PID] = true
			pids = append(pids, e.PID)
		}
	}
	return pids
}

func isPIDAliveWindows(pid string) bool {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+pid, "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false
	}
	r := csv.NewReader(strings.NewReader(string(out)))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return false
	}
	for _, row := range rows {
		if len(row) >= 2 && strings.TrimSpace(row[1]) == pid {
			return true
		}
	}
	return false
}

func killPIDs(pids []string) int {
	c := ui.Colors()
	killed := 0
	graceful := make(map[string]bool)
	for _, pid := range pids {
		if err := exec.Command("taskkill", "/PID", pid, "/T").Run(); err == nil {
			graceful[pid] = true
			killed++
		}
	}
	if len(graceful) > 0 {
		time.Sleep(1 * time.Second)
	}
	for _, pid := range pids {
		if isPIDAliveWindows(pid) {
			if graceful[pid] {
				fmt.Printf("  %sWARN%s  PID %s did not exit, forcing termination...\n", c.Yellow, c.Reset, pid)
			}
			if err := exec.Command("taskkill", "/F", "/PID", pid, "/T").Run(); err == nil && !graceful[pid] {
				killed++
			}
		}
	}
	return killed
}
