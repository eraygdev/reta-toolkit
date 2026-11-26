package shell

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func() int) (int, string) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := fn()
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return code, buf.String()
}

func TestRunShellInitOutputs(t *testing.T) {
	for _, sh := range []string{"bash", "zsh"} {
		code, out := captureStdout(t, func() int {
			return RunShellInit([]string{sh})
		})
		if code != 0 {
			t.Fatalf("RunShellInit(%s) = %d", sh, code)
		}
		if !strings.Contains(out, "rcd()") || !strings.Contains(out, "reta search") {
			t.Fatalf("unexpected %s shell-init output:\n%s", sh, out)
		}
	}

	code, psOut := captureStdout(t, func() int {
		return RunShellInit([]string{"powershell"})
	})
	if code != 0 {
		t.Fatalf("RunShellInit(powershell) = %d", code)
	}
	if !strings.Contains(psOut, "function rcd") || !strings.Contains(psOut, "Set-Location") {
		t.Fatalf("unexpected powershell shell-init output:\n%s", psOut)
	}

	if code := RunShellInit([]string{"fish"}); code != 2 {
		t.Fatalf("expected exit 2 for unsupported shell, got %d", code)
	}
}

func TestRunCompletionOutputs(t *testing.T) {
	tests := []struct {
		shell string
		want  string
	}{
		{"bash", "_reta_completions"},
		{"zsh", "#compdef reta"},
		{"powershell", "Register-ArgumentCompleter"},
	}
	for _, tc := range tests {
		code, out := captureStdout(t, func() int {
			return RunCompletion([]string{tc.shell})
		})
		if code != 0 {
			t.Fatalf("RunCompletion(%s) = %d", tc.shell, code)
		}
		if !strings.Contains(out, tc.want) {
			t.Fatalf("completion for %s missing %q", tc.shell, tc.want)
		}
		for _, cmd := range allCommands {
			if !strings.Contains(out, cmd) {
				t.Fatalf("completion for %s missing command %q", tc.shell, cmd)
			}
		}
	}

	if code := RunCompletion([]string{"tcsh"}); code != 2 {
		t.Fatalf("expected exit 2 for unsupported shell, got %d", code)
	}
}
