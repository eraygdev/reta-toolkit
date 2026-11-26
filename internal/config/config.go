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

var defaults = map[string]string{
	"hints":        "on",
	"learn":        "off",
	"search-depth": strconv.Itoa(DefaultSearchDepth),
}

func FilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "reta", "config"), nil
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

func printHelp() {
	fmt.Println("Usage: reta config [--reset] [key] [value]")
	fmt.Println("")
	fmt.Println("View and change persistent reta settings.")
	fmt.Println("")
	fmt.Println("Keys:")
	fmt.Println("  hints         on/off (show or hide Next steps)")
	fmt.Println("  learn         on/off (print underlying make commands)")
	fmt.Println("  search-depth  integer >= 1 (max depth for reta search)")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --reset       Reset all settings to defaults")
	fmt.Println("  -h, --help    Show this help message")
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
			marker := ""
			if cfg[k] != def {
				marker = " (custom)"
			}
			extra := ""
			if k == "search-depth" {
				extra = fmt.Sprintf("  %s(recommended: %s)%s", c.Dim, def, c.Reset)
			}
			fmt.Printf("  %-14s %s%s%s\n", k, cfg[k], marker, extra)
		}
		fmt.Println("")
		fmt.Println("Change: reta config <key> <value>")
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
		if key == "search-depth" {
			fmt.Printf("%s=%s  %s(recommended: %s)%s\n", key, cfg[key], c.Dim, defaults[key], c.Reset)
		} else {
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
	}

	cfg[key] = val
	if err := Save(cfg); err != nil {
		return cli.Wrap(1, err, "")
	}

	if key == "search-depth" {
		fmt.Printf("OK  %s=%s  %s(recommended: %s)%s\n", key, val, c.Dim, defaults[key], c.Reset)
	} else {
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
	}
	return nil
}
