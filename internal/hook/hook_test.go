package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

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

	// Non-git directory must fail
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"install"}); code != 1 {
		t.Fatalf("expected install outside git repo to fail with 1, got %d", code)
	}

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

	if code := Run([]string{"install"}); code != 0 {
		t.Fatalf("hook install from subdir = %d", code)
	}
	hookFile := filepath.Join(repo, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookFile); err != nil {
		t.Fatalf("expected hook at %s: %v", hookFile, err)
	}
	if code := Run([]string{"status"}); code != 0 {
		t.Fatalf("hook status = %d", code)
	}
	data, err := os.ReadFile(hookFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "check-secrets --staged") {
		t.Fatalf("hook script must invoke 'check-secrets --staged':\n%s", string(data))
	}

	if code := Run([]string{"remove"}); code != 0 {
		t.Fatalf("hook remove = %d", code)
	}
	if _, err := os.Stat(hookFile); !os.IsNotExist(err) {
		t.Fatal("expected pre-commit hook to be removed")
	}
}
