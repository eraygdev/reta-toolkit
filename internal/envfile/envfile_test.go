package envfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func enterTempDir(t *testing.T) string {
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
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("NO_COLOR", "1")
	t.Setenv("RETA_HINTS", "off")
	return tmp
}

func TestParseEnvFileCRLFQuotesAndDuplicates(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	raw := "# header\r\nexport APP_NAME=\"my app\"\r\nSINGLE='val'\r\nDUP=first\r\nDUP=second\r\n"
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := parseEnvFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Values["APP_NAME"] != "my app" {
		t.Fatalf("APP_NAME = %q", parsed.Values["APP_NAME"])
	}
	if parsed.Values["SINGLE"] != "val" {
		t.Fatalf("SINGLE = %q", parsed.Values["SINGLE"])
	}
	if parsed.Values["DUP"] != "second" {
		t.Fatalf("expected last duplicate to win, got %q", parsed.Values["DUP"])
	}
	if len(parsed.Duplicates) != 1 || parsed.Duplicates[0] != "DUP" {
		t.Fatalf("expected DUP in Duplicates, got %v", parsed.Duplicates)
	}
}

func TestRunInitEnvAndEnvSubcommands(t *testing.T) {
	testutil.SilenceStdio(t)
	enterTempDir(t)

	example := strings.Join([]string{
		"APP_KEY=GENERATE_SECRET # random key",
		"export MODE=production",
		"EMPTY_KEY=",
		"QUOTED=\"hello world\"",
	}, "\r\n") + "\r\n"
	if err := os.WriteFile(".env.example", []byte(example), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := RunInitEnv([]string{"--dry-run"}); code != 0 {
		t.Fatalf("RunInitEnv(--dry-run) = %d", code)
	}
	if _, err := os.Stat(".env"); !os.IsNotExist(err) {
		t.Fatal("--dry-run must not write .env")
	}

	if code := RunInitEnv(nil); code != 0 {
		t.Fatalf("RunInitEnv() = %d", code)
	}
	if code := RunInitEnv(nil); code != 1 {
		t.Fatalf("expected RunInitEnv without --force to exit 1 when .env exists, got %d", code)
	}
	if code := RunInitEnv([]string{"--force"}); code != 0 {
		t.Fatalf("RunInitEnv(--force) = %d", code)
	}

	data, err := os.ReadFile(".env")
	if err != nil {
		t.Fatal(err)
	}
	txt := string(data)
	if strings.Contains(txt, "GENERATE_SECRET") {
		t.Fatalf("GENERATE_SECRET was not resolved:\n%s", txt)
	}
	if !strings.Contains(txt, "export MODE=production") {
		t.Fatalf("export line lost:\n%s", txt)
	}

	// EMPTY_KEY is empty -> check exits 1
	if code := RunEnv([]string{"check"}); code != 1 {
		t.Fatalf("RunEnv(check) with empty key = %d, want 1", code)
	}
	if code := RunEnv([]string{"diff"}); code != 0 {
		t.Fatalf("RunEnv(diff) with identical keys = %d, want 0", code)
	}
	if code := RunEnv([]string{"list", "--show"}); code != 0 {
		t.Fatalf("RunEnv(list --show) = %d, want 0", code)
	}

	// Add a new key to .env.example and sync
	example += "NEW_SECRET=GENERATE_SECRET\nCOMMENTED= # ignored comment\n"
	if err := os.WriteFile(".env.example", []byte(example), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := RunEnv([]string{"diff"}); code != 1 {
		t.Fatalf("RunEnv(diff) with missing keys = %d, want 1", code)
	}
	if code := RunEnv([]string{"sync"}); code != 0 {
		t.Fatalf("RunEnv(sync) = %d, want 0", code)
	}

	syncedBytes, _ := os.ReadFile(".env")
	synced := string(syncedBytes)
	if !strings.Contains(synced, "NEW_SECRET=") || strings.Contains(synced, "GENERATE_SECRET") {
		t.Fatalf("sync did not resolve NEW_SECRET:\n%s", synced)
	}
	if strings.Contains(synced, "# ignored comment") {
		t.Fatalf("inline comment leaked into synced value:\n%s", synced)
	}
}

func TestInlineCommentsQuotedValuesAndUrls(t *testing.T) {
	testutil.SilenceStdio(t)
	enterTempDir(t)

	example := strings.Join([]string{
		"PORT=8080 # server port",
		`COLOR="#ffffff" # hex color`,
		"DOC_URL=https://example.com#section",
		"EMPTY_COMMENTED= # fill later",
		`QUOTED_SECRET="GENERATE_SECRET"`,
	}, "\n") + "\n"
	if err := os.WriteFile(".env.example", []byte(example), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := RunInitEnv(nil); code != 0 {
		t.Fatalf("RunInitEnv() = %d", code)
	}
	initBytes, _ := os.ReadFile(".env")
	if !strings.Contains(string(initBytes), `QUOTED_SECRET="GENERATE_SECRET"`) {
		t.Fatalf("quoted GENERATE_SECRET should remain literal:\n%s", string(initBytes))
	}

	// EMPTY_COMMENTED is empty -> check must fail with 1
	if code := RunEnv([]string{"check"}); code != 1 {
		t.Fatalf("RunEnv(check) with EMPTY_COMMENTED = %d, want 1", code)
	}

	// Verify sync strips trailing comments on unquoted defaults and preserves URL fragments
	if err := os.WriteFile(".env", []byte("EMPTY_COMMENTED=ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := RunEnv([]string{"sync"}); code != 0 {
		t.Fatalf("RunEnv(sync) = %d, want 0", code)
	}
	syncedBytes, _ := os.ReadFile(".env")
	synced := string(syncedBytes)
	for _, want := range []string{
		"PORT=8080\n",
		`COLOR="#ffffff"` + "\n",
		"DOC_URL=https://example.com#section\n",
		`QUOTED_SECRET="GENERATE_SECRET"`,
	} {
		if !strings.Contains(synced, want) {
			t.Fatalf("synced .env missing %q:\n%s", want, synced)
		}
	}
	if strings.Contains(synced, "# server port") || strings.Contains(synced, "# hex color") {
		t.Fatalf("inline comment leaked into synced .env:\n%s", synced)
	}
}
