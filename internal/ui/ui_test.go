package ui

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestColorsRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	p := Colors()
	if p.Bold != "" || p.Red != "" || p.Reset != "" {
		t.Fatalf("expected empty palette when NO_COLOR is set, got %+v", p)
	}
}

func TestHintsAndLearnConfigAndEnvOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("RETA_HINTS", "")
	t.Setenv("RETA_LEARN", "")

	if !HintsEnabled() {
		t.Fatal("expected hints enabled by default")
	}
	if LearnEnabled() {
		t.Fatal("expected learn disabled by default")
	}

	cfgPath := filepath.Join(home, ".config", "reta", "config")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("hints=off\r\nlearn=on\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if HintsEnabled() {
		t.Fatal("expected hints=off from config")
	}
	if !LearnEnabled() {
		t.Fatal("expected learn=on from config")
	}

	t.Setenv("RETA_LEARN", "off")
	if LearnEnabled() {
		t.Fatal("expected RETA_LEARN=off to override config")
	}

	t.Setenv("RETA_HINTS", "0")
	if HintsEnabled() {
		t.Fatal("expected RETA_HINTS=0 to disable hints")
	}
}

func TestNextStepsOutputAndSuppression(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("NO_COLOR", "1")
	t.Setenv("RETA_HINTS", "")

	out := captureStdout(t, func() {
		NextSteps([]string{"make help", "reta doctor"})
	})
	if !strings.Contains(out, "Next steps:") || !strings.Contains(out, "make help") {
		t.Fatalf("unexpected NextSteps output:\n%s", out)
	}

	t.Setenv("RETA_HINTS", "off")
	suppressed := captureStdout(t, func() {
		NextSteps([]string{"make help"})
	})
	if suppressed != "" {
		t.Fatalf("expected empty output when hints disabled, got %q", suppressed)
	}
}
