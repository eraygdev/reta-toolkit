package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func setTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestConfigLoadSaveAndValidation(t *testing.T) {
	testutil.SilenceStdio(t)
	home := setTempHome(t)

	cfg := Load()
	if cfg["hints"] != "on" || cfg["learn"] != "off" || cfg["search-depth"] != "4" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	if code := Run([]string{"hints", "off"}); code != 0 {
		t.Fatalf("Run(hints off) = %d, want 0", code)
	}
	if code := Run([]string{"learn", "on"}); code != 0 {
		t.Fatalf("Run(learn on) = %d, want 0", code)
	}
	if code := Run([]string{"search-depth", "7"}); code != 0 {
		t.Fatalf("Run(search-depth 7) = %d, want 0", code)
	}

	if got := SearchDepth(); got != 7 {
		t.Fatalf("SearchDepth() = %d, want 7", got)
	}

	cfgPath := filepath.Join(home, ".config", "reta", "config")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config file at %s: %v", cfgPath, err)
	}

	if code := Run([]string{"unknown"}); code != 1 {
		t.Fatalf("Run(unknown) = %d, want 1", code)
	}
	if code := Run([]string{"hints", "maybe"}); code != 2 {
		t.Fatalf("Run(hints maybe) = %d, want 2", code)
	}
	if code := Run([]string{"search-depth", "0"}); code != 2 {
		t.Fatalf("Run(search-depth 0) = %d, want 2", code)
	}
	if code := Run([]string{"--bad-flag"}); code != 2 {
		t.Fatalf("Run(--bad-flag) = %d, want 2", code)
	}

	if code := Run([]string{"--reset"}); code != 0 {
		t.Fatalf("Run(--reset) = %d, want 0", code)
	}
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
