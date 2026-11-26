package doctor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	reta "github.com/eraygdev/reta-toolkit"
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

func TestDoctorRunFlagsAndFix(t *testing.T) {
	testutil.SilenceStdio(t)
	t.Setenv("NO_COLOR", "1")
	if code := Run([]string{"--help"}); code != 0 {
		t.Fatalf("Run(--help) = %d, want 0", code)
	}
	if code := Run([]string{"--unknown"}); code != 2 {
		t.Fatalf("Run(--unknown) = %d, want 2", code)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if code := Run(nil); code != 1 {
		t.Fatalf("expected Run() on empty HOME to fail with 1, got %d", code)
	}

	_ = Run([]string{"--fix"})
	if _, err := os.Stat(filepath.Join(home, ".reta", "lib", "base.mk")); err != nil {
		t.Fatalf("expected --fix to create lib/base.mk: %v", err)
	}
}
