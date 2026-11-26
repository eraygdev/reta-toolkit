package exporter

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
