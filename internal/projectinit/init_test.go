package projectinit

import (
	"errors"
	"os"
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

func TestInitScaffoldsAndConverts(t *testing.T) {
	testutil.SilenceStdio(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("NO_COLOR", "1")
	t.Setenv("RETA_HINTS", "off")

	proj := filepath.Join(t.TempDir(), "fresh")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	assertExitCode(t, Run([]string{proj}), 0)

	for _, f := range []string{"Makefile", ".env.example", ".gitignore", "README.md", ".retasign"} {
		if _, err := os.Stat(filepath.Join(proj, f)); err != nil {
			t.Fatalf("missing scaffolded file %s: %v", f, err)
		}
	}

	// Existing Makefile is preserved without --convert in non-TTY tests
	existing := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	mkPath := filepath.Join(existing, "Makefile")
	if err := os.WriteFile(mkPath, []byte("custom:\n\techo hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertExitCode(t, Run([]string{existing}), 0)

	got, _ := os.ReadFile(mkPath)
	if string(got) != "custom:\n\techo hi\n" {
		t.Fatalf("expected Makefile untouched without --convert, got:\n%s", string(got))
	}

	// --convert adopts Makefile and keeps a backup
	assertExitCode(t, Run([]string{existing, "--convert"}), 0)
	if _, err := os.Stat(filepath.Join(existing, "Makefile.bak")); err != nil {
		t.Fatalf("expected Makefile.bak: %v", err)
	}
	conv, _ := os.ReadFile(mkPath)
	if !strings.Contains(string(conv), "include $(RETA_BASE)") {
		t.Fatalf("expected converted Makefile to include RETA_BASE:\n%s", string(conv))
	}
}
