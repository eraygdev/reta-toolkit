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

	if err := RunInitEnv([]string{"--dry-run"}); err != nil {
		t.Fatalf("RunInitEnv(--dry-run) failed: %v", err)
	}
	if _, err := os.Stat(".env"); !os.IsNotExist(err) {
		t.Fatal("--dry-run must not write .env")
	}

	if err := RunInitEnv(nil); err != nil {
		t.Fatalf("RunInitEnv() failed: %v", err)
	}
	if err := RunInitEnv(nil); err == nil {
		t.Fatal("expected RunInitEnv without --force to fail when .env exists")
	}
	if err := RunInitEnv([]string{"--force"}); err != nil {
		t.Fatalf("RunInitEnv(--force) failed: %v", err)
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
	if err := RunEnv([]string{"check"}); err == nil {
		t.Fatal("RunEnv(check) with empty key should fail")
	}
	if err := RunEnv([]string{"diff"}); err != nil {
		t.Fatalf("RunEnv(diff) with identical keys failed: %v", err)
	}
	if err := RunEnv([]string{"list", "--show"}); err != nil {
		t.Fatalf("RunEnv(list --show) failed: %v", err)
	}

	// Add a new key to .env.example and sync
	example += "NEW_SECRET=GENERATE_SECRET\nCOMMENTED= # ignored comment\n"
	if err := os.WriteFile(".env.example", []byte(example), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RunEnv([]string{"diff"}); err == nil {
		t.Fatal("RunEnv(diff) with missing keys should fail")
	}
	if err := RunEnv([]string{"sync"}); err != nil {
		t.Fatalf("RunEnv(sync) failed: %v", err)
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

	if err := RunInitEnv(nil); err != nil {
		t.Fatalf("RunInitEnv() failed: %v", err)
	}
	initBytes, _ := os.ReadFile(".env")
	if !strings.Contains(string(initBytes), `QUOTED_SECRET="GENERATE_SECRET"`) {
		t.Fatalf("quoted GENERATE_SECRET should remain literal:\n%s", string(initBytes))
	}

	// EMPTY_COMMENTED is empty -> check must fail with 1
	if err := RunEnv([]string{"check"}); err == nil {
		t.Fatal("RunEnv(check) with EMPTY_COMMENTED should fail")
	}

	// Verify sync strips trailing comments on unquoted defaults and preserves URL fragments
	if err := os.WriteFile(".env", []byte("EMPTY_COMMENTED=ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RunEnv([]string{"sync"}); err != nil {
		t.Fatalf("RunEnv(sync) failed: %v", err)
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
