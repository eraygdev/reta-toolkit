//go:build darwin

package clean

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

func darwinDiskStats(path string) (usedGB, availGB float64, availKB, totalKB uint64, pctUsed float64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil || st.Blocks == 0 {
		return 0, 0, 0, 0, 0
	}
	bsize := uint64(st.Bsize)
	totalBytes := st.Blocks * bsize
	availBytes := st.Bavail * bsize
	usedBytes := (st.Blocks - st.Bfree) * bsize
	totalKB = totalBytes / 1024
	availKB = availBytes / 1024
	usedGB = float64(usedBytes) / (1024 * 1024 * 1024)
	availGB = float64(availBytes) / (1024 * 1024 * 1024)
	pctUsed = (float64(usedBytes) / float64(totalBytes)) * 100
	return
}

func runPlatformClean() int {
	c := ui.Colors()
	home, _ := os.UserHomeDir()
	fmt.Println("reta clean")
	fmt.Println("")

	rUsedBefore, rAvailBefore, beforeAvailKB, totalKB, rPctBefore := darwinDiskStats(home)
	fmt.Printf("  %sBefore Cleanup:%s\n", c.Bold, c.Reset)
	fmt.Printf("    Disk:         Used: %.1fG / Free: %.1fG (%.0f%% used)\n", rUsedBefore, rAvailBefore, rPctBefore)
	fmt.Println("")

	fmt.Printf("  %s── Safe Cleanup ──%s\n", c.Yellow, c.Reset)

	fmt.Println("  [1/5] Purging pip cache...")
	purgePipCache(
		filepath.Join(home, ".cache", "pip"),
		filepath.Join(home, "Library", "Caches", "pip"),
	)
	fmt.Printf("  %sOK%s    pip cache cleared\n", c.Green, c.Reset)

	fmt.Println("  [2/5] Cleaning Homebrew cache...")
	if _, err := exec.LookPath("brew"); err == nil {
		_ = exec.Command("brew", "cleanup", "-s").Run()
		fmt.Printf("  %sOK%s    Homebrew cache cleaned\n", c.Green, c.Reset)
	} else {
		ui.PrintDim(os.Stdout, "Homebrew cleanup skipped.")
	}

	fmt.Println("  [3/5] Removing Python cache files...")
	cleanPythonCaches(home)
	fmt.Printf("  %sOK%s    Python caches removed\n", c.Green, c.Reset)

	fmt.Println("  [4/5] User trash bin...")
	trashDir := filepath.Join(home, ".Trash")
	if entries, err := os.ReadDir(trashDir); err == nil && len(entries) > 0 {
		if confirmYes("    Empty user trash bin permanently? [y/N] ") {
			removeDirContents(trashDir)
			fmt.Printf("  %sOK%s    Trash emptied\n", c.Green, c.Reset)
		} else {
			ui.PrintDim(os.Stdout, "Trash cleanup skipped.")
		}
	} else {
		ui.PrintDim(os.Stdout, "Trash is already empty.")
	}

	fmt.Println("  [5/5] Cleaning developer caches...")
	_ = os.RemoveAll(filepath.Join(home, ".cache", "pre-commit"))
	_ = os.RemoveAll(filepath.Join(home, ".cache", "go-build"))
	_ = os.RemoveAll(filepath.Join(home, "Library", "Caches", "go-build"))
	fmt.Printf("  %sOK%s    Developer caches cleared\n", c.Green, c.Reset)
	fmt.Println("")

	fmt.Printf("  %s── Google Chrome Cache ──%s\n", c.Yellow, c.Reset)
	ui.PrintDim(os.Stdout, "Note: Running Chrome instances will be terminated.")
	if confirmYes("  Clean Google Chrome cache? [y/N] ") {
		fmt.Println("")
		if exec.Command("pgrep", "-x", "Google Chrome").Run() == nil {
			_ = exec.Command("pkill", "-x", "Google Chrome").Run()
			time.Sleep(2 * time.Second)
		}
		removeDirContents(filepath.Join(home, "Library", "Caches", "Google", "Chrome"))
		chromeApp := filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
		profiles, _ := filepath.Glob(filepath.Join(chromeApp, "Profile *"))
		profiles = append([]string{filepath.Join(chromeApp, "Default")}, profiles...)
		for _, prof := range profiles {
			if info, err := os.Stat(prof); err == nil && info.IsDir() {
				removeDirContents(filepath.Join(prof, "Code Cache"))
				removeDirContents(filepath.Join(prof, "GPUCache"))
				removeDirContents(filepath.Join(prof, "Service Worker", "CacheStorage"))
			}
		}
		fmt.Printf("  %sOK%s    Chrome cache cleared\n", c.Green, c.Reset)
	} else {
		fmt.Println("")
		ui.PrintDim(os.Stdout, "Chrome cache cleanup skipped.")
	}
	fmt.Println("")

	fmt.Printf("  %s── Top 10 Memory Consuming Processes ──%s\n", c.Yellow, c.Reset)
	if out, err := exec.Command("ps", "aux", "-m").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for i := 0; i < len(lines) && i < 11; i++ {
			fmt.Printf("    %s\n", lines[i])
		}
	}
	fmt.Println("")

	rUsedAfter, rAvailAfter, afterAvailKB, _, rPctAfter := darwinDiskStats(home)
	freedKB := int64(afterAvailKB) - int64(beforeAvailKB)
	fmt.Printf("  %sAfter Cleanup:%s\n", c.Bold, c.Reset)
	fmt.Printf("    Disk:         Used: %.1fG / Free: %.1fG (%.0f%% used)\n", rUsedAfter, rAvailAfter, rPctAfter)
	fmt.Println("")
	if freedKB > 0 {
		pct := (float64(freedKB) / float64(totalKB)) * 100
		fmt.Printf("    • Disk Space Freed : %s%.1f MB%s (%.2f%%)\n\n", c.Green, float64(freedKB)/1024, c.Reset, pct)
	} else {
		fmt.Print("    • Disk Space Freed : 0 B (No net change)\n\n")
	}

	fmt.Printf("  %sOK%s    System cleanup completed.\n", c.Green, c.Reset)
	ui.NextSteps([]string{"reta doctor      Verify system health"})
	return 0
}
