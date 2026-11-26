package search

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eraygdev/reta-toolkit/internal/retasign"
)

func TestIsSkippedPathCrossPlatform(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if isSkippedPath(filepath.Join(home, "work", "myapp")) {
		t.Fatal("normal project under home should not be skipped")
	}
	if !isSkippedPath(filepath.Join(home, "work", "node_modules", "pkg")) {
		t.Fatal("node_modules path must be skipped")
	}
	if runtime.GOOS == "darwin" {
		if isSkippedPath("/private/tmp/myproj") {
			t.Fatal("macOS /private/tmp project path must not be falsely skipped")
		}
		if !isSkippedPath("/private/tmp/myproj/node_modules") {
			t.Fatal("node_modules inside /private/tmp must still be skipped")
		}
	}
}

func TestResolveQueryAndDepthDiscovery(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("NO_COLOR", "1")

	projA := filepath.Join(home, "team1", "api")
	projB := filepath.Join(home, "team2", "api")
	projDeep := filepath.Join(home, "a", "b", "c", "d", "deep")
	for _, p := range []string{projA, projB, projDeep} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := retasign.WriteSignature(p); err != nil {
			t.Fatal(err)
		}
	}

	discovered4 := scanHomeForProjects(home, 4)
	if _, ok := discovered4[resolvePath(projA)]; !ok {
		t.Fatal("expected projA at depth 2")
	}
	if _, ok := discovered4[resolvePath(projDeep)]; ok {
		t.Fatal("projDeep at depth 5 must not be found when maxDepth=4")
	}

	discovered5 := scanHomeForProjects(home, 5)
	if _, ok := discovered5[resolvePath(projDeep)]; !ok {
		t.Fatal("expected projDeep when maxDepth=5")
	}

	status, matched := resolveQuery("api", []string{resolvePath(projA), resolvePath(projB)})
	if status != "jump" || len(matched) != 1 || matched[0] != resolvePath(projA) {
		t.Fatalf("unexpected exact query result: %q %v", status, matched)
	}

	status, matched = resolveQuery("2", []string{resolvePath(projA), resolvePath(projB)})
	if status != "jump" || matched[0] != resolvePath(projB) {
		t.Fatalf("unexpected numeric query result: %q %v", status, matched)
	}

	if code := Run([]string{"deep"}); code != 1 {
		t.Fatalf("expected depth=4 search for deep to fail with 1, got %d", code)
	}

	if code := Run([]string{"api"}); code != 0 {
		t.Fatalf("expected search api to succeed, got %d", code)
	}
	jfData, err := os.ReadFile(jumpFile())
	if err != nil {
		t.Fatalf("expected jump file written: %v", err)
	}
	if strings.TrimSpace(string(jfData)) != resolvePath(projA) {
		t.Fatalf("jump file = %q, want %q", string(jfData), resolvePath(projA))
	}
}
