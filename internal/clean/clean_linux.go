//go:build linux

package clean

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/eraygdev/reta-toolkit/internal/ui"
)

func linuxDiskStats(path string) (usedGB, availGB float64, availKB, totalKB uint64, pctUsed float64) {
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

func sameDevice(a, b string) bool {
	var sa, sb syscall.Stat_t
	if syscall.Stat(a, &sa) != nil || syscall.Stat(b, &sb) != nil {
		return true
	}
	return sa.Dev == sb.Dev
}

func linuxMemStats() (usedMB, freeMB, availMB, swapUsedMB int64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return
	}
	vals := make(map[string]int64)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			if n, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				vals[key] = n
			}
		}
	}
	totalKB := vals["MemTotal"]
	freeKB := vals["MemFree"]
	availKB := vals["MemAvailable"]
	swapTotal := vals["SwapTotal"]
	swapFree := vals["SwapFree"]
	usedMB = (totalKB - availKB) / 1024
	freeMB = freeKB / 1024
	availMB = availKB / 1024
	swapUsedMB = (swapTotal - swapFree) / 1024
	return
}

func runPlatformClean() int {
	c := ui.Colors()
	home, _ := os.UserHomeDir()
	fmt.Println("reta clean")
	fmt.Println("")

	hasSudo := false
	ui.PrintDim(os.Stdout, "Requesting administrative privileges...")
	sudoCmd := exec.Command("sudo", "-v")
	sudoCmd.Stdin = os.Stdin
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	if sudoCmd.Run() == nil {
		hasSudo = true
	} else {
		fmt.Printf("  %sWARN%s  Sudo access unavailable; skipping system-level cleanup steps.\n", c.Yellow, c.Reset)
	}
	fmt.Println("")

	sameFS := sameDevice("/", home)
	rUsedBefore, rAvailBefore, rootBeforeAvailKB, rootTotalKB, rPctBefore := linuxDiskStats("/")
	hUsedBefore, hAvailBefore, homeBeforeAvailKB, homeTotalKB, hPctBefore := 0.0, 0.0, uint64(0), uint64(0), 0.0
	if !sameFS {
		hUsedBefore, hAvailBefore, homeBeforeAvailKB, homeTotalKB, hPctBefore = linuxDiskStats(home)
	}
	memUsedBefore, memFreeBefore, memAvailBefore, _ := linuxMemStats()

	fmt.Printf("  %sBefore Cleanup:%s\n", c.Bold, c.Reset)
	fmt.Printf("    Disk (/):     Used: %.1fG / Free: %.1fG (%.0f%% used)\n", rUsedBefore, rAvailBefore, rPctBefore)
	if !sameFS {
		fmt.Printf("    Disk (HOME):  Used: %.1fG / Free: %.1fG (%.0f%% used)\n", hUsedBefore, hAvailBefore, hPctBefore)
	}
	fmt.Printf("    RAM:          Used: %dMi / Free: %dMi (Available: %dMi)\n", memUsedBefore, memFreeBefore, memAvailBefore)
	fmt.Println("")

	fmt.Printf("  %s── Safe Cleanup ──%s\n", c.Yellow, c.Reset)

	fmt.Println("  [1/9] Purging pip cache...")
	purgePipCache(filepath.Join(home, ".cache", "pip"))
	fmt.Printf("  %sOK%s    pip cache cleared\n", c.Green, c.Reset)

	fmt.Println("  [2/9] Cleaning apt package cache...")
	if hasSudo {
		if _, err := exec.LookPath("apt-get"); err == nil {
			_ = exec.Command("sudo", "apt-get", "clean").Run()
			_ = exec.Command("sudo", "apt-get", "autoclean").Run()
			fmt.Printf("  %sOK%s    apt cache cleaned\n", c.Green, c.Reset)
			if confirmYes("    Remove unused apt dependencies (autoremove)? [y/N] ") {
				_ = exec.Command("sudo", "apt-get", "autoremove", "-y").Run()
				fmt.Printf("  %sOK%s    Unused apt packages removed\n", c.Green, c.Reset)
			} else {
				ui.PrintDim(os.Stdout, "apt autoremove skipped.")
			}
		} else {
			ui.PrintDim(os.Stdout, "apt cleanup skipped.")
		}
	} else {
		ui.PrintDim(os.Stdout, "apt cleanup skipped.")
	}

	fmt.Println("  [3/9] Removing Python cache files...")
	cleanPythonCaches(home)
	fmt.Printf("  %sOK%s    Python caches removed\n", c.Green, c.Reset)

	fmt.Println("  [4/9] User trash bin...")
	trashDir := filepath.Join(home, ".local", "share", "Trash")
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

	fmt.Println("  [5/9] Vacuuming system journal logs...")
	if hasSudo {
		if _, err := exec.LookPath("journalctl"); err == nil {
			_ = exec.Command("sudo", "journalctl", "--vacuum-time=3d").Run()
			fmt.Printf("  %sOK%s    Journal logs trimmed\n", c.Green, c.Reset)
		} else {
			ui.PrintDim(os.Stdout, "Journal cleanup skipped.")
		}
	} else {
		ui.PrintDim(os.Stdout, "Journal cleanup skipped.")
	}

	fmt.Println("  [6/9] Removing disabled snap revisions...")
	if hasSudo {
		if _, err := exec.LookPath("snap"); err == nil {
			if out, err := exec.Command("snap", "list", "--all").Output(); err == nil {
				for _, line := range strings.Split(string(out), "\n") {
					if !strings.Contains(line, "disabled") {
						continue
					}
					fields := strings.Fields(line)
					if len(fields) >= 3 {
						_ = exec.Command("sudo", "snap", "remove", fields[0], "--revision="+fields[2]).Run()
					}
				}
			}
			fmt.Printf("  %sOK%s    Old snap revisions removed\n", c.Green, c.Reset)
		} else {
			ui.PrintDim(os.Stdout, "Snap cleanup skipped.")
		}
	} else {
		ui.PrintDim(os.Stdout, "Snap cleanup skipped.")
	}

	fmt.Println("  [7/9] Clearing thumbnail cache...")
	removeDirContents(filepath.Join(home, ".cache", "thumbnails"))
	fmt.Printf("  %sOK%s    Thumbnail cache cleared\n", c.Green, c.Reset)

	fmt.Println("  [8/9] Cleaning developer caches...")
	_ = os.RemoveAll(filepath.Join(home, ".cache", "pre-commit"))
	_ = os.RemoveAll(filepath.Join(home, ".cache", "go-build"))
	fmt.Printf("  %sOK%s    Developer caches cleared\n", c.Green, c.Reset)

	fmt.Println("  [9/9] Syncing disk buffers...")
	syscall.Sync()
	fmt.Printf("  %sOK%s    Disk synced\n", c.Green, c.Reset)
	fmt.Println("")

	if hasSudo {
		fmt.Printf("  %s── Heavy Operations (Optional) ──%s\n", c.Yellow, c.Reset)
		ui.PrintDim(os.Stdout, "Actions: Drop kernel page cache, Reset swap space")
		if confirmYes("  Perform heavy operations? [y/N] ") {
			fmt.Println("")
			fmt.Println("  Dropping kernel page cache...")
			_ = exec.Command("sudo", "sh", "-c", "echo 3 > /proc/sys/vm/drop_caches").Run()
			fmt.Printf("  %sOK%s    Kernel cache dropped\n", c.Green, c.Reset)

			fmt.Println("  Resetting swap space...")
			_, _, availRAM, usedSwap := linuxMemStats()
			if availRAM > usedSwap {
				if exec.Command("sudo", "swapoff", "-a").Run() == nil {
					_ = exec.Command("sudo", "swapon", "-a").Run()
				}
				fmt.Printf("  %sOK%s    Swap space reset successfully\n", c.Green, c.Reset)
			} else {
				fmt.Printf("  %sWARN%s  Available RAM (%d MB) < Used Swap (%d MB)\n", c.Yellow, c.Reset, availRAM, usedSwap)
				fmt.Printf("  %sWARN%s  Skipped swap reset to prevent potential OOM risks.\n", c.Yellow, c.Reset)
			}
		} else {
			fmt.Println("")
			ui.PrintDim(os.Stdout, "Heavy operations skipped.")
		}
		fmt.Println("")
	}

	fmt.Printf("  %s── Google Chrome Cache ──%s\n", c.Yellow, c.Reset)
	ui.PrintDim(os.Stdout, "Note: Running Chrome instances will be terminated.")
	if confirmYes("  Clean Google Chrome cache? [y/N] ") {
		fmt.Println("")
		fmt.Println("  Closing Chrome and clearing caches...")
		if exec.Command("pgrep", "-x", "chrome").Run() == nil {
			_ = exec.Command("pkill", "-x", "chrome").Run()
			time.Sleep(2 * time.Second)
		}
		removeDirContents(filepath.Join(home, ".cache", "google-chrome"))
		chromeCfg := filepath.Join(home, ".config", "google-chrome")
		profiles, _ := filepath.Glob(filepath.Join(chromeCfg, "Profile *"))
		profiles = append([]string{filepath.Join(chromeCfg, "Default")}, profiles...)
		for _, prof := range profiles {
			if info, err := os.Stat(prof); err == nil && info.IsDir() {
				removeDirContents(filepath.Join(prof, "Cache"))
				removeDirContents(filepath.Join(prof, "Code Cache"))
				removeDirContents(filepath.Join(prof, "GPUCache"))
				removeDirContents(filepath.Join(prof, "Service Worker", "CacheStorage"))
				removeDirContents(filepath.Join(prof, "Service Worker", "ScriptCache"))
			}
		}
		fmt.Printf("  %sOK%s    Chrome cache cleared\n", c.Green, c.Reset)
	} else {
		fmt.Println("")
		ui.PrintDim(os.Stdout, "Chrome cache cleanup skipped.")
	}
	fmt.Println("")

	fmt.Printf("  %s── Top 10 Memory Consuming Processes ──%s\n", c.Yellow, c.Reset)
	if out, err := exec.Command("ps", "aux", "--sort=-%mem").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for i := 0; i < len(lines) && i < 11; i++ {
			fmt.Printf("    %s\n", lines[i])
		}
	}
	fmt.Println("")

	rUsedAfter, rAvailAfter, rootAfterAvailKB, _, rPctAfter := linuxDiskStats("/")
	hUsedAfter, hAvailAfter, homeAfterAvailKB, _, hPctAfter := 0.0, 0.0, uint64(0), uint64(0), 0.0
	if !sameFS {
		hUsedAfter, hAvailAfter, homeAfterAvailKB, _, hPctAfter = linuxDiskStats(home)
	}
	memUsedAfter, memFreeAfter, memAvailAfter, _ := linuxMemStats()

	freedKB := int64(rootAfterAvailKB) - int64(rootBeforeAvailKB)
	if !sameFS {
		freedKB += int64(homeAfterAvailKB) - int64(homeBeforeAvailKB)
	}
	totalKB := int64(rootTotalKB)
	if !sameFS {
		totalKB += int64(homeTotalKB)
	}
	ramFreedMB := memAvailAfter - memAvailBefore

	fmt.Printf("  %sAfter Cleanup:%s\n", c.Bold, c.Reset)
	fmt.Printf("    Disk (/):     Used: %.1fG / Free: %.1fG (%.0f%% used)\n", rUsedAfter, rAvailAfter, rPctAfter)
	if !sameFS {
		fmt.Printf("    Disk (HOME):  Used: %.1fG / Free: %.1fG (%.0f%% used)\n", hUsedAfter, hAvailAfter, hPctAfter)
	}
	fmt.Printf("    RAM:          Used: %dMi / Free: %dMi (Available: %dMi)\n", memUsedAfter, memFreeAfter, memAvailAfter)
	fmt.Println("")

	printSummaryBox(freedKB, totalKB, ramFreedMB)
	fmt.Printf("  %sOK%s    System cleanup completed.\n", c.Green, c.Reset)
	ui.NextSteps([]string{"reta doctor      Verify system health"})
	return 0
}

func printSummaryBox(freedKB, totalKB, ramFreedMB int64) {
	c := ui.Colors()
	fmt.Printf("  %s┌────────────────────────────────────────┐%s\n", c.Bold, c.Reset)
	fmt.Printf("  %s│           CLEANUP SUMMARY              │%s\n", c.Bold, c.Reset)
	fmt.Printf("  %s└────────────────────────────────────────┘%s\n", c.Bold, c.Reset)

	if freedKB > 0 {
		var freedStr string
		if freedKB > 1048576 {
			freedStr = fmt.Sprintf("%.2f GB", float64(freedKB)/1024/1024)
		} else if freedKB > 1024 {
			freedStr = fmt.Sprintf("%.1f MB", float64(freedKB)/1024)
		} else {
			freedStr = fmt.Sprintf("%d KB", freedKB)
		}
		pct := 0.0
		if totalKB > 0 {
			pct = (float64(freedKB) / float64(totalKB)) * 100
		}
		fmt.Printf("    • Disk Space Freed : %s%s%s (%.2f%% of monitored partition(s))\n", c.Green, freedStr, c.Reset, pct)
	} else {
		fmt.Println("    • Disk Space Freed : 0 B (No net change)")
	}

	if ramFreedMB > 0 {
		fmt.Printf("    • RAM Available Gain : %s+%d MB%s\n", c.Green, ramFreedMB, c.Reset)
	} else if ramFreedMB < 0 {
		fmt.Printf("    • RAM Available Gain : %s%d MB%s\n", c.Yellow, ramFreedMB, c.Reset)
	} else {
		fmt.Println("    • RAM Available Gain : 0 MB")
	}
	fmt.Println("")
}
