package clean

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/testutil"
)

func TestRunHelpAndUnknownOption(t *testing.T) {
	testutil.SilenceStdio(t)
	t.Setenv("NO_COLOR", "1")
	if code := Run([]string{"--help"}); code != 0 {
		t.Fatalf("Run(--help) = %d, want 0", code)
	}
	if code := Run([]string{"--unknown"}); code != 1 {
		t.Fatalf("Run(--unknown) = %d, want 1", code)
	}
}

func TestCleanPythonCachesRespectsSkipDirs(t *testing.T) {
	home := t.TempDir()
	mustWrite := func(rel string) string {
		p := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	cacheFile := mustWrite(filepath.Join("proj", "__pycache__", "mod.cpython-312.pyc"))
	loosePyc := mustWrite(filepath.Join("proj", "old.pyc"))
	sourcePy := mustWrite(filepath.Join("proj", "app.py"))
	venvPyc := mustWrite(filepath.Join("proj", ".venv", "__pycache__", "keep.pyc"))
	nodePyc := mustWrite(filepath.Join("proj", "node_modules", "keep.pyc"))

	cleanPythonCaches(home)

	if _, err := os.Stat(filepath.Dir(cacheFile)); !os.IsNotExist(err) {
		t.Fatal("expected __pycache__ directory to be removed")
	}
	if _, err := os.Stat(loosePyc); !os.IsNotExist(err) {
		t.Fatal("expected loose .pyc file to be removed")
	}
	for _, keep := range []string{sourcePy, venvPyc, nodePyc} {
		if _, err := os.Stat(keep); err != nil {
			t.Fatalf("expected %s to be preserved: %v", keep, err)
		}
	}
}

func TestRemoveDirContentsKeepsParentDir(t *testing.T) {
	dir := t.TempDir()
	subFile := filepath.Join(dir, "sub", "item.txt")
	if err := os.MkdirAll(filepath.Dir(subFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subFile, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "top.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	removeDirContents(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("expected parent dir to remain: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty dir, got %d entries", len(entries))
	}
}
