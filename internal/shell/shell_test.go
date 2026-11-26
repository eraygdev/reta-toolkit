package shell

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func captureStdout(t *testing.T, fn func() error) (error, string) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	cmdErr := fn()
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return cmdErr, buf.String()
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

func TestRunShellInitOutputs(t *testing.T) {
	testutil.SilenceStdio(t)
	for _, sh := range []string{"bash", "zsh"} {
		err, out := captureStdout(t, func() error {
			return RunShellInit([]string{sh})
		})
		assertExitCode(t, err, 0)
		if !strings.Contains(out, "rcd()") || !strings.Contains(out, "reta search") {
			t.Fatalf("unexpected %s shell-init output:\n%s", sh, out)
		}
	}

	err, psOut := captureStdout(t, func() error {
		return RunShellInit([]string{"powershell"})
	})
	assertExitCode(t, err, 0)
	if !strings.Contains(psOut, "function rcd") || !strings.Contains(psOut, "Set-Location") {
		t.Fatalf("unexpected powershell shell-init output:\n%s", psOut)
	}

	assertExitCode(t, RunShellInit([]string{"fish"}), 2)
}

func TestRunCompletionOutputs(t *testing.T) {
	testutil.SilenceStdio(t)
	tests := []struct {
		shell string
		want  string
	}{
		{"bash", "_reta_completions"},
		{"zsh", "#compdef reta"},
		{"powershell", "Register-ArgumentCompleter"},
	}
	for _, tc := range tests {
		err, out := captureStdout(t, func() error {
			return RunCompletion([]string{tc.shell})
		})
		assertExitCode(t, err, 0)
		if !strings.Contains(out, tc.want) {
			t.Fatalf("completion for %s missing %q", tc.shell, tc.want)
		}
		for _, cmd := range allCommands {
			if !strings.Contains(out, cmd) {
				t.Fatalf("completion for %s missing command %q", tc.shell, cmd)
			}
		}
	}

	assertExitCode(t, RunCompletion([]string{"tcsh"}), 2)
}
