package hook

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func assertExitCode(t *testing.T, err error, want int) {
	t.Helper()
	if want == 0 {
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		return
	}
	var exitErr *cli.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != want {
		t.Fatalf("expected exit code %d, got %v", want, err)
	}
}

func TestHookInstallStatusRemoveFromSubdir(t *testing.T) {
	testutil.SilenceStdio(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	t.Setenv("NO_COLOR", "1")
	t.Setenv("RETA_HINTS", "off")

	// Non-git directory must fail with code 1
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	assertExitCode(t, Run([]string{"install"}), 1)

	if err := exec.Command("git", "init", "-q").Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	assertExitCode(t, Run([]string{"install"}), 0)
	hookFile := filepath.Join(repo, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookFile); err != nil {
		t.Fatalf("expected hook at %s: %v", hookFile, err)
	}
	assertExitCode(t, Run([]string{"status"}), 0)
	data, err := os.ReadFile(hookFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "check-secrets --staged") {
		t.Fatalf("hook script must invoke 'check-secrets --staged':\n%s", string(data))
	}

	assertExitCode(t, Run([]string{"remove"}), 0)
	if _, err := os.Stat(hookFile); !os.IsNotExist(err) {
		t.Fatal("expected pre-commit hook to be removed")
	}
}
