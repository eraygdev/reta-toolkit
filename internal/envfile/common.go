package envfile

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/eraygdev/reta-toolkit/internal/ui"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

type parsedEnv struct {
	Order      []string
	Values     map[string]string
	RawValues  map[string]string
	Quoted     map[string]bool
	Duplicates []string
}

func generateSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func isStdinTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func readPasswordPrompt(prompt string) (string, error) {
	fmt.Print(prompt)
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(pwBytes), nil
}

func readLinePrompt(prompt string) string {
	fmt.Print(prompt)
	line, err := ui.ReadLine()
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimSpace(line)
}

func splitEnvValue(rawVal string) (val, cleanRaw string, isQuoted bool) {
	rawVal = strings.TrimSpace(rawVal)
	if rawVal == "" {
		return "", "", false
	}
	if q := rawVal[0]; q == '\'' || q == '"' {
		for i := 1; i < len(rawVal); i++ {
			if q == '"' && rawVal[i] == '\\' && i+1 < len(rawVal) {
				i++
				continue
			}
			if rawVal[i] == q {
				after := strings.TrimSpace(rawVal[i+1:])
				if after == "" || strings.HasPrefix(after, "#") {
					return rawVal[1:i], rawVal[:i+1], true
				}
				break
			}
		}
	}

	for i := 0; i < len(rawVal); i++ {
		if rawVal[i] == '#' && (i == 0 || rawVal[i-1] == ' ' || rawVal[i-1] == '\t') {
			clean := strings.TrimSpace(rawVal[:i])
			return clean, clean, false
		}
	}
	return rawVal, rawVal, false
}

func parseEnvFile(path string) (parsedEnv, error) {
	out := parsedEnv{
		Values:    make(map[string]string),
		RawValues: make(map[string]string),
		Quoted:    make(map[string]bool),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	content := strings.TrimPrefix(string(data), "\xef\xbb\xbf")
	seenDup := make(map[string]bool)
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimLeft(line[7:], " \t")
		}
		if !strings.Contains(line, "=") {
			continue
		}
		key, rawVal, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		val, cleanRaw, isQuoted := splitEnvValue(rawVal)
		if _, exists := out.Values[key]; exists {
			if !seenDup[key] {
				out.Duplicates = append(out.Duplicates, key)
				seenDup[key] = true
			}
		} else {
			out.Order = append(out.Order, key)
		}
		out.Values[key] = val
		out.RawValues[key] = cleanRaw
		out.Quoted[key] = isQuoted
	}
	return out, nil
}
