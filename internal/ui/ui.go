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
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		k, v, _ := strings.Cut(line, "=")
		if strings.TrimSpace(k) == key {
			val = strings.TrimSpace(v)
		}
	}
	return val
}

func HintsEnabled() bool {
	switch strings.ToLower(os.Getenv("RETA_HINTS")) {
	case "off", "false", "0", "no":
		return false
	case "on", "true", "1", "yes":
		return true
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

// ReadLine reads a single line from stdin without buffering past the newline.
func ReadLine() (string, error) {
	var buf [1]byte
	var line []byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if n > 0 {
			if buf[0] == '\n' {
				return strings.TrimSuffix(string(line), "\r"), nil
			}
			line = append(line, buf[0])
		}
		if err != nil {
			if len(line) > 0 {
				return strings.TrimSuffix(string(line), "\r"), nil
			}
			return "", err
		}
	}
}

func ConfirmYes(w io.Writer, prompt string) bool {
	if prompt != "" {
		fmt.Fprint(w, prompt)
	}
	line, err := ReadLine()
	if err != nil && line == "" {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}

func NotInstalled() {
	fmt.Fprintln(os.Stderr, "❌ [reta] reta toolkit is not installed.")
	fmt.Fprintln(os.Stderr, "Install: git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta && ~/.reta/install.sh")
}
