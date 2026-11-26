//go:build windows

package clean

import "testing"

func TestWindowsSystemStats(t *testing.T) {
	usedGB, availGB, availKB, totalKB, pctUsed := windowsDiskStats(t.TempDir())
	if totalKB == 0 || availKB == 0 || availGB <= 0 || usedGB < 0 {
		t.Fatalf("unexpected windowsDiskStats: used=%.2f avail=%.2f availKB=%d totalKB=%d", usedGB, availGB, availKB, totalKB)
	}
	if pctUsed < 0 || pctUsed > 100 {
		t.Fatalf("unexpected pctUsed: %.2f", pctUsed)
	}

	usedMB, availMB := windowsMemStats()
	if availMB <= 0 || usedMB < 0 {
		t.Fatalf("unexpected windowsMemStats: used=%d avail=%d", usedMB, availMB)
	}
}
