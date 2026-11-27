package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/ui"
)

const DefaultSearchDepth = 4
const DefaultAgentsFile = "AGENTS.md"

var defaults = map[string]string{
	"agents-file":     DefaultAgentsFile,
	"agents-template": "",
	"hints":           "on",
	"learn":           "off",
	"search-depth":    strconv.Itoa(DefaultSearchDepth),
}

func FilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "reta", "config"), nil
}

func ExpandPath(p string) string {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return ""
	}
	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(trimmed, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			if trimmed == "~" {
				return home
			}
			return filepath.Clean(filepath.Join(home, trimmed[2:]))
		}
	}
	return filepath.Clean(trimmed)
}

func validateAgentsFileName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasSuffix(trimmed, "/") || strings.HasSuffix(trimmed, `\`) {
		return "", fmt.Errorf("must be a file name, not a directory")
	}
	if filepath.IsAbs(trimmed) || strings.HasPrefix(trimmed, "~") {
		return "", fmt.Errorf("must be a relative path inside the project")
	}
	slashed := filepath.ToSlash(trimmed)
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(slashed)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("must stay inside the project directory")
	}
	return clean, nil
}

func Load() map[string]string {
	cfg := make(map[string]string, len(defaults))
	for k, v := range defaults {
		cfg[k] = v
	}
	p, err := FilePath()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return cfg
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		k, v, _ := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if _, ok := defaults[k]; !ok {
			continue
		}
		if k == "search-depth" {
			if n, err := strconv.Atoi(v); err == nil && n >= 1 {
				cfg[k] = strconv.Itoa(n)
			}
			continue
		}
		if k == "agents-file" {
			if cleaned, err := validateAgentsFileName(v); err == nil {
				cfg[k] = cleaned
			}
			continue
		}
		cfg[k] = v
	}
	return cfg
}

func Save(cfg map[string]string) error {
	p, err := FilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(cfg))
	for k := range cfg {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("# reta settings — managed by 'reta config'\n")
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("%s=%s\n", k, cfg[k]))
	}
	return os.WriteFile(p, []byte(b.String()), 0o644)
}

func SearchDepth() int {
	cfg := Load()
	if n, err := strconv.Atoi(cfg["search-depth"]); err == nil && n >= 1 {
		return n
	}
	return DefaultSearchDepth
}

func AgentsFile() string {
	return Load()["agents-file"]
}

func AgentsTemplate() string {
	return Load()["agents-template"]
}

func printHelp() {
	fmt.Println("Usage: reta config [--reset] [key] [value]")
	fmt.Println("")
	fmt.Println("View and change persistent reta settings.")
	fmt.Println("")
	fmt.Println("Keys:")
	fmt.Println("  agents-file      File created by 'reta init' (default: AGENTS.md; \"\" disables)")
	fmt.Println("  agents-template  Local prompt file copied into agents-file (default: empty)")
	fmt.Println("  hints            on/off (show or hide Next steps)")
	fmt.Println("  learn            on/off (print underlying make commands)")
	fmt.Println("  search-depth     integer >= 1 (max depth for reta search)")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --reset          Reset all settings to defaults")
	fmt.Println("  -h, --help       Show this help message")
}

func Run(args []string) error {
	parser := cli.NewParser()
	var reset bool
	var showHelp bool
	parser.Bool(&reset, "reset", "")
	parser.Bool(&showHelp, "help", "h")

	pos, err := parser.Parse(args)
	if err != nil {
		return err
	}

	if showHelp {
		printHelp()
		return nil
	}

	p, err := FilePath()
	if err != nil {
		return cli.Wrap(1, err, "")
	}

	if reset {
		if _, err := os.Stat(p); err == nil {
			if err := os.Remove(p); err != nil {
				return cli.Wrap(1, err, "")
			}
			fmt.Printf("OK  reset %s\n", p)
		} else {
			fmt.Println("OK  already at defaults")
		}
		return nil
	}

	cfg := Load()
	c := ui.Colors()

	if len(pos) == 0 {
		fmt.Println("reta settings")
		fmt.Printf("  config: %s\n", p)
		fmt.Println("")
		keys := make([]string, 0, len(cfg))
		for k := range cfg {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			def := defaults[k]
			valStr := cfg[k]
			marker := ""
			if cfg[k] != def {
				marker = " (custom)"
			}
			extra := ""
			switch k {
			case "search-depth":
				extra = fmt.Sprintf("  %s(recommended: %s)%s", c.Dim, def, c.Reset)
			case "agents-file":
				if valStr == "" {
					valStr = fmt.Sprintf("%s(disabled)%s", c.Dim, c.Reset)
				}
				extra = fmt.Sprintf("  %s(default: %s; \"\" disables)%s", c.Dim, def, c.Reset)
			case "agents-template":
				if valStr == "" {
					valStr = fmt.Sprintf("%s(empty)%s", c.Dim, c.Reset)
				}
				extra = fmt.Sprintf("  %s(prompt file path; \"\" leaves blank)%s", c.Dim, c.Reset)
			}
			fmt.Printf("  %-17s %s%s%s\n", k, valStr, marker, extra)
		}
		fmt.Println("")
		fmt.Println("Change: reta config <key> <value>")
		fmt.Println("Clear:  reta config <key> \"\"")
		return nil
	}

	key := pos[0]
	if _, ok := defaults[key]; !ok {
		keys := make([]string, 0, len(defaults))
		for k := range defaults {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(os.Stderr, "Error: unknown setting '%s'.\n", key)
		fmt.Fprintf(os.Stderr, "Valid: %s\n", strings.Join(keys, ", "))
		return cli.Exit(1, "")
	}

	if len(pos) == 1 {
		switch key {
		case "search-depth":
			fmt.Printf("%s=%s  %s(recommended: %s)%s\n", key, cfg[key], c.Dim, defaults[key], c.Reset)
		case "agents-file":
			if cfg[key] == "" {
				fmt.Printf("%s=  %s(disabled; skipped on reta init)%s\n", key, c.Dim, c.Reset)
			} else {
				fmt.Printf("%s=%s\n", key, cfg[key])
			}
		case "agents-template":
			if cfg[key] == "" {
				fmt.Printf("%s=  %s(empty; creates blank file)%s\n", key, c.Dim, c.Reset)
			} else {
				fmt.Printf("%s=%s\n", key, cfg[key])
			}
		default:
			fmt.Printf("%s=%s\n", key, cfg[key])
		}
		return nil
	}

	val := pos[1]
	switch key {
	case "hints", "learn":
		if val != "on" && val != "off" {
			return cli.Exit(2, "invalid value '%s' for '%s' (on/off).", val, key)
		}
	case "search-depth":
		n, err := strconv.Atoi(val)
		if err != nil || n < 1 {
			return cli.Exit(2, "invalid value '%s' for 'search-depth' (integer >= 1).", val)
		}
		val = strconv.Itoa(n)
	case "agents-file":
		cleaned, err := validateAgentsFileName(val)
		if err != nil {
			return cli.Exit(2, "invalid value '%s' for 'agents-file' (%v).", val, err)
		}
		val = cleaned
	case "agents-template":
		trimmed := strings.TrimSpace(val)
		if trimmed == "" {
			val = ""
		} else {
			expanded := ExpandPath(trimmed)
			abs, err := filepath.Abs(expanded)
			if err != nil {
				return cli.Exit(2, "invalid path '%s' for 'agents-template': %v", val, err)
			}
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				return cli.Exit(2, "invalid value '%s' for 'agents-template' (path is a directory).", val)
			}
			val = abs
		}
	}

	cfg[key] = val
	if err := Save(cfg); err != nil {
		return cli.Wrap(1, err, "")
	}

	switch key {
	case "search-depth":
		fmt.Printf("OK  %s=%s  %s(recommended: %s)%s\n", key, val, c.Dim, defaults[key], c.Reset)
	case "agents-file":
		if val == "" {
			fmt.Printf("OK  %s=  %s(disabled)%s\n", key, c.Dim, c.Reset)
		} else {
			fmt.Printf("OK  %s=%s\n", key, val)
		}
	case "agents-template":
		if val == "" {
			fmt.Printf("OK  %s=  %s(empty)%s\n", key, c.Dim, c.Reset)
		} else {
			fmt.Printf("OK  %s=%s\n", key, val)
		}
	default:
		fmt.Printf("OK  %s=%s\n", key, val)
	}

	switch {
	case key == "hints" && val == "off":
		fmt.Println("Next steps hidden.")
	case key == "hints" && val == "on":
		fmt.Println("Next steps shown.")
	case key == "learn" && val == "on":
		fmt.Println("Learning mode enabled.")
	case key == "learn" && val == "off":
		fmt.Println("Learning mode disabled.")
	case key == "agents-file" && val == "":
		fmt.Println("Agent file creation disabled for 'reta init'.")
	case key == "agents-file":
		fmt.Printf("'reta init' will create %s.\n", val)
	case key == "agents-template" && val == "":
		fmt.Println("Agent template cleared; 'reta init' will create an empty file.")
	case key == "agents-template":
		if _, err := os.Stat(val); err != nil {
			fmt.Printf("%sWARN%s  File not found yet: %s ('reta init' will create an empty file until it exists).\n", c.Yellow, c.Reset, val)
		} else if cfg["agents-file"] == "" {
			fmt.Println("Note: 'agents-file' is currently disabled; set 'reta config agents-file AGENTS.md' to enable.")
		} else {
			fmt.Printf("'reta init' will copy this file into %s.\n", cfg["agents-file"])
		}
	}
	return nil
}
