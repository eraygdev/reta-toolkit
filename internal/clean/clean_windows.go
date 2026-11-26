//go:build windows

package clean

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/eraygdev/reta-toolkit/internal/ui"
	"golang.org/x/sys/windows"
)

type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func windowsDiskStats(path string) (usedGB, availGB float64, availKB, totalKB uint64, pctUsed float64) {
	p16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	var freeBytes, totalBytes, availBytes uint64
	if err := windows.GetDiskFreeSpaceEx(p16, &freeBytes, &totalBytes, &availBytes); err != nil || totalBytes == 0 {
		return
	}
	usedBytes := totalBytes - availBytes
	totalKB = totalBytes / 1024
	availKB = freeBytes / 1024
	usedGB = float64(usedBytes) / (1024 * 1024 * 1024)
	availGB = float64(freeBytes) / (1024 * 1024 * 1024)
	pctUsed = (float64(usedBytes) / float64(totalBytes)) * 100
	return
}

func windowsMemStats() (usedMB, availMB int64) {
	k32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := k32.NewProc("GlobalMemoryStatusEx")
	var ms memoryStatusEx
	ms.dwLength = uint32(unsafe.Sizeof(ms))
	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&ms)))
	if ret == 0 {
		return 0, 0
	}
	availMB = int64(ms.ullAvailPhys / (1024 * 1024))
	usedMB = int64((ms.ullTotalPhys - ms.ullAvailPhys) / (1024 * 1024))
	return usedMB, availMB
}

func runPlatformClean() int {
	c := ui.Colors()
	home, _ := os.UserHomeDir()
	localAppData := os.Getenv("LOCALAPPDATA")
	fmt.Println("reta clean")
	fmt.Println("")

	usedBefore, availBefore, beforeAvailKB, totalKB, pctBefore := windowsDiskStats(home)
	memUsedBefore, memAvailBefore := windowsMemStats()

	fmt.Printf("  %sBefore Cleanup:%s\n", c.Bold, c.Reset)
	fmt.Printf("    Disk:         Used: %.1fG / Free: %.1fG (%.0f%% used)\n", usedBefore, availBefore, pctBefore)
	fmt.Printf("    RAM:          Used: %dMi / Available: %dMi\n", memUsedBefore, memAvailBefore)
	fmt.Println("")

	fmt.Printf("  %s── Safe Cleanup ──%s\n", c.Yellow, c.Reset)

	fmt.Println("  [1/4] Purging pip cache...")
	pipLocal := ""
	if localAppData != "" {
		pipLocal = filepath.Join(localAppData, "pip", "cache")
	}
	purgePipCache(pipLocal, filepath.Join(home, ".cache", "pip"))
	fmt.Printf("  %sOK%s    pip cache cleared\n", c.Green, c.Reset)

	fmt.Println("  [2/4] Removing Python cache files...")
	cleanPythonCaches(home)
	fmt.Printf("  %sOK%s    Python caches removed\n", c.Green, c.Reset)

	fmt.Println("  [3/4] Cleaning temporary files...")
	if tmpDir := os.TempDir(); tmpDir != "" {
		removeDirContents(tmpDir)
	}
	fmt.Printf("  %sOK%s    Temp folder cleaned\n", c.Green, c.Reset)

	fmt.Println("  [4/4] Cleaning developer caches...")
	_ = os.RemoveAll(filepath.Join(home, ".cache", "pre-commit"))
	if localAppData != "" {
		_ = os.RemoveAll(filepath.Join(localAppData, "go-build"))
	}
	fmt.Printf("  %sOK%s    Developer caches cleared\n", c.Green, c.Reset)
	fmt.Println("")

	fmt.Printf("  %s── Google Chrome Cache ──%s\n", c.Yellow, c.Reset)
	ui.PrintDim(os.Stdout, "Note: Running Chrome instances will be terminated.")
	if confirmYes("  Clean Google Chrome cache? [y/N] ") && localAppData != "" {
		fmt.Println("")
		_ = exec.Command("taskkill", "/F", "/IM", "chrome.exe").Run()
		time.Sleep(1 * time.Second)
		chromeData := filepath.Join(localAppData, "Google", "Chrome", "User Data")
		profiles, _ := filepath.Glob(filepath.Join(chromeData, "Profile *"))
		profiles = append([]string{filepath.Join(chromeData, "Default")}, profiles...)
		for _, prof := range profiles {
			if info, err := os.Stat(prof); err == nil && info.IsDir() {
				removeDirContents(filepath.Join(prof, "Cache"))
				removeDirContents(filepath.Join(prof, "Code Cache"))
				removeDirContents(filepath.Join(prof, "GPUCache"))
			}
		}
		fmt.Printf("  %sOK%s    Chrome cache cleared\n", c.Green, c.Reset)
	} else {
		fmt.Println("")
		ui.PrintDim(os.Stdout, "Chrome cache cleanup skipped.")
	}
	fmt.Println("")

	usedAfter, availAfter, afterAvailKB, _, pctAfter := windowsDiskStats(home)
	memUsedAfter, memAvailAfter := windowsMemStats()
	freedKB := int64(afterAvailKB) - int64(beforeAvailKB)

	fmt.Printf("  %sAfter Cleanup:%s\n", c.Bold, c.Reset)
	fmt.Printf("    Disk:         Used: %.1fG / Free: %.1fG (%.0f%% used)\n", usedAfter, availAfter, pctAfter)
	fmt.Printf("    RAM:          Used: %dMi / Available: %dMi (Delta: %+d Mi)\n", memUsedAfter, memAvailAfter, memAvailAfter-memAvailBefore)
	fmt.Println("")
	if freedKB > 0 && totalKB > 0 {
		pct := (float64(freedKB) / float64(totalKB)) * 100
		fmt.Printf("    • Disk Space Freed : %s%.1f MB%s (%.2f%%)\n\n", c.Green, float64(freedKB)/1024, c.Reset, pct)
	} else {
		fmt.Print("    • Disk Space Freed : 0 B (No net change)\n\n")
	}

	fmt.Printf("  %sOK%s    System cleanup completed.\n", c.Green, c.Reset)
	ui.NextSteps([]string{"reta doctor      Verify system health"})
	return 0
}
