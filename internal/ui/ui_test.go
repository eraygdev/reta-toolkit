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

	t.Setenv("RETA_HINTS", "on")
	if !HintsEnabled() {
		t.Fatal("expected RETA_HINTS=on to override hints=off in config")
	}

	t.Setenv("RETA_HINTS", "")
	t.Setenv("RETA_LEARN", "")
	if err := os.WriteFile(cfgPath, []byte("hints = off\nlearn = on\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if HintsEnabled() || !LearnEnabled() {
		t.Fatal("expected config parser to tolerate whitespace around '='")
	}
}

func TestReadLineAndConfirmYesSequentialStdin(t *testing.T) {
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Stdin = origStdin
		_ = r.Close()
	})
	os.Stdin = r

	go func() {
		_, _ = w.Write([]byte("first\r\nyes\nno\n"))
		_ = w.Close()
	}()

	line, err := ReadLine()
	if err != nil || line != "first" {
		t.Fatalf("ReadLine() = (%q, %v), want (%q, nil)", line, err, "first")
	}
	if !ConfirmYes(io.Discard, "") {
		t.Fatal("expected first ConfirmYes to read 'yes'")
	}
	if ConfirmYes(io.Discard, "") {
		t.Fatal("expected second ConfirmYes to read 'no'")
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
