package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SilenceStdio redirects os.Stdout and os.Stderr to a temporary file during a
// test and logs the captured output only if the test fails.
func SilenceStdio(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "stdio.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("failed to create stdio capture file: %v", err)
	}

	origOut := os.Stdout
	origErr := os.Stderr
	os.Stdout = f
	os.Stderr = f

	t.Cleanup(func() {
		os.Stdout = origOut
		os.Stderr = origErr
		_ = f.Close()
		if t.Failed() {
			if data, rErr := os.ReadFile(logPath); rErr == nil {
				if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
					t.Logf("captured stdout/stderr:\n%s", trimmed)
				}
			}
		}
	})
}
