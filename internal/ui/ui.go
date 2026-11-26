package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

type Palette struct {
	Bold    string
	Dim     string
	Cyan    string
	Green   string
	Yellow  string
	Magenta string
	Red     string
	Reset   string
}

func useColor() bool {
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func Colors() Palette {
	if !useColor() {
		return Palette{}
	}
	return Palette{
		Bold:    "\033[1m",
		Dim:     "\033[2m",
		Cyan:    "\033[36m",
		Green:   "\033[32m",
		Yellow:  "\033[1;33m",
		Magenta: "\033[1;35m",
		Red:     "\033[31m",
		Reset:   "\033[0m",
	}
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "reta", "config")
}

func readConfigValue(key string) string {
	p := configPath()
	if p == "" {
		return ""
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	val := ""
	prefix := key + "="
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, prefix) {
			val = strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return val
}

func HintsEnabled() bool {
	switch strings.ToLower(os.Getenv("RETA_HINTS")) {
	case "off", "false", "0", "no":
		return false
	}
	return readConfigValue("hints") != "off"
}

func LearnEnabled() bool {
	switch strings.ToLower(os.Getenv("RETA_LEARN")) {
	case "off", "false", "0", "no":
		return false
	case "on", "true", "1", "yes":
		return true
	}
	return readConfigValue("learn") == "on"
}

func PrintDim(w io.Writer, text string) {
	c := Colors()
	fmt.Fprintf(w, "  %s%s%s\n", c.Dim, text, c.Reset)
}

func NextSteps(steps []string) {
	if len(steps) == 0 || !HintsEnabled() {
		return
	}
	c := Colors()
	fmt.Printf("\n%sNext steps:%s\n", c.Bold, c.Reset)
	for _, step := range steps {
		fmt.Printf("  → %s\n", step)
	}
}

func NotInstalled() {
	fmt.Fprintln(os.Stderr, "❌ [reta] reta toolkit is not installed.")
	fmt.Fprintln(os.Stderr, "Install: git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta && ~/.reta/install.sh")
}
