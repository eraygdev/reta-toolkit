package envfile

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

type parsedEnv struct {
	Order      []string
	Values     map[string]string
	RawValues  map[string]string
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
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimSpace(line)
}

func parseEnvFile(path string) (parsedEnv, error) {
	out := parsedEnv{
		Values:    make(map[string]string),
		RawValues: make(map[string]string),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	seenDup := make(map[string]bool)
	for _, raw := range strings.Split(string(data), "\n") {
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
		rawVal = strings.TrimSpace(rawVal)
		val := rawVal
		if len(val) >= 2 && val[0] == val[len(val)-1] && (val[0] == '\'' || val[0] == '"') {
			val = val[1 : len(val)-1]
		}
		if _, exists := out.Values[key]; exists {
			if !seenDup[key] {
				out.Duplicates = append(out.Duplicates, key)
				seenDup[key] = true
			}
		} else {
			out.Order = append(out.Order, key)
		}
		out.Values[key] = val
		out.RawValues[key] = rawVal
	}
	return out, nil
}
