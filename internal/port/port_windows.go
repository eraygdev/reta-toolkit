//go:build windows

package port

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

func windowsTaskMap() map[string]string {
	procMap := make(map[string]string)
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return procMap
	}
	r := csv.NewReader(strings.NewReader(string(out)))
	rows, err := r.ReadAll()
	if err != nil {
		return procMap
	}
	for _, row := range rows {
		if len(row) >= 2 {
			procMap[strings.TrimSpace(row[1])] = strings.TrimSuffix(strings.TrimSpace(row[0]), ".exe")
		}
	}
	return procMap
}

func collectPorts() ([]portEntry, error) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run netstat: %w", err)
	}
	dkrMap := dockerPortMap()
	taskMap := windowsTaskMap()
	seen := make(map[string]bool)
	var entries []portEntry

	for _, raw := range strings.Split(string(out), "\n") {
		fields := strings.Fields(raw)
		if len(fields) < 5 {
			continue
		}
		if !strings.EqualFold(fields[0], "TCP") || !strings.EqualFold(fields[3], "LISTENING") {
			continue
		}
		addr := fields[1]
		idx := strings.LastIndex(addr, ":")
		if idx < 0 || idx+1 >= len(addr) {
			continue
		}
		p := addr[idx+1:]
		if _, err := strconv.Atoi(p); err != nil || seen[p] {
			continue
		}
		seen[p] = true

		if cname, ok := dkrMap[p]; ok {
			entries = append(entries, portEntry{Port: p, Command: cname, PID: "docker", Addr: addr})
			continue
		}
		pid := fields[4]
		proc := taskMap[pid]
		if proc == "" {
			proc = serviceNameFor(p)
			if proc == "" {
				proc = "-"
			}
		}
		entries = append(entries, portEntry{Port: p, Command: proc, PID: pid, Addr: addr})
	}
	return entries, nil
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
	for _, pid := range pids {
		if err := exec.Command("taskkill", "/PID", pid, "/T").Run(); err == nil {
			killed++
		}
	}
	time.Sleep(1 * time.Second)
	for _, pid := range pids {
		if isPIDAliveWindows(pid) {
			fmt.Printf("  %sWARN%s  PID %s did not exit, forcing termination...\n", c.Yellow, c.Reset, pid)
			_ = exec.Command("taskkill", "/F", "/PID", pid, "/T").Run()
		}
	}
	return killed
}
