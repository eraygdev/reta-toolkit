package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	reta "github.com/eraygdev/reta-toolkit"
	"github.com/eraygdev/reta-toolkit/internal/adopt"
	"github.com/eraygdev/reta-toolkit/internal/clean"
	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/config"
	"github.com/eraygdev/reta-toolkit/internal/doctor"
	"github.com/eraygdev/reta-toolkit/internal/envfile"
	"github.com/eraygdev/reta-toolkit/internal/exporter"
	"github.com/eraygdev/reta-toolkit/internal/hook"
	"github.com/eraygdev/reta-toolkit/internal/lifecycle"
	"github.com/eraygdev/reta-toolkit/internal/port"
	"github.com/eraygdev/reta-toolkit/internal/projectinit"
	"github.com/eraygdev/reta-toolkit/internal/search"
	"github.com/eraygdev/reta-toolkit/internal/secrets"
	"github.com/eraygdev/reta-toolkit/internal/shell"
	"github.com/eraygdev/reta-toolkit/internal/ui"
)

type command struct {
	Name  string
	Desc  string
	Group string
	Order int
	Run   func(args []string) error
}

var commands = []command{
	{
		Name:  "init",
		Desc:  "Copy reta project templates into the current directory.",
		Group: "Setup",
		Order: 10,
		Run:   projectinit.Run,
	},
	{
		Name:  "init-env",
		Desc:  "Generate .env from .env.example. Resolves GENERATE_SECRET and AUTO_HASH markers.",
		Group: "Setup",
		Order: 20,
		Run:   envfile.RunInitEnv,
	},
	{
		Name:  "env",
		Desc:  "Inspect .env against .env.example: missing keys, masked values, key diff.",
		Group: "Setup",
		Order: 30,
		Run:   envfile.RunEnv,
	},
	{
		Name:  "config",
		Desc:  "View and change persistent reta settings.",
		Group: "Setup",
		Order: 40,
		Run:   config.Run,
	},
	{
		Name:  "hook",
		Desc:  "Install or remove pre-commit hooks for secret scanning.",
		Group: "Setup",
		Order: 55,
		Run:   hook.Run,
	},
	{
		Name:  "update",
		Desc:  "Pull latest reta changes or update standalone binary.",
		Group: "Setup",
		Order: 90,
		Run:   lifecycle.RunUpdate,
	},
	{
		Name:  "uninstall",
		Desc:  "Uninstall reta: remove binary/symlinks and optionally ~/.reta.",
		Group: "Setup",
		Order: 999,
		Run:   lifecycle.RunUninstall,
	},
	{
		Name:  "check-secrets",
		Desc:  "Scan a project for tracked .env files and hardcoded secrets.",
		Group: "Utilities",
		Order: 30,
		Run:   secrets.Run,
	},
	{
		Name:  "clean",
		Desc:  "Safe system cleanup with before/after disk and RAM summary.",
		Group: "Utilities",
		Order: 10,
		Run:   clean.Run,
	},
	{
		Name:  "search",
		Desc:  "Search and jump across all reta projects (use 'rcd' shortcut to jump).",
		Group: "Utilities",
		Order: 25,
		Run:   search.Run,
	},
	{
		Name:  "port",
		Desc:  "List listening ports, or kill a process on a specific port.",
		Group: "Utilities",
		Order: 50,
		Run:   port.Run,
	},
	{
		Name:  "completion",
		Desc:  "Generate shell completion script (bash, zsh, powershell).",
		Group: "Utilities",
		Order: 60,
		Run:   shell.RunCompletion,
	},
	{
		Name:  "shell-init",
		Desc:  "Print shell integration code for 'rcd' project jumping.",
		Group: "Utilities",
		Order: 65,
		Run:   shell.RunShellInit,
	},
	{
		Name:  "doctor",
		Desc:  "Check reta installation health.",
		Group: "Diagnostics",
		Order: 10,
		Run:   doctor.Run,
	},
	{
		Name:  "export",
		Desc:  "Export project source files into markdown for AI context.",
		Group: "Utilities",
		Order: 35,
		Run:   exporter.Run,
	},
	{
		Name:  "adopt-makefile",
		Desc:  "Convert an existing Makefile to use ~/.reta/lib/base.mk.",
		Group: "Utilities",
		Order: 40,
		Run:   adopt.Run,
	},
}

func findCommand(name string) *command {
	for i := range commands {
		if commands[i].Name == name {
			return &commands[i]
		}
	}
	return nil
}

func printMainHelp(version string) {
	c := ui.Colors()
	fmt.Println("")
	fmt.Printf("  %s%sreta%s %sv%s%s — personal dev toolkit\n", c.Magenta, c.Bold, c.Reset, c.Dim, version, c.Reset)
	fmt.Printf("  %s────────────────────────────────────────%s\n", c.Dim, c.Reset)
	fmt.Println("")
	fmt.Printf("  %sUsage:%s %sreta%s <command> [args...]\n", c.Dim, c.Reset, c.Bold, c.Reset)
	fmt.Printf("         %sreta%s --version\n", c.Bold, c.Reset)

	groupOrder := map[string]int{}
	idx := 0
	sorted := make([]command, len(commands))
	copy(sorted, commands)
	for _, cmd := range sorted {
		if _, ok := groupOrder[cmd.Group]; !ok {
			idx++
			groupOrder[cmd.Group] = idx
		}
	}

	sort.Slice(sorted, func(i, j int) bool {
		gi, gj := groupOrder[sorted[i].Group], groupOrder[sorted[j].Group]
		if gi != gj {
			return gi < gj
		}
		if sorted[i].Order != sorted[j].Order {
			return sorted[i].Order < sorted[j].Order
		}
		return sorted[i].Name < sorted[j].Name
	})

	prevGroup := ""
	for _, cmd := range sorted {
		if cmd.Group != prevGroup {
			fmt.Println("")
			fmt.Printf("  %s%s%s\n", c.Yellow, cmd.Group, c.Reset)
			prevGroup = cmd.Group
		}
		fmt.Printf("  %s%-15s%s %s\n", c.Cyan, cmd.Name, c.Reset, cmd.Desc)
	}

	fmt.Println("")
	fmt.Printf("  %sRun 'reta <command>' to execute.%s\n", c.Dim, c.Reset)
	fmt.Println("")
}

func handleExit(err error) {
	if err == nil {
		os.Exit(0)
	}
	var exitErr *cli.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.Message != "" {
			c := ui.Colors()
			fmt.Fprintf(os.Stderr, "%sError:%s %s\n", c.Red, c.Reset, exitErr.Message)
		}
		os.Exit(exitErr.Code)
	}
	c := ui.Colors()
	fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
	os.Exit(1)
}

func main() {
	version := strings.TrimSpace(reta.VersionRaw)
	if version == "" {
		version = "unknown"
	}

	args := os.Args[1:]
	if len(args) == 0 {
		printMainHelp(version)
		os.Exit(0)
	}

	cmdName := args[0]
	rest := args[1:]

	if cmdName == "help" && len(rest) > 0 && rest[0] != "" {
		if cmd := findCommand(rest[0]); cmd != nil {
			handleExit(cmd.Run([]string{"--help"}))
		}
		fmt.Printf("reta: unknown command '%s'\n", rest[0])
		fmt.Println("Try: reta help")
		os.Exit(1)
	}

	switch cmdName {
	case "help", "--help", "-h", "":
		printMainHelp(version)
		os.Exit(0)
	case "--version", "-v":
		fmt.Println(version)
		os.Exit(0)
	}

	if cmdName == "export-context" {
		handleExit(exporter.Run(rest))
	}

	if cmd := findCommand(cmdName); cmd != nil {
		handleExit(cmd.Run(rest))
	}

	fmt.Printf("reta: unknown command '%s'\n", cmdName)
	fmt.Println("Try: reta help")
	os.Exit(1)
}
