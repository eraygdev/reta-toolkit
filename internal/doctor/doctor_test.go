package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	reta "github.com/eraygdev/reta-toolkit"
	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func TestIsOnPath(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "other")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+other)

	if !isOnPath(dir) {
		t.Fatalf("expected %s to be detected on PATH", dir)
	}
	if isOnPath(filepath.Join(dir, "missing")) {
		t.Fatal("expected missing dir not to be on PATH")
	}
	if runtime.GOOS == "windows" && !isOnPath(strings.ToUpper(dir)) {
		t.Fatal("expected Windows PATH check to be case-insensitive")
	}
}

func TestEnsureBaseMK(t *testing.T) {
	plainRoot := t.TempDir()
	if err := EnsureBaseMK(plainRoot); err != nil {
		t.Fatalf("EnsureBaseMK(plain) failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(plainRoot, "lib", "base.mk"))
	if err != nil {
		t.Fatalf("expected lib/base.mk created: %v", err)
	}
	if string(got) != reta.BaseMK {
		t.Fatal("written base.mk does not match embedded BaseMK")
	}

	gitRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gitRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gitRoot, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	customPath := filepath.Join(gitRoot, "lib", "base.mk")
	if err := os.WriteFile(customPath, []byte("custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureBaseMK(gitRoot); err != nil {
		t.Fatalf("EnsureBaseMK(git) failed: %v", err)
	}
	after, _ := os.ReadFile(customPath)
	if string(after) != "custom\n" {
		t.Fatalf("EnsureBaseMK must not overwrite git-managed base.mk, got %q", string(after))
	}
}

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

func TestDoctorRunFlagsAndFix(t *testing.T) {
	testutil.SilenceStdio(t)
	t.Setenv("NO_COLOR", "1")

	assertExitCode(t, Run([]string{"--help"}), 0)
	assertExitCode(t, Run([]string{"--unknown"}), 2)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	assertExitCode(t, Run(nil), 1)

	_ = Run([]string{"--fix"})
	if _, err := os.Stat(filepath.Join(home, ".reta", "lib", "base.mk")); err != nil {
		t.Fatalf("expected --fix to create lib/base.mk: %v", err)
	}
}
