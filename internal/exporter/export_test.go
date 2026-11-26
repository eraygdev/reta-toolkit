package exporter

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func TestParseSizeAndFnmatch(t *testing.T) {
	sizes := map[string]int64{
		"100":  100,
		"2K":   2048,
		"1.5M": 1572864,
		"1G":   1073741824,
	}
	for in, want := range sizes {
		got, err := parseSize(in)
		if err != nil || got != want {
			t.Fatalf("parseSize(%q) = (%d, %v), want %d", in, got, err, want)
		}
	}

	if !fnmatch("src/*.go", "src/main.go") {
		t.Fatal("expected src/*.go to match src/main.go")
	}
	if !fnmatch("*.md", "docs/readme.md") {
		t.Fatal("expected *.md to match across '/'")
	}
	if !fnmatch("file[0-9].txt", "file3.txt") || fnmatch("file[0-9].txt", "filea.txt") {
		t.Fatal("character class matching failed")
	}
	if !fnmatch("file[!0-9].txt", "filea.txt") || fnmatch("file[!0-9].txt", "file3.txt") {
		t.Fatal("negated character class matching failed")
	}
}

func TestResolveOutputAndDangerousPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if !isDangerousPath(home) {
		t.Fatal("home directory must be treated as dangerous")
	}
	if runtime.GOOS == "windows" {
		if !isDangerousPath(strings.ToLower(home)) {
			t.Fatal("Windows dangerous path check must be case-insensitive")
		}
		if !isDangerousPath(`C:\`) {
			t.Fatal(`C:\ must be dangerous on Windows`)
		}
	} else {
		if !isDangerousPath("/") {
			t.Fatal("/ must be dangerous")
		}
		if runtime.GOOS == "darwin" && !isDangerousPath("/Users") {
			t.Fatal("/Users must be dangerous on macOS")
		}
	}

	proj := filepath.Join(home, "proj")
	tildeOut := resolveOutputFromDir(proj, "~/out")
	want := resolveAbs(filepath.Join(home, "out", "project_export.md"))
	if tildeOut != want {
		t.Fatalf("resolveOutputFromDir(~/out) = %q, want %q", tildeOut, want)
	}
}

func TestDecodeTextLatin1Fallback(t *testing.T) {
	if got := decodeText([]byte("hello \xc3\xa7")); got != "hello ç" {
		t.Fatalf("UTF-8 decode failed: %q", got)
	}
	if got := decodeText([]byte{0x63, 0xe7, 0x61}); got != "cça" {
		t.Fatalf("Latin-1 fallback failed: %q", got)
	}
}

func TestRunExportFilteringAndExportignore(t *testing.T) {
	testutil.SilenceStdio(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}

	_ = os.WriteFile(filepath.Join(root, ".exportignore"), []byte("# ignore\r\nsrc/ignored.py\r\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "src", "main.py"), []byte("print('ok')\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "src", "ignored.py"), []byte("print('no')\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "data", "big.csv"), []byte("a,b\n1,2\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "node_modules", "pkg", "index.js"), []byte("x\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=123\n"), 0o644)

	outPath := filepath.Join(root, "exports", "project_export.md")
	if code := Run([]string{"-r", root, "--include", `src\*.py`}); code != 0 {
		t.Fatalf("Run(export) = %d, want 0", code)
	}

	contentBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)
	if !strings.Contains(content, "print('ok')") {
		t.Fatalf("expected main.py content in export:\n%s", content)
	}
	if strings.Contains(content, "print('no')") {
		t.Fatalf("ignored.py should have been excluded by .exportignore")
	}
	if strings.Contains(content, "SECRET=123") || strings.Contains(content, "node_modules") {
		t.Fatalf("secret or hidden dir leaked into export")
	}
	if !strings.Contains(content, "## `data/`") {
		t.Fatalf("expected data/ summary entry in export")
	}
}

func TestMatchesIgnoreDirAndFileDistinction(t *testing.T) {
	patterns := []string{"src/generated", "keepfile/"}

	if !matchesIgnore("src/generated", true, patterns) {
		t.Fatal("directory pattern without trailing slash should match directory")
	}
	if matchesIgnore("keepfile", false, patterns) {
		t.Fatal("trailing-slash directory pattern must not match a regular file")
	}
	if !matchesIgnore("keepfile", true, patterns) {
		t.Fatal("trailing-slash directory pattern must match directory")
	}
}

func TestEstimateTokensCodeAndProse(t *testing.T) {
	numEst := estimateTokens("123456")
	if numEst.ChatGPT != 2 || numEst.Gemini != 6 || numEst.Claude != 6 {
		t.Fatalf("unexpected digit token counts for 123456: %+v", numEst)
	}

	trEst := estimateTokens("Merhaba dünya! 2026")
	if trEst.ChatGPT < 5 || trEst.ChatGPT > 8 || trEst.Gemini < 7 || trEst.Gemini > 10 || trEst.Claude < 8 || trEst.Claude > 13 {
		t.Fatalf("unexpected Turkish+digit token counts: %+v", trEst)
	}

	emojiEst := estimateTokens("👨‍👩‍👧‍👦")
	if emojiEst.ChatGPT > 6 || emojiEst.Claude > 6 {
		t.Fatalf("ZWJ emoji inflated token estimate: %+v", emojiEst)
	}

	jsonEst := estimateTokens(`{"id":123456,"user_name":"eray_dev","active":true}`)
	if jsonEst.Claude <= jsonEst.ChatGPT {
		t.Fatalf("expected Claude 4.7+ > ChatGPT on compact JSON: %+v", jsonEst)
	}

	code := "func get_user_by_id_from_db(userId int, isActiveSession bool) (string, error) {\n\tif userId <= 0 || !isActiveSession {\n\t\treturn \"\", nil\n\t}\n\treturn \"ok\", nil\n}\n"
	est := estimateTokens(code)
	if est.ChatGPT <= 0 || est.Gemini <= est.ChatGPT || est.Claude <= est.Gemini {
		t.Fatalf("unexpected model token counts for code: %+v", est)
	}

	var sink tokenEstimate
	allocs := testing.AllocsPerRun(100, func() {
		sink = estimateTokens(code)
	})
	if allocs != 0 || sink.ChatGPT == 0 {
		t.Fatalf("expected 0 allocations, got %.0f", allocs)
	}

	prose := "The quick brown fox jumps over the lazy dog."
	pEst := estimateTokens(prose)
	if pEst.ChatGPT < 8 || pEst.ChatGPT > 14 || pEst.Claude < 8 || pEst.Claude > 15 {
		t.Fatalf("unexpected prose token estimate: %+v", pEst)
	}
}
