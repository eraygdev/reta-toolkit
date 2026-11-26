package port

import (
	"encoding/csv"
	"regexp"
	"strconv"
	"strings"
)

var (
	ssProcRE = regexp.MustCompile(`users:\(\("([^"]+)"`)
	ssPidRE  = regexp.MustCompile(`pid=([0-9]+)`)
)

func parseWindowsTasklistCSV(raw string) map[string]string {
	procMap := make(map[string]string)
	r := csv.NewReader(strings.NewReader(raw))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return procMap
	}
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		name := strings.TrimSpace(row[0])
		if len(name) > 4 && strings.EqualFold(name[len(name)-4:], ".exe") {
			name = name[:len(name)-4]
		}
		pid := strings.TrimSpace(row[1])
		if pid != "" {
			procMap[pid] = name
		}
	}
	return procMap
}

func parseWindowsNetstat(raw string, taskMap, dkrMap map[string]string) []portEntry {
	seen := make(map[string]bool)
	var entries []portEntry

	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
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
	return entries
}

func parseLsofListen(raw string, dkrMap map[string]string) []portEntry {
	seen := make(map[string]bool)
	var entries []portEntry

	for i, line := range strings.Split(raw, "\n") {
		if i == 0 || !strings.Contains(line, "(LISTEN)") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		addr := ""
		p := ""
		for _, f := range fields {
			if idx := strings.LastIndex(f, ":"); idx >= 0 && idx+1 < len(f) {
				cand := f[idx+1:]
				if _, err := strconv.Atoi(cand); err == nil {
					p = cand
					addr = f
				}
			}
		}
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		if cname, ok := dkrMap[p]; ok {
			entries = append(entries, portEntry{Port: p, Command: cname, PID: "docker", Addr: addr})
		} else {
			entries = append(entries, portEntry{Port: p, Command: fields[0], PID: fields[1], Addr: addr})
		}
	}
	return entries
}

func parseSSListen(raw string, dkrMap map[string]string) []portEntry {
	seen := make(map[string]bool)
	var entries []portEntry

	for i, line := range strings.Split(raw, "\n") {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[1] != "LISTEN" {
			continue
		}
		addr := fields[4]
		idx := strings.LastIndex(addr, ":")
		if idx < 0 || idx+1 >= len(addr) {
			continue
		}
		p := addr[idx+1:]
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true

		if cname, ok := dkrMap[p]; ok {
			entries = append(entries, portEntry{Port: p, Command: cname, PID: "docker", Addr: addr})
			continue
		}

		proc := ""
		pid := ""
		if m := ssProcRE.FindStringSubmatch(line); len(m) == 2 {
			proc = m[1]
		}
		if m := ssPidRE.FindStringSubmatch(line); len(m) == 2 {
			pid = m[1]
		}
		if proc == "" {
			proc = serviceNameFor(p)
			if proc == "" {
				proc = "-"
			}
		}
		if pid == "" {
			pid = "-"
		}
		entries = append(entries, portEntry{Port: p, Command: proc, PID: pid, Addr: addr})
	}
	return entries
}
