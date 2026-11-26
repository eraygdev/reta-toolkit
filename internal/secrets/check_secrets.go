package secrets

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

const (
	IgnoreMarker = "check-secrets: ignore"
	MaxScanBytes = 1024 * 1024
)

var sensitiveFilenames = map[string]bool{
	".env":             true,
	".env.local":       true,
	".env.production":  true,
	".env.prod":        true,
	".env.staging":     true,
	".env.development": true,
	".env.dev":         true,
	".env.test":        true,
	"id_rsa":           true,
	"id_ed25519":       true,
	"id_ecdsa":         true,
}

var sensitiveExts = map[string]bool{
	".pem": true,
	".key": true,
	".p12": true,
	".pfx": true,
}

var skipFiles = map[string]bool{
	".env.example":       true,
	".env.sample":        true,
	".env.template":      true,
	"project_export.md":  true,
	"project_export.txt": true,
}

var skipDirs = map[string]bool{
	".git":          true,
	".hg":           true,
	".svn":          true,
	"__pycache__":   true,
	"venv":          true,
	".venv":         true,
	"env":           true,
	"node_modules":  true,
	"dist":          true,
	"build":         true,
	"target":        true,
	".next":         true,
	".nuxt":         true,
	".pytest_cache": true,
	".mypy_cache":   true,
	".ruff_cache":   true,
}

type secretPattern struct {
	Label   string
	Pattern *regexp.Regexp
}

var secretPatterns = []secretPattern{
	{"Private key block", regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----`)},
	{"OpenAI API key", regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{24,}\b`)},
	{"Anthropic API key", regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}\b`)},
	{"Google/Gemini API key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`)},
	{"GitHub token", regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,})\b`)},
	{"AWS Access Key ID", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"Slack token", regexp.MustCompile(`\bxox[baprs]-[0-9A-Za-z-]{10,}\b`)},
	{"Stripe live secret", regexp.MustCompile(`\bsk_live_[0-9A-Za-z]{20,}\b`)},
}

type leakFinding struct {
	Rel   string
	Line  int
	Label string
}

func printHelp() {
	fmt.Println("Usage: reta check-secrets [--staged]")
	fmt.Println("")
	fmt.Println("Scan a project for tracked .env files and hardcoded secrets.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --staged      Scan only git staged changes (for pre-commit hooks)")
	fmt.Println("  -h, --help    Show this help message")
}

func isGitRepo(root string) bool {
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		return true
	}
	cmd := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func gitToplevel(root string) string {
	cmd := exec.Command("git", "-C", root, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err == nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed != "" {
			if abs, err := filepath.Abs(trimmed); err == nil {
				return abs
			}
			return trimmed
		}
	}
	return root
}

func splitNullStrings(raw []byte) []string {
	parts := bytes.Split(raw, []byte{0})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) > 0 {
			out = append(out, string(p))
		}
	}
	return out
}

func gitTrackedFiles(root string) []string {
	cmd := exec.Command("git", "-C", root, "ls-files", "-z")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	rels := splitNullStrings(out)
	paths := make([]string, 0, len(rels))
	for _, r := range rels {
		paths = append(paths, filepath.Join(root, filepath.FromSlash(r)))
	}
	return paths
}

func gitStagedFiles(root string) []string {
	cmd := exec.Command("git", "-C", root, "diff", "--cached", "--name-only", "--diff-filter=ACMR", "-z")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return splitNullStrings(out)
}

func readStagedBlob(root, relPosix string) ([]byte, bool) {
	cmd := exec.Command("git", "-C", root, "show", ":"+relPosix)
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	return out, true
}

func walkProjectFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		files = append(files, p)
		return nil
	})
	return files
}

func checkGitignore(root string, inGit bool) (bool, string) {
	if inGit {
		cmd := exec.Command("git", "-C", root, "check-ignore", "-q", ".env")
		if err := cmd.Run(); err == nil {
			return true, ".env is ignored in .gitignore"
		}
	}

	giPath := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(giPath)
	if err != nil {
		return false, ".gitignore not found"
	}

	envRules := map[string]bool{
		".env": true, "/.env": true, ".env*": true, "/.env*": true,
		"*.env": true, "**/.env": true, "**/.env*": true,
	}
	for _, raw := range strings.Split(string(data), "\n") {
		pat := strings.TrimSpace(raw)
		if pat == "" || strings.HasPrefix(pat, "#") {
			continue
		}
		if envRules[pat] {
			return true, ".env is ignored in .gitignore"
		}
	}
	return false, ".env is not listed in .gitignore"
}

func isSensitivePath(p string) bool {
	base := filepath.Base(p)
	if skipFiles[base] {
		return false
	}
	if sensitiveFilenames[base] || strings.HasPrefix(base, ".env.") {
		return true
	}
	return sensitiveExts[strings.ToLower(filepath.Ext(base))]
}

func scanText(relPosix, text string) []leakFinding {
	var findings []leakFinding
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for idx, line := range lines {
		if strings.Contains(line, IgnoreMarker) {
			continue
		}
		for _, sp := range secretPatterns {
			if sp.Pattern.MatchString(line) {
				findings = append(findings, leakFinding{
					Rel:   relPosix,
					Line:  idx + 1,
					Label: sp.Label,
				})
				break
			}
		}
	}
	return findings
}

func isBinaryBytes(data []byte) bool {
	limit := len(data)
	if limit > 4096 {
		limit = 4096
	}
	return bytes.IndexByte(data[:limit], 0) >= 0
}

func scanFile(path, root string) []leakFinding {
	base := filepath.Base(path)
	if skipFiles[base] || isSensitivePath(path) {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxScanBytes {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil || isBinaryBytes(data) {
		return nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	return scanText(filepath.ToSlash(rel), string(data))
}

func scanStagedBlob(root, relPosix string) []leakFinding {
	base := filepath.Base(filepath.FromSlash(relPosix))
	if skipFiles[base] || isSensitivePath(relPosix) {
		return nil
	}
	raw, ok := readStagedBlob(root, relPosix)
	if !ok || len(raw) > MaxScanBytes || isBinaryBytes(raw) {
		return nil
	}
	return scanText(relPosix, string(raw))
}

func Run(args []string) int {
	staged := false
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			printHelp()
			return 0
		case "--staged":
			staged = true
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
			return 2
		}
	}

	c := ui.Colors()
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", c.Red, c.Reset, err)
		return 1
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		root = cwd
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}

	inGit := isGitRepo(root)
	if staged && !inGit {
		fmt.Fprintf(os.Stderr, "%sError:%s --staged requires a git repository.\n", c.Red, c.Reset)
		return 1
	}

	scanRoot := root
	if staged {
		scanRoot = gitToplevel(root)
		if resolved, err := filepath.EvalSymlinks(scanRoot); err == nil {
			scanRoot = resolved
		}
	}

	fmt.Printf("%scheck-secrets%s\n", c.Bold, c.Reset)
	fmt.Printf("  root: %s\n", scanRoot)
	if staged {
		fmt.Println("  mode: staged")
	}
	fmt.Println("")

	failures := 0
	warnings := 0

	okIgnore, ignoreMsg := checkGitignore(scanRoot, inGit)
	if okIgnore {
		fmt.Printf("  %sOK%s    %s\n", c.Green, c.Reset, ignoreMsg)
	} else {
		fmt.Printf("  %sWARN%s  %s\n", c.Yellow, c.Reset, ignoreMsg)
		warnings++
	}

	var trackedSensitive []string
	var leaks []leakFinding
	scannedCount := 0

	if staged {
		stagedFiles := gitStagedFiles(scanRoot)
		for _, relPosix := range stagedFiles {
			if isSensitivePath(relPosix) {
				trackedSensitive = append(trackedSensitive, relPosix)
			}
		}
		if len(trackedSensitive) > 0 {
			fmt.Printf("  %sFAIL%s  Sensitive file(s) staged for commit (%d):\n", c.Red, c.Reset, len(trackedSensitive))
			for _, rel := range trackedSensitive {
				fmt.Printf("        - %s%s%s\n", c.Cyan, rel, c.Reset)
			}
			failures += len(trackedSensitive)
		} else {
			fmt.Printf("  %sOK%s    No sensitive files staged for commit\n", c.Green, c.Reset)
		}

		for _, relPosix := range stagedFiles {
			leaks = append(leaks, scanStagedBlob(scanRoot, relPosix)...)
		}
		scannedCount = len(stagedFiles)
	} else if inGit {
		tracked := gitTrackedFiles(root)
		for _, p := range tracked {
			if isSensitivePath(p) {
				rel, err := filepath.Rel(root, p)
				if err != nil {
					rel = p
				}
				trackedSensitive = append(trackedSensitive, filepath.ToSlash(rel))
			}
		}
		if len(trackedSensitive) > 0 {
			fmt.Printf("  %sFAIL%s  Sensitive file(s) tracked by git (%d):\n", c.Red, c.Reset, len(trackedSensitive))
			for _, rel := range trackedSensitive {
				fmt.Printf("        - %s%s%s\n", c.Cyan, rel, c.Reset)
			}
			failures += len(trackedSensitive)
		} else {
			fmt.Printf("  %sOK%s    No sensitive files tracked by git\n", c.Green, c.Reset)
		}

		for _, p := range tracked {
			leaks = append(leaks, scanFile(p, root)...)
		}
		scannedCount = len(tracked)
	} else {
		fmt.Printf("  %sNot a git repository; scanning working directory files%s\n", c.Dim, c.Reset)
		candidates := walkProjectFiles(root)
		for _, p := range candidates {
			leaks = append(leaks, scanFile(p, root)...)
		}
		scannedCount = len(candidates)
	}

	if len(leaks) > 0 {
		fmt.Printf("  %sFAIL%s  Potential hardcoded secret(s) found (%d):\n", c.Red, c.Reset, len(leaks))
		for _, lk := range leaks {
			fmt.Printf("        - %s%s:%d%s (%s)\n", c.Cyan, lk.Rel, lk.Line, c.Reset, lk.Label)
		}
		failures += len(leaks)
	} else {
		fmt.Printf("  %sOK%s    No hardcoded secrets detected (%d files scanned)\n", c.Green, c.Reset, scannedCount)
	}

	fmt.Println("")
	if failures > 0 {
		fmt.Printf("%sSummary:%s %d issue(s), %d warning(s)\n", c.Red, c.Reset, failures, warnings)
		var steps []string
		if len(trackedSensitive) > 0 {
			if staged {
				steps = append(steps, "git reset HEAD <file>    Unstage sensitive file")
			} else {
				steps = append(steps, "git rm --cached <file>   Untrack sensitive file without deleting it locally")
			}
		}
		if len(leaks) > 0 {
			steps = append(steps, "Move secrets to .env, or add '# check-secrets: ignore' if false positive")
		}
		ui.NextSteps(steps)
		return 1
	}

	fmt.Printf("%sSummary:%s clean (%d warning(s))\n", c.Green, c.Reset, warnings)
	return 0
}
