package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func TestExpectedAssetName(t *testing.T) {
	got := expectedAssetName()
	wantPrefix := "reta-" + runtime.GOOS + "-" + runtime.GOARCH
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("expectedAssetName() = %q, want prefix %q", got, wantPrefix)
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(got, ".exe") {
		t.Fatalf("expected .exe suffix on Windows, got %q", got)
	}
	if runtime.GOOS != "windows" && strings.HasSuffix(got, ".exe") {
		t.Fatalf("did not expect .exe suffix on %s, got %q", runtime.GOOS, got)
	}
}

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()
	exePath := filepath.Join(dir, "reta")
	if runtime.GOOS == "windows" {
		exePath += ".exe"
	}
	if err := os.WriteFile(exePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(exePath, []byte("new-binary")); err != nil {
		t.Fatalf("replaceBinary failed: %v", err)
	}
	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-binary" {
		t.Fatalf("binary content = %q, want new-binary", string(got))
	}
	if _, err := os.Stat(exePath + ".new"); !os.IsNotExist(err) {
		t.Fatal("expected temporary .new file to be removed")
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

func TestRunUpdateAndUninstallFlags(t *testing.T) {
	testutil.SilenceStdio(t)
	assertExitCode(t, RunUpdate([]string{"--help"}), 0)
	assertExitCode(t, RunUpdate([]string{"--bad-flag"}), 2)
	assertExitCode(t, RunUninstall([]string{"--help"}), 0)
	assertExitCode(t, RunUninstall([]string{"--bad-flag"}), 2)
}

func setupFakeUninstallEnv(t *testing.T) (home, retaRoot, binPath string) {
	t.Helper()
	home = t.TempDir()
	localAppData := filepath.Join(home, "AppData", "Local")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", localAppData)
	t.Setenv("NO_COLOR", "1")

	retaRoot = filepath.Join(home, ".reta")
	if runtime.GOOS == "windows" {
		binPath = filepath.Join(localAppData, "reta", "bin", "reta.exe")
	} else {
		binPath = filepath.Join(home, ".local", "bin", "reta")
	}
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	return home, retaRoot, binPath
}

func TestRunUninstallKeepAndFull(t *testing.T) {
	testutil.SilenceStdio(t)
	home, retaRoot, binPath := setupFakeUninstallEnv(t)
	if err := os.MkdirAll(filepath.Join(retaRoot, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(retaRoot, "lib", "base.mk"), []byte("help:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertExitCode(t, RunUninstall([]string{"--keep"}), 0)
	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Fatal("expected binary to be removed with --keep")
	}
	if _, err := os.Stat(retaRoot); err != nil {
		t.Fatalf("expected ~/.reta preserved with --keep: %v", err)
	}

	cacheDir := filepath.Join(home, ".cache", "reta")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	assertExitCode(t, RunUninstall([]string{"-y"}), 0)
	if _, err := os.Stat(retaRoot); !os.IsNotExist(err) {
		t.Fatal("expected ~/.reta removed with -y")
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatal("expected ~/.cache/reta removed")
	}
}

func TestRunUninstallSafetyAndSymlink(t *testing.T) {
	testutil.SilenceStdio(t)
	_, retaRoot, _ := setupFakeUninstallEnv(t)
	if err := os.MkdirAll(retaRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	assertExitCode(t, RunUninstall([]string{"-y"}), 1)
	if _, err := os.Stat(retaRoot); err != nil {
		t.Fatal("expected invalid ~/.reta not to be deleted")
	}
	_ = os.RemoveAll(retaRoot)

	realSrc := t.TempDir()
	marker := filepath.Join(realSrc, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realSrc, retaRoot); err != nil {
		t.Skipf("symlink creation not permitted: %v", err)
	}

	assertExitCode(t, RunUninstall([]string{"-y"}), 0)
	if _, err := os.Lstat(retaRoot); !os.IsNotExist(err) {
		t.Fatal("expected symlink ~/.reta to be removed")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("expected symlink target contents to remain intact")
	}
}
