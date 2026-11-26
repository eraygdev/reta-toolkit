package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	reta "github.com/eraygdev/reta-toolkit"
	"github.com/eraygdev/reta-toolkit/internal/ui"
)

func printHelp() {
	fmt.Println("Usage: reta doctor [--fix]")
	fmt.Println("")
	fmt.Println("  --fix    Automatically repair ~/.reta/lib/base.mk and installation links")
}

func isOnPath(dir string) bool {
	pathEnv := os.Getenv("PATH")
	cleanDir := filepath.Clean(dir)
	for _, entry := range filepath.SplitList(pathEnv) {
		cleaned := filepath.Clean(entry)
		if runtime.GOOS == "windows" {
			if strings.EqualFold(cleaned, cleanDir) {
				return true
			}
		} else if cleaned == cleanDir {
			return true
		}
	}
	return false
}

func EnsureBaseMK(retaRoot string) error {
	baseMKPath := filepath.Join(retaRoot, "lib", "base.mk")
	gitDir := filepath.Join(retaRoot, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		// Do not overwrite a git-managed working copy of reta-toolkit.
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(baseMKPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(baseMKPath, []byte(reta.BaseMK), 0o644)
}

func Run(args []string) int {
	fix := false
	for _, arg := range args {
		switch arg {
		case "--fix":
			fix = true
		case "-h", "--help":
			printHelp()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
			return 2
		}
	}

	c := ui.Colors()
	home, _ := os.UserHomeDir()
	retaRoot := filepath.Join(home, ".reta")

	okCount, warnCount, failCount := 0, 0, 0
	checkOK := func(msg string) {
		fmt.Printf("  %sOK%s    %s\n", c.Green, c.Reset, msg)
		okCount++
	}
	checkWarn := func(msg string) {
		fmt.Printf("  %sWARN%s  %s\n", c.Yellow, c.Reset, msg)
		warnCount++
	}
	checkFail := func(msg string) {
		fmt.Printf("  %sFAIL%s  %s\n", c.Red, c.Reset, msg)
		failCount++
	}

	fmt.Println("reta doctor")
	if fix {
		fmt.Println("Mode: AUTO-FIX enabled")
	}
	fmt.Println("")

	if fix {
		installSh := filepath.Join(retaRoot, "install.sh")
		if runtime.GOOS != "windows" {
			if _, err := os.Stat(installSh); err == nil {
				fmt.Println("Running installer to repair environment...")
				if exec.Command(installSh).Run() == nil {
					fmt.Printf("  %sOK%s    Repair completed via install.sh\n", c.Green, c.Reset)
				} else {
					fmt.Fprintf(os.Stderr, "  %sERR%s   Repair script encountered issues\n", c.Red, c.Reset)
				}
			}
		}
		if err := EnsureBaseMK(retaRoot); err == nil {
			fmt.Printf("  %sOK%s    Ensured ~/.reta/lib/base.mk\n", c.Green, c.Reset)
		} else {
			fmt.Fprintf(os.Stderr, "  %sERR%s   Could not write base.mk: %v\n", c.Red, c.Reset, err)
		}
		fmt.Println("")
	}

	fmt.Println("Binary & Paths:")
	if exePath, err := os.Executable(); err == nil {
		checkOK(fmt.Sprintf("executable: %s", exePath))
	}
	if resolved, err := exec.LookPath("reta"); err == nil {
		checkOK(fmt.Sprintf("reta on PATH (%s)", resolved))
	} else {
		if runtime.GOOS == "windows" {
			localBin := filepath.Join(os.Getenv("LOCALAPPDATA"), "reta", "bin")
			if isOnPath(localBin) {
				checkWarn("reta.exe not found in " + localBin)
			} else {
				checkFail("reta is not on PATH")
			}
		} else {
			checkFail("reta is not on PATH (~/.local/bin missing or binary not linked)")
		}
	}

	if fi, err := os.Lstat(retaRoot); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(retaRoot)
			if _, statErr := os.Stat(retaRoot); statErr == nil {
				checkOK(fmt.Sprintf("~/.reta -> %s", target))
			} else {
				checkFail("~/.reta symlink is broken")
			}
		} else if fi.IsDir() {
			checkOK("~/.reta exists")
		}
	} else {
		checkFail("~/.reta missing (run install or reta doctor --fix)")
	}

	baseMK := filepath.Join(retaRoot, "lib", "base.mk")
	if _, err := os.Stat(baseMK); err == nil {
		checkOK("lib/base.mk")
	} else {
		checkWarn("lib/base.mk missing (run 'reta doctor --fix' if using Makefile)")
	}

	fmt.Println("")
	fmt.Println("Shell Integration:")
	if runtime.GOOS == "windows" {
		profileOut, err := exec.Command("powershell", "-NoProfile", "-Command", "echo $PROFILE").Output()
		if err == nil {
			profPath := strings.TrimSpace(string(profileOut))
			if data, rErr := os.ReadFile(profPath); rErr == nil && (strings.Contains(string(data), "RETA") || strings.Contains(string(data), "function rcd")) {
				checkOK("PowerShell rcd hook configured")
			} else {
				checkWarn("PowerShell rcd hook not configured (add: Invoke-Expression (& reta shell-init powershell | Out-String))")
			}
		}
	} else {
		bashComp := filepath.Join(home, ".local", "share", "bash-completion", "completions", "reta")
		zshComp := filepath.Join(home, ".zfunc", "_reta")
		if _, err := os.Stat(bashComp); err == nil {
			checkOK(fmt.Sprintf("bash completion (%s)", bashComp))
		} else {
			checkWarn("bash completion not linked (run 'reta completion bash')")
		}
		if _, err := exec.LookPath("zsh"); err == nil {
			if _, err := os.Stat(zshComp); err == nil {
				checkOK(fmt.Sprintf("zsh completion (%s)", zshComp))
			} else {
				checkWarn("zsh completion not linked (run 'reta completion zsh')")
			}
		} else {
			ui.PrintDim(os.Stdout, "zsh not installed (skipped)")
		}

		rcdConfigured := false
		for _, rcName := range []string{".bashrc", ".zshrc"} {
			rcPath := filepath.Join(home, rcName)
			if data, err := os.ReadFile(rcPath); err == nil {
				content := string(data)
				if strings.Contains(content, "# ===RETA*===") || strings.Contains(content, "reta shell-init") {
					checkOK(fmt.Sprintf("rcd hook configured in %s", rcName))
					rcdConfigured = true
				}
			}
		}
		if !rcdConfigured {
			checkWarn("rcd hook not configured in ~/.bashrc or ~/.zshrc")
		}
	}

	fmt.Println("")
	fmt.Println("Environment:")
	if _, err := exec.LookPath("make"); err == nil {
		checkOK("make available")
	} else if runtime.GOOS == "windows" {
		checkWarn("make not found (optional on Windows; base.mk targets require GNU make)")
	} else {
		checkFail("make not found (base.mk targets won't work)")
	}

	if _, err := exec.LookPath("git"); err == nil {
		checkOK("git available")
	} else {
		checkWarn("git not found (reta hook / check-secrets git features disabled)")
	}

	if _, err := exec.LookPath("docker"); err == nil {
		checkOK("docker available")
	} else {
		checkWarn("docker not found (make up/down won't work)")
	}

	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("ss"); err == nil {
			checkOK("ss available (reta port)")
		} else if _, err := exec.LookPath("lsof"); err == nil {
			checkOK("lsof available (reta port)")
		} else {
			checkFail("neither ss nor lsof found (reta port won't work)")
		}
	case "darwin":
		if _, err := exec.LookPath("lsof"); err == nil {
			checkOK("lsof available (reta port)")
		} else {
			checkFail("lsof not found (reta port won't work)")
		}
	case "windows":
		if _, err := exec.LookPath("netstat"); err == nil {
			checkOK("netstat available (reta port)")
		} else {
			checkFail("netstat not found (reta port won't work)")
		}
	}

	fmt.Println("")
	fmt.Printf("Summary: %d OK, %d WARN, %d FAIL\n", okCount, warnCount, failCount)
	if failCount > 0 {
		return 1
	}
	return 0
}
