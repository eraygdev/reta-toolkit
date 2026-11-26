package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func setTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
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

func TestConfigLoadSaveAndValidation(t *testing.T) {
	testutil.SilenceStdio(t)
	home := setTempHome(t)

	cfg := Load()
	if cfg["hints"] != "on" || cfg["learn"] != "off" || cfg["search-depth"] != "4" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	assertExitCode(t, Run([]string{"hints", "off"}), 0)
	assertExitCode(t, Run([]string{"learn", "on"}), 0)
	assertExitCode(t, Run([]string{"search-depth", "7"}), 0)

	if got := SearchDepth(); got != 7 {
		t.Fatalf("SearchDepth() = %d, want 7", got)
	}

	cfgPath := filepath.Join(home, ".config", "reta", "config")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config file at %s: %v", cfgPath, err)
	}

	assertExitCode(t, Run([]string{"unknown"}), 1)
	assertExitCode(t, Run([]string{"hints", "maybe"}), 2)
	assertExitCode(t, Run([]string{"search-depth", "0"}), 2)
	assertExitCode(t, Run([]string{"--bad-flag"}), 2)

	assertExitCode(t, Run([]string{"--reset"}), 0)
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("expected config file removed after --reset")
	}
	if got := SearchDepth(); got != DefaultSearchDepth {
		t.Fatalf("SearchDepth() after reset = %d, want %d", got, DefaultSearchDepth)
	}
}

func TestConfigIgnoresCorruptLinesAndCRLF(t *testing.T) {
	home := setTempHome(t)
	cfgPath := filepath.Join(home, ".config", "reta", "config")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := "# comment\r\ninvalid_line\r\nunknown=123\r\nsearch-depth=0\r\nhints=off\r\n"
	if err := os.WriteFile(cfgPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg["hints"] != "off" {
		t.Fatalf("expected hints=off, got %q", cfg["hints"])
	}
	if cfg["search-depth"] != "4" {
		t.Fatalf("invalid search-depth=0 must fallback to 4, got %q", cfg["search-depth"])
	}
}
