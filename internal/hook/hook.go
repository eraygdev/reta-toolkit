package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eraygdev/reta-toolkit/internal/cli"
	"github.com/eraygdev/reta-toolkit/internal/ui"
)

const hookScript = `#!/usr/bin/env bash
CHECK_SECRETS="$HOME/.reta/lib/check-secrets"

run_reta_check() {
    local bin="$1"
    if "$bin" check-secrets --staged >/dev/null 2>&1; then
        exit 0
    fi
    echo "❌ [reta] Secret or sensitive file detected in commit!"
    echo "       Run 'reta check-secrets --staged' to inspect details."
    echo "       Use '# check-secrets: ignore' in-line to bypass false positives."
    exit 1
}

if command -v reta >/dev/null 2>&1 && reta check-secrets --help >/dev/null 2>&1; then
    run_reta_check "reta"
elif [ -x "$HOME/.local/bin/reta" ]; then
    run_reta_check "$HOME/.local/bin/reta"
elif [ -n "${LOCALAPPDATA:-}" ] && [ -x "$LOCALAPPDATA/reta/bin/reta.exe" ]; then
    run_reta_check "$LOCALAPPDATA/reta/bin/reta.exe"
elif [ -x "$CHECK_SECRETS" ]; then
    if "$CHECK_SECRETS" --staged >/dev/null 2>&1; then
        exit 0
    fi
    echo "❌ [reta] Secret or sensitive file detected in commit!"
    echo "       Run '$HOME/.reta/lib/check-secrets --staged' to inspect details."
    echo "       Use '# check-secrets: ignore' in-line to bypass false positives."
    exit 1
elif [ ! -e "$HOME/.reta" ]; then
    echo "❌ [reta] reta toolkit is not installed." >&2
    echo "Install: git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta && ~/.reta/install.sh" >&2
    exit 1
elif [ -f Makefile ] && grep -qs '^check-secrets:' Makefile; then
    if make check-secrets >/dev/null 2>&1; then
        exit 0
    fi
    echo "❌ [reta] Secret or sensitive file detected in commit!"
    echo "       Run 'make check-secrets' to inspect details."
    exit 1
fi

echo "❌ [reta] check-secrets not found." >&2
echo "       Run 'reta doctor --fix' to repair your reta installation." >&2
exit 1
`

func printHelp() {
	fmt.Println("Usage: reta hook [install|remove|status]")
	fmt.Println("")
	fmt.Println("Manage git pre-commit hooks for secret scanning.")
	fmt.Println("")
	fmt.Println("Actions:")
	fmt.Println("  install   Install pre-commit secret scanner (default)")
	fmt.Println("  remove    Remove reta pre-commit hook")
	fmt.Println("  status    Check whether reta pre-commit hook is active")
}

func resolveGitPaths(cwd string) (toplevel, gitDir string, ok bool) {
	topCmd := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	topOut, err := topCmd.Output()
	if err != nil {
		return "", "", false
	}
	dirCmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-common-dir")
	dirOut, err := dirCmd.Output()
	if err != nil {
		return "", "", false
	}
	topRaw := strings.TrimSpace(string(topOut))
	dirRaw := strings.TrimSpace(string(dirOut))
	if topRaw == "" || dirRaw == "" {
		return "", "", false
	}
	toplevel, err = filepath.Abs(topRaw)
	if err != nil {
		toplevel = topRaw
	}
	if !filepath.IsAbs(dirRaw) {
		gitDir = filepath.Clean(filepath.Join(cwd, dirRaw))
	} else {
		gitDir = filepath.Clean(dirRaw)
	}
	return toplevel, gitDir, true
}

func Run(args []string) error {
	parser := cli.NewParser()
	var showHelp bool
	parser.Bool(&showHelp, "help", "h")

	pos, err := parser.Parse(args)
	if err != nil {
		return err
	}
	if showHelp {
		printHelp()
		return nil
	}

	action := "install"
	if len(pos) == 1 {
		switch pos[0] {
		case "install", "remove", "status":
			action = pos[0]
		default:
			return cli.Exit(2, "invalid action '%s' (choose from install, remove, status).", pos[0])
		}
	} else if len(pos) > 1 {
		return cli.Exit(2, "unexpected argument: %s", pos[1])
	}

	c := ui.Colors()
	cwd, err := os.Getwd()
	if err != nil {
		return cli.Wrap(1, err, "")
	}

	toplevel, gitDir, ok := resolveGitPaths(cwd)
	if !ok {
		return cli.Exit(1, "Not a git repository (no .git directory found).")
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return cli.Wrap(1, err, "")
	}
	hookFile := filepath.Join(hooksDir, "pre-commit")

	switch action {
	case "install":
		if data, err := os.ReadFile(hookFile); err == nil {
			if !strings.Contains(string(data), "check-secrets") {
				fmt.Printf("%sWarning:%s A different pre-commit hook already exists.\n", c.Yellow, c.Reset)
				if !ui.ConfirmYes(os.Stdout, "Overwrite? [y/N] ") {
					fmt.Println("Cancelled.")
					return nil
				}
			}
		}
		if err := os.WriteFile(hookFile, []byte(hookScript), 0o755); err != nil {
			return cli.Wrap(1, err, "")
		}
		_ = os.Chmod(hookFile, 0o755)
		displayPath := hookFile
		if rel, err := filepath.Rel(toplevel, hookFile); err == nil {
			displayPath = filepath.ToSlash(rel)
		}
		fmt.Printf("%sOK%s Installed pre-commit secret scanner hook into %s\n", c.Green, c.Reset, displayPath)
		ui.NextSteps([]string{"git commit -m 'test'    Test the pre-commit hook protection"})
		return nil

	case "remove":
		if data, err := os.ReadFile(hookFile); err == nil && strings.Contains(string(data), "check-secrets") {
			if err := os.Remove(hookFile); err != nil {
				return cli.Wrap(1, err, "")
			}
			fmt.Printf("%sOK%s Removed pre-commit hook.\n", c.Green, c.Reset)
		} else {
			fmt.Println("No reta pre-commit hook found.")
		}
		return nil

	case "status":
		if data, err := os.ReadFile(hookFile); err == nil && strings.Contains(string(data), "check-secrets") {
			fmt.Printf("%sActive%s pre-commit hook is installed.\n", c.Green, c.Reset)
		} else {
			fmt.Printf("%sInactive%s no pre-commit hook installed.\n", c.Dim, c.Reset)
		}
		return nil
	}
	return nil
}
