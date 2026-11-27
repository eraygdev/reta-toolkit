package projectinit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/config"
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

	for _, f := range []string{"Makefile", ".env.example", ".gitignore", "README.md", ".retasign", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(proj, f)); err != nil {
			t.Fatalf("missing scaffolded file %s: %v", f, err)
		}
	}
	agentsRaw, err := os.ReadFile(filepath.Join(proj, "AGENTS.md"))
	if err != nil || len(agentsRaw) != 0 {
		t.Fatalf("expected empty default AGENTS.md, got len=%d err=%v", len(agentsRaw), err)
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

func TestInitAgentsFileRenameDisableAndTemplate(t *testing.T) {
	testutil.SilenceStdio(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("NO_COLOR", "1")
	t.Setenv("RETA_HINTS", "off")

	promptPath := filepath.Join(t.TempDir(), "project.md")
	wantBody := "# Agent Rules\nFollow project conventions.\n"
	if err := os.WriteFile(promptPath, []byte(wantBody), 0o644); err != nil {
		t.Fatal(err)
	}

	assertExitCode(t, config.Run([]string{"agents-file", "CLAUDE.md"}), 0)
	assertExitCode(t, config.Run([]string{"agents-template", promptPath}), 0)

	projCustom := filepath.Join(t.TempDir(), "custom")
	if err := os.MkdirAll(projCustom, 0o755); err != nil {
		t.Fatal(err)
	}
	assertExitCode(t, Run([]string{projCustom}), 0)

	gotBody, err := os.ReadFile(filepath.Join(projCustom, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("expected CLAUDE.md created: %v", err)
	}
	if string(gotBody) != wantBody {
		t.Fatalf("CLAUDE.md = %q, want %q", string(gotBody), wantBody)
	}
	if _, err := os.Stat(filepath.Join(projCustom, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("did not expect AGENTS.md when renamed to CLAUDE.md")
	}

	// Clearing template creates an empty file
	assertExitCode(t, config.Run([]string{"agents-template", ""}), 0)
	projBlank := filepath.Join(t.TempDir(), "blank")
	if err := os.MkdirAll(projBlank, 0o755); err != nil {
		t.Fatal(err)
	}
	assertExitCode(t, Run([]string{projBlank}), 0)
	blankBody, err := os.ReadFile(filepath.Join(projBlank, "CLAUDE.md"))
	if err != nil || len(blankBody) != 0 {
		t.Fatalf("expected empty CLAUDE.md after clearing template, got len=%d err=%v", len(blankBody), err)
	}

	// Empty agents-file disables file creation
	assertExitCode(t, config.Run([]string{"agents-file", ""}), 0)
	projDisabled := filepath.Join(t.TempDir(), "disabled")
	if err := os.MkdirAll(projDisabled, 0o755); err != nil {
		t.Fatal(err)
	}
	assertExitCode(t, Run([]string{projDisabled}), 0)
	for _, f := range []string{"AGENTS.md", "CLAUDE.md"} {
		if _, err := os.Stat(filepath.Join(projDisabled, f)); !os.IsNotExist(err) {
			t.Fatalf("did not expect %s when agents-file is empty", f)
		}
	}
}
