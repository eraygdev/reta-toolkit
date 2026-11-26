package clean

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

func printHelp() {
	fmt.Println("Usage: reta clean")
	fmt.Println("")
	fmt.Println("Safe system cleanup (pip, python caches, trash, OS/developer caches)")
	fmt.Println("with optional heavy/browser cleanup and before/after disk/RAM summary.")
}

func confirmYes(prompt string) bool {
	return ui.ConfirmYes(os.Stdout, prompt)
}

func cleanPythonCaches(home string) {
	skip := map[string]bool{
		".git": true, "node_modules": true, "venv": true, ".venv": true,
		".pyenv": true, ".conda": true, ".virtualenvs": true, ".cache": true,
	}
	_ = filepath.WalkDir(home, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if p != home && skip[name] {
				return filepath.SkipDir
			}
			if name == "__pycache__" {
				_ = os.RemoveAll(p)
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(name, ".pyc") {
			_ = os.Remove(p)
		}
		return nil
	})
}

func purgePipCache(extraDirs ...string) {
	if _, err := exec.LookPath("pip"); err == nil {
		_ = exec.Command("pip", "cache", "purge").Run()
	} else if _, err := exec.LookPath("pip3"); err == nil {
		_ = exec.Command("pip3", "cache", "purge").Run()
	}
	for _, d := range extraDirs {
		if d != "" {
			_ = os.RemoveAll(d)
		}
	}
}

func removeDirContents(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}
}

func Run(args []string) int {
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			printHelp()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "  %sERR%s   Unknown option: %s\n", ui.Colors().Red, ui.Colors().Reset, arg)
			return 1
		}
	}
	return runPlatformClean()
}
