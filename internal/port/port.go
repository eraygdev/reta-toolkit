package port

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

type portEntry struct {
	Port    string
	Command string
	PID     string
	Addr    string
}

var knownSystemPorts = map[string]bool{
	"22": true, "53": true, "80": true, "123": true, "443": true, "631": true,
	"3306": true, "5432": true, "6379": true, "27017": true, "11211": true,
}

var wellKnownServices = map[string]string{
	"22": "ssh", "53": "dns", "80": "http", "123": "ntp", "443": "https",
	"631": "cups", "3306": "mysql", "5432": "postgres", "6379": "redis",
	"27017": "mongodb", "11211": "memcached",
}

func printHelp() {
	fmt.Println("reta port — list or kill listening ports")
	fmt.Println("")
	fmt.Println("Usage: reta port <command> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  list              List all listening ports (grouped)")
	fmt.Println("  list <PORT>       Show only the process on PORT")
	fmt.Println("  kill              Interactive selector (pick from a menu)")
	fmt.Println("  kill <PORT>       Kill whatever listens on PORT")
	fmt.Println("  docker            List all active/running Docker containers")
	fmt.Println("")
	fmt.Println("Port classes:")
	fmt.Println("  [SYSTEM]  Privileged (<1024) or known service ports (22, 53, 443, ...)")
	fmt.Println("  [DOCKER]  Handled by a docker container")
	fmt.Println("  [USER]    Regular user processes")
}

func serviceNameFor(port string) string {
	if s, ok := wellKnownServices[port]; ok {
		return s
	}
	if runtime.GOOS != "windows" {
		if data, err := os.ReadFile("/etc/services"); err == nil {
			tcpSuffix := port + "/tcp"
			udpSuffix := port + "/udp"
			for _, raw := range strings.Split(string(data), "\n") {
				line := strings.TrimSpace(raw)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) >= 2 && (fields[1] == tcpSuffix || fields[1] == udpSuffix) {
					return fields[0]
				}
			}
		}
	}
	return ""
}

func portClass(port, pid string) string {
	if pid == "docker" {
		return "docker"
	}
	if knownSystemPorts[port] {
		return "system"
	}
	if p, err := strconv.Atoi(port); err == nil && p < 1024 {
		return "system"
	}
	return "user"
}

func dockerAvailable() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	return exec.Command("docker", "info").Run() == nil
}

func dockerPortMap() map[string]string {
	outMap := make(map[string]string)
	if !dockerAvailable() {
		return outMap
	}
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}|{{.Ports}}").Output()
	if err != nil {
		return outMap
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		name, portStr, ok := strings.Cut(line, "|")
		if !ok || strings.TrimSpace(portStr) == "" {
			continue
		}
		for _, mapping := range strings.Split(portStr, ",") {
			m := strings.TrimSpace(mapping)
			arrowIdx := strings.Index(m, "->")
			if arrowIdx <= 0 {
				continue
			}
			left := m[:arrowIdx]
			colonIdx := strings.LastIndex(left, ":")
			if colonIdx >= 0 && colonIdx+1 < len(left) {
				p := left[colonIdx+1:]
				if _, err := strconv.Atoi(p); err == nil {
					outMap[p] = name
				}
			}
		}
	}
	return outMap
}

func formatEntry(e portEntry) string {
	return fmt.Sprintf("%-8s %-20s %-8s %s", e.Port, e.Command, e.PID, e.Addr)
}

func validatePort(port string) bool {
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Invalid port: %s (must be 1-65535)\n", ui.Colors().Red, ui.Colors().Reset, port)
		return false
	}
	return true
}

func printGroup(label string, entries []portEntry) {
	if len(entries) == 0 {
		return
	}
	c := ui.Colors()
	fmt.Printf("\n  %s%s%s\n", c.Bold, label, c.Reset)
	for _, e := range entries {
		fmt.Printf("  %s\n", formatEntry(e))
	}
}

func cmdDocker() int {
	c := ui.Colors()
	if !dockerAvailable() {
		fmt.Fprintf(os.Stderr, "  %sERR%s   Docker is not available or not running.\n", c.Red, c.Reset)
		return 1
	}
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}|{{.Image}}|{{.Status}}|{{.Ports}}").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		ui.PrintDim(os.Stdout, "No running Docker containers found.")
		return 0
	}

	fmt.Printf("\n  %s%-20s %-22s %-12s %s%s\n", c.Bold, "NAME", "IMAGE", "STATUS", "PORTS", c.Reset)
	fmt.Println("  ────────────────────────────────────────────────────────────────────────────────")

	for _, raw := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if raw == "" {
			continue
		}
		parts := strings.SplitN(raw, "|", 4)
		if len(parts) < 4 {
			continue
		}
		name, image, status, ports := parts[0], parts[1], parts[2], parts[3]

		for _, rep := range [][2]string{
			{"minutes", "m"}, {"minute", "m"},
			{"hours", "h"}, {"hour", "h"},
			{"days", "d"}, {"day", "d"},
			{"seconds", "s"}, {"second", "s"},
			{"About an", "~"}, {"an ", "1 "},
		} {
			status = strings.ReplaceAll(status, rep[0], rep[1])
		}

		if strings.TrimSpace(ports) == "" {
			ports = "-"
		} else {
			ports = strings.ReplaceAll(ports, "[::]:", "")
			ports = strings.ReplaceAll(ports, "0.0.0.0:", "")
			seen := make(map[string]bool)
			var uniq []string
			for _, p := range strings.Split(ports, ",") {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" && !seen[trimmed] {
					seen[trimmed] = true
					uniq = append(uniq, trimmed)
				}
			}
			ports = strings.Join(uniq, ", ")
		}

		if len(image) > 22 {
			image = image[:19] + "..."
		}
		fmt.Printf("  %s%-20s%s %-22s %-12s %s\n", c.Cyan, name, c.Reset, image, status, ports)
	}
	fmt.Println("")
	return 0
}

func cmdList(args []string) int {
	filter := ""
	if len(args) > 0 {
		filter = args[0]
		if !validatePort(filter) {
			return 1
		}
	}
	entries, err := collectPorts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   %v\n", ui.Colors().Red, ui.Colors().Reset, err)
		return 1
	}
	if len(entries) == 0 {
		ui.PrintDim(os.Stdout, "No listening ports found.")
		return 0
	}

	var filtered []portEntry
	for _, e := range entries {
		if filter == "" || e.Port == filter {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		ui.PrintDim(os.Stdout, fmt.Sprintf("No process listening on port %s.", filter))
		return 0
	}

	var sysEntries, dkrEntries, usrEntries []portEntry
	for _, e := range filtered {
		switch portClass(e.Port, e.PID) {
		case "system":
			sysEntries = append(sysEntries, e)
		case "docker":
			dkrEntries = append(dkrEntries, e)
		default:
			usrEntries = append(usrEntries, e)
		}
	}

	c := ui.Colors()
	fmt.Printf("  %s%-8s %-20s %-8s %s%s\n", c.Bold, "PORT", "COMMAND", "PID", "ADDRESS", c.Reset)
	printGroup("SYSTEM", sysEntries)
	printGroup("DOCKER", dkrEntries)
	printGroup("USER", usrEntries)
	return 0
}

func confirmYes(prompt string) bool {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes"
}

func cmdKillByPort(port string) int {
	if !validatePort(port) {
		return 1
	}
	c := ui.Colors()
	dkrMap := dockerPortMap()
	if container, ok := dkrMap[port]; ok {
		fmt.Println("")
		fmt.Printf("Port %s is handled by Docker container: %s\n", port, container)
		if !confirmYes("Stop this container? [y/N] ") {
			ui.PrintDim(os.Stdout, "Cancelled.")
			return 0
		}
		if err := exec.Command("docker", "stop", container).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  %sERR%s   failed to stop container %s\n", c.Red, c.Reset, container)
			return 1
		}
		fmt.Printf("  %sOK%s    stopped container %s (port %s released)\n", c.Green, c.Reset, container, port)
		return 0
	}

	pids := collectPIDsForPort(port)
	if len(pids) == 0 {
		ui.PrintDim(os.Stdout, fmt.Sprintf("No process listening on port %s (or you need elevated privileges).", port))
		return 0
	}

	if portClass(port, "-") == "system" {
		fmt.Println("")
		fmt.Printf("  %sWARN%s  Port %s is a system service port.\n", c.Yellow, c.Reset, port)
		fmt.Println("         Stopping it can break networking, printing, or other")
		fmt.Println("         system services.")
		fmt.Println("")
		if !confirmYes("Really continue? [y/N] ") {
			ui.PrintDim(os.Stdout, "Cancelled.")
			return 0
		}
	}

	fmt.Println("")
	fmt.Printf("Target: port %s\n", port)
	fmt.Printf("PIDs:   %s\n", strings.Join(pids, " "))
	fmt.Println("")
	if !confirmYes("Kill these processes? [y/N] ") {
		ui.PrintDim(os.Stdout, "Cancelled.")
		return 0
	}

	killed := killPIDs(pids)
	fmt.Printf("  %sOK%s    Killed %d process(es).\n", c.Green, c.Reset, killed)
	return 0
}

func cmdKillInteractive() int {
	c := ui.Colors()
	entries, err := collectPorts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %sERR%s   %v\n", c.Red, c.Reset, err)
		return 1
	}
	if len(entries) == 0 {
		ui.PrintDim(os.Stdout, "No listening ports found.")
		return 0
	}

	var sysList, dkrList, usrList []portEntry
	for _, e := range entries {
		switch portClass(e.Port, e.PID) {
		case "system":
			sysList = append(sysList, e)
		case "docker":
			dkrList = append(dkrList, e)
		default:
			usrList = append(usrList, e)
		}
	}
	ordered := append(append(sysList, dkrList...), usrList...)

	fmt.Println("Listening ports:")
	fmt.Println("")
	for i, e := range ordered {
		tag := "[USER]  "
		switch portClass(e.Port, e.PID) {
		case "system":
			tag = "[SYSTEM]"
		case "docker":
			tag = "[DOCKER]"
		}
		fmt.Printf("  %2d) %s %s\n", i+1, tag, formatEntry(e))
	}
	fmt.Println("")

	fmt.Print("Select (index/port, space or comma separated, q to quit): ")
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" || choice == "q" {
		return 0
	}

	tokens := strings.Fields(strings.ReplaceAll(choice, ",", " "))
	var dockerNames, pids, report, risky []string
	for _, tok := range tokens {
		var selected *portEntry
		if idx, err := strconv.Atoi(tok); err == nil && idx >= 1 && idx <= len(ordered) {
			selected = &ordered[idx-1]
		}
		if selected == nil {
			for i := range ordered {
				if ordered[i].Port == tok {
					selected = &ordered[i]
					break
				}
			}
		}
		if selected == nil {
			fmt.Printf("  %sWARN%s  Not found: %s\n", c.Yellow, c.Reset, tok)
			continue
		}

		if selected.PID == "docker" {
			dockerNames = append(dockerNames, selected.Command)
			report = append(report, fmt.Sprintf("port %s -> docker container %s", selected.Port, selected.Command))
		} else if selected.PID == "" || selected.PID == "-" {
			report = append(report, fmt.Sprintf("port %s -> no PID visible (elevated privileges may be needed)", selected.Port))
		} else {
			pids = append(pids, selected.PID)
			report = append(report, fmt.Sprintf("port %s -> PID %s (%s)", selected.Port, selected.PID, selected.Command))
		}
		if portClass(selected.Port, selected.PID) == "system" {
			risky = append(risky, selected.Port)
		}
	}

	if len(dockerNames) == 0 && len(pids) == 0 {
		ui.PrintDim(os.Stdout, "Nothing to kill.")
		return 0
	}

	fmt.Println("")
	fmt.Println("Targets:")
	for _, r := range report {
		fmt.Printf("  - %s\n", r)
	}

	if len(risky) > 0 {
		fmt.Println("")
		fmt.Printf("  %sWARN%s  You are about to stop system service port(s): %s\n", c.Yellow, c.Reset, strings.Join(risky, " "))
		fmt.Println("         This can break networking, printing, databases, or other")
		fmt.Println("         system services. Proceed with extreme caution.")
		fmt.Println("")
		if !confirmYes("Really continue? [y/N] ") {
			ui.PrintDim(os.Stdout, "Cancelled.")
			return 0
		}
	} else {
		fmt.Println("")
		if !confirmYes("Proceed? [y/N] ") {
			ui.PrintDim(os.Stdout, "Cancelled.")
			return 0
		}
	}

	fmt.Println("")
	if len(dockerNames) > 0 {
		cmdArgs := append([]string{"stop"}, dockerNames...)
		if err := exec.Command("docker", cmdArgs...).Run(); err == nil {
			fmt.Printf("  %sOK%s    stopped container(s): %s\n", c.Green, c.Reset, strings.Join(dockerNames, " "))
		} else {
			fmt.Fprintf(os.Stderr, "  %sERR%s   failed to stop container(s): %s\n", c.Red, c.Reset, strings.Join(dockerNames, " "))
		}
	}
	if len(pids) > 0 {
		killed := killPIDs(pids)
		fmt.Printf("  %sOK%s    killed %d process(es)\n", c.Green, c.Reset, killed)
	}
	return 0
}

func Run(args []string) int {
	if len(args) == 0 {
		printHelp()
		return 0
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "ls":
		return cmdList(rest)
	case "kill":
		if len(rest) > 0 {
			return cmdKillByPort(rest[0])
		}
		return cmdKillInteractive()
	case "docker", "ps":
		return cmdDocker()
	case "help", "--help", "-h", "":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "  %sERR%s   Unknown subcommand: %s\n", ui.Colors().Red, ui.Colors().Reset, sub)
		fmt.Println("Run: reta port help")
		return 1
	}
}
