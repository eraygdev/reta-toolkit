package retasign

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const SignFilename = ".retasign"

var signPattern = regexp.MustCompile(`^RETA\*rid([0-9]+)r$`)

func GenerateSignature() (string, error) {
	var b strings.Builder
	b.Grow(12)
	ten := big.NewInt(10)
	for i := 0; i < 12; i++ {
		n, err := rand.Int(rand.Reader, ten)
		if err != nil {
			return "", err
		}
		b.WriteByte('0' + byte(n.Int64()))
	}
	return fmt.Sprintf("RETA*rid%sr", b.String()), nil
}

func ReadSignatureID(projectDir string) string {
	signFile := filepath.Join(projectDir, SignFilename)
	info, err := os.Stat(signFile)
	if err != nil || info.IsDir() {
		return ""
	}
	data, err := os.ReadFile(signFile)
	if err != nil {
		return ""
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if m := signPattern.FindStringSubmatch(line); len(m) == 2 {
			return "rid" + m[1] + "r"
		}
	}
	return ""
}

func WriteSignature(projectDir string) (string, error) {
	sig, err := GenerateSignature()
	if err != nil {
		return "", err
	}
	signFile := filepath.Join(projectDir, SignFilename)
	if err := os.WriteFile(signFile, []byte(sig+"\n"), 0o644); err != nil {
		return "", err
	}
	m := signPattern.FindStringSubmatch(sig)
	return "rid" + m[1] + "r", nil
}
