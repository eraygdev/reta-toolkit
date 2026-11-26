package secrets

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func enterTempRepo(t *testing.T) string {
	t.Helper()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NO_COLOR", "1")
	t.Setenv("RETA_HINTS", "off")
	return tmp
}

func TestIsSensitivePathAndBinaryDetection(t *testing.T) {
	sensitive := []string{".env", ".env.local", ".env.production", "server.pem", "priv.key", "id_rsa"}
	for _, p := range sensitive {
		if !isSensitivePath(p) {
			t.Fatalf("expected %q to be sensitive", p)
		}
	}
	safe := []string{".env.example", ".env.sample", "project_export.md", "main.go"}
	for _, p := range safe {
		if isSensitivePath(p) {
			t.Fatalf("expected %q not to be sensitive", p)
		}
	}

	if !isBinaryBytes([]byte{'a', 'b', 0, 'c'}) {
		t.Fatal("expected NUL byte to mark binary data")
	}
	if isBinaryBytes([]byte("plain utf8 text\n")) {
		t.Fatal("expected plain text not to be binary")
	}
}

func TestScanTextAndIgnoreMarker(t *testing.T) {
	openai := "sk-proj-" + "abcdefghijklmnopqrstuvwxyz1234"
	aws := "AKIA" + "IOSFODNN7EXAMPLE"
	gemini := "AIza" + "SyDummyKeyForExampleTemplate1234567"

	src := "KEY1 = \"" + openai + "\"\r\n" +
		"KEY2 = \"" + aws + "\" # check-secrets: ignore\r\n" +
		"KEY3 = \"" + gemini + "\"\r\n"

	findings := scanText("app.py", src)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %+v", findings)
	}
	if findings[0].Line != 1 || findings[1].Line != 3 {
		t.Fatalf("unexpected line numbers: %+v", findings)
	}
}

func TestRunStagedFromSubdirectory(t *testing.T) {
	testutil.SilenceStdio(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := enterTempRepo(t)
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
	} {
		if err := exec.Command("git", args...).Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	if err := os.WriteFile(".gitignore", []byte(".env\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("sub", 0o755); err != nil {
		t.Fatal(err)
	}
	secretLine := "TOKEN=\"" + "ghp_" + "1234567890abcdef1234567890abcdef1234" + "\"\n"
	if err := os.WriteFile(filepath.Join("sub", "cfg.py"), []byte(secretLine), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "add", ".gitignore", "sub/cfg.py").Run(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(filepath.Join(root, "sub")); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"--staged"}); code != 1 {
		t.Fatalf("expected --staged to detect secret from subdirectory, got %d", code)
	}
}
