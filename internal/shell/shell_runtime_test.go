package shell

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func TestShellScriptsParseWithHostShells(t *testing.T) {
	testutil.SilenceStdio(t)

	if bashPath, err := exec.LookPath("bash"); err == nil && runtime.GOOS != "windows" {
		for _, fn := range []func([]string) error{RunShellInit, RunCompletion} {
			err, out := captureStdout(t, func() error { return fn([]string{"bash"}) })
			assertExitCode(t, err, 0)
			cmd := exec.Command(bashPath, "-n")
			cmd.Stdin = strings.NewReader(out)
			if res, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("bash -n syntax check failed: %v\n%s", err, string(res))
			}
		}
	}

	if zshPath, err := exec.LookPath("zsh"); err == nil {
		for _, fn := range []func([]string) error{RunShellInit, RunCompletion} {
			err, out := captureStdout(t, func() error { return fn([]string{"zsh"}) })
			assertExitCode(t, err, 0)
			cmd := exec.Command(zshPath, "-n")
			cmd.Stdin = strings.NewReader(out)
			if res, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("zsh -n syntax check failed: %v\n%s", err, string(res))
			}
		}
	}

	psBin := ""
	for _, cand := range []string{"pwsh", "powershell"} {
		if p, err := exec.LookPath(cand); err == nil {
			psBin = p
			break
		}
	}
	if psBin != "" {
		for _, fn := range []func([]string) error{RunShellInit, RunCompletion} {
			err, out := captureStdout(t, func() error { return fn([]string{"powershell"}) })
			assertExitCode(t, err, 0)
			check := `$errors = $null; $null = [System.Management.Automation.Language.Parser]::ParseInput([Console]::In.ReadToEnd(), [ref]$null, [ref]$errors); if ($errors.Count -gt 0) { $errors | ForEach-Object { Write-Error $_.Message }; exit 1 }`
			cmd := exec.Command(psBin, "-NoProfile", "-Command", check)
			cmd.Stdin = strings.NewReader(out)
			if res, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("PowerShell AST check failed: %v\n%s", err, string(res))
			}
		}
	}
}
