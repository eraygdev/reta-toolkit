package adopt

import (
	"reflect"
	"strings"
	"testing"

	reta "github.com/eraygdev/reta-toolkit"
)

func TestAlreadyUsesReta(t *testing.T) {
	if !AlreadyUsesReta("include $(RETA_BASE)\n") {
		t.Fatal("expected active include $(RETA_BASE) to be detected")
	}
	if !AlreadyUsesReta("include ~/.reta/lib/base.mk\n") {
		t.Fatal("expected include base.mk to be detected")
	}
	if AlreadyUsesReta("# include $(RETA_BASE)\nall:\n\techo hi\n") {
		t.Fatal("commented include must be ignored")
	}
	if AlreadyUsesReta("APP := demo\nall:\n\techo hi\n") {
		t.Fatal("plain Makefile must return false")
	}
}

func TestLoadBaseTargets(t *testing.T) {
	targets := LoadBaseTargets(reta.BaseMK)
	for _, expected := range []string{"help", "up", "down", "logs", "clean", "clean-docker", "export", "git-sync", "check-secrets"} {
		if !targets[expected] {
			t.Fatalf("expected base target %q to be loaded", expected)
		}
	}
	if targets[".PHONY"] || targets[".DEFAULT_GOAL"] {
		t.Fatal("directives must not be loaded as base targets")
	}
}

func TestSplitBlocks(t *testing.T) {
	blocks := splitBlocks([]string{
		`SRCS := a.py \`,
		`        b.py \`,
		"        c.py",
		"build:",
		"\tpython $(SRCS)",
	})
	if blocks[0].Kind != "directive" || len(blocks[0].Lines) != 3 {
		t.Fatalf("unexpected directive block: %+v", blocks[0])
	}
	if blocks[1].Kind != "target" || blocks[1].Lines[1] != "\tpython $(SRCS)" {
		t.Fatalf("unexpected target block: %+v", blocks[1])
	}

	condBlocks := splitBlocks([]string{
		"ifeq ($(ENV),prod)",
		"ifdef DEBUG",
		"FLAGS := -g",
		"endif",
		"else",
		"FLAGS := -O2",
		"endif",
		"run:",
		"\techo $(FLAGS)",
	})
	if condBlocks[0].Kind != "directive" || len(condBlocks[0].Lines) != 7 {
		t.Fatalf("expected nested conditional block of 7 lines, got %+v", condBlocks[0])
	}

	defBlocks := splitBlocks([]string{
		"define HELPER_MSG",
		"usage: make run",
		"endef",
		"run:",
		"\techo ok",
	})
	if defBlocks[0].Kind != "directive" || len(defBlocks[0].Lines) != 3 {
		t.Fatalf("expected define block of 3 lines, got %+v", defBlocks[0])
	}

	recipeBlocks := splitBlocks([]string{
		"deploy:",
		"\techo step1",
		"",
		"\techo step2",
		"",
		"test:",
		"\tpytest",
	})
	var targetCount int
	var firstTarget block
	for _, b := range recipeBlocks {
		if b.Kind == "target" {
			if targetCount == 0 {
				firstTarget = b
			}
			targetCount++
		}
	}
	if targetCount != 2 || len(firstTarget.Lines) != 4 || firstTarget.Lines[3] != "\techo step2" {
		t.Fatalf("unexpected recipe block split: count=%d first=%+v", targetCount, firstTarget)
	}
}

func TestExtractHelpDescriptions(t *testing.T) {
	blocks := splitBlocks([]string{
		"help:",
		`	@echo "  make status   Show service status"`,
		`	@echo "  make lint - Run ruff checks"`,
		`	@echo "  make fmt : Format source files"`,
		`	@printf "  %-10s %s\n" "build" "Build binary"`,
		`	@echo "Example: make train EPOCHS=30"`,
		`	@echo "  n8n-publish       Publish all workflows"`,
		`	@echo "  cd ~/projects/demo"`,
		"n8n-publish:",
		"\techo ok",
	})
	descs := extractHelpDescriptions(blocks)
	want := map[string]string{
		"status":      "Show service status",
		"lint":        "Run ruff checks",
		"fmt":         "Format source files",
		"build":       "Build binary",
		"n8n-publish": "Publish all workflows",
	}
	if !reflect.DeepEqual(descs, want) {
		t.Fatalf("got %v, want %v", descs, want)
	}
}

func TestClassifyHeading(t *testing.T) {
	lines := []string{
		"# === DATA MODEL ===",
		"a:",
		"",
		"# --- DOCKER ---",
		"b:",
		"",
		"# ═══════════════════════════════════════════════════════════",
		"# N8N WORKFLOWS — Export / Import (exports/ klasoru)",
		"# ═══════════════════════════════════════════════════════════",
		"export-workflows:",
		"\techo ok",
		"",
		"# --- Broken Header",
		"",
		"# Database Tasks",
		"migrate:",
		"\techo ok",
		"",
		"# This is a normal explanatory sentence.",
		"seed:",
		"\techo ok",
		"",
		"# TRAILING HEADING",
	}
	blocks := splitBlocks(lines)
	got := make(map[string][2]string)
	for i, b := range blocks {
		if b.Kind == "comment" {
			st, title := classifyHeading(blocks, i)
			got[b.Lines[0]] = [2]string{st, title}
		}
	}

	if got["# === DATA MODEL ==="] != [2]string{"heading", "DATA MODEL"} {
		t.Fatalf("unexpected DATA MODEL classification: %v", got["# === DATA MODEL ==="])
	}
	if got["# --- DOCKER ---"] != [2]string{"heading", "DOCKER"} {
		t.Fatalf("unexpected DOCKER classification: %v", got["# --- DOCKER ---"])
	}
	if got["# N8N WORKFLOWS — Export / Import (exports/ klasoru)"] != [2]string{"heading", "N8N WORKFLOWS — Export / Import (exports/ klasoru)"} {
		t.Fatalf("unexpected unicode heading: %v", got["# N8N WORKFLOWS — Export / Import (exports/ klasoru)"])
	}
	if got["# --- Broken Header"] != [2]string{"uncertain", "Broken Header"} {
		t.Fatalf("unexpected Broken Header: %v", got["# --- Broken Header"])
	}
	if got["# Database Tasks"] != [2]string{"uncertain", "Database Tasks"} {
		t.Fatalf("unexpected Database Tasks: %v", got["# Database Tasks"])
	}
	if got["# This is a normal explanatory sentence."] != [2]string{"normal", ""} {
		t.Fatalf("unexpected sentence: %v", got["# This is a normal explanatory sentence."])
	}
	if got["# TRAILING HEADING"] != [2]string{"uncertain", "TRAILING HEADING"} {
		t.Fatalf("unexpected trailing heading: %v", got["# TRAILING HEADING"])
	}
}

func TestConvertMakefileEndToEnd(t *testing.T) {
	baseTargets := LoadBaseTargets(reta.BaseMK)
	header := LoadTemplateHeader(false, false)

	raw := strings.Join([]string{
		"# === Project Header ===",
		".DEFAULT_GOAL := help",
		"SHELL := /bin/bash",
		"SHELL_CUSTOM := /bin/zsh",
		"APP := demo",
		"",
		"# ====================",
		"# HELP",
		"# ====================",
		"help:",
		`	@echo "  make status   Check status"`,
		`	@printf "  %-10s %s\n" "build" "Build binary"`,
		"",
		"status:",
		"\t@echo $(APP)",
		"",
		"prep:",
		"\t@echo prep",
		"",
		"# --- DATA ---",
		"migrate: # Run DB migrations",
		"\talembic upgrade head",
		"",
		"## DOCKER",
		"up:",
		"\tdocker compose up -d",
		"",
		"clean:",
		"\trm -rf dist",
		"",
		"# --- Half Heading",
		"",
		"# Database Tasks",
		"build:",
		"\tgo build ./...",
		"",
		"# TRAILING SECTION",
	}, "\n")

	converted, report := ConvertMakefile(raw, baseTargets, header)

	if !AlreadyUsesReta(converted) {
		t.Fatal("converted Makefile must be recognized as using base.mk")
	}
	for _, want := range []string{
		"# Project Header\n",
		"# .DEFAULT_GOAL := help",
		"# SHELL := /bin/bash",
		"SHELL_CUSTOM := /bin/zsh",
		"# HELP\n",
		"# DOCKER\n",
		"# === DATA ===",
		"status:  ## Check status",
		"prep:  ## EDIT_HERE: add description",
		"migrate:  ## Run DB migrations",
		"build:  ## Build binary",
		"\talembic upgrade head",
		"# EDIT_HERE: verify if 'Half Heading'",
		"# EDIT_HERE: verify if 'Database Tasks'",
		"# EDIT_HERE: verify if 'TRAILING SECTION'",
	} {
		if !strings.Contains(converted, want) {
			t.Fatalf("converted output missing %q\n---\n%s", want, converted)
		}
	}
	if strings.Contains(converted, "# === HELP ===") || strings.Contains(converted, "# === DOCKER ===") {
		t.Fatal("empty sections must be demoted")
	}
	if len(report.CommentedTargets) != 3 {
		t.Fatalf("expected 3 commented targets, got %v", report.CommentedTargets)
	}
}

func TestSkipsGreetingHIWhenOriginalDefinesHI(t *testing.T) {
	baseTargets := LoadBaseTargets(reta.BaseMK)
	headerWithHI := LoadTemplateHeader(true, true)
	raw := "hi:  ## Custom hi target\n\t@echo custom\n"

	converted, _ := ConvertMakefile(raw, baseTargets, headerWithHI)
	if strings.Count(converted, "\nhi:") != 1 {
		t.Fatalf("expected exactly 1 hi target, got:\n%s", converted)
	}
	if !strings.Contains(converted, "Custom hi target") || strings.Contains(converted, "Say hello") {
		t.Fatalf("unexpected hi target replacement:\n%s", converted)
	}
}
