//go:build linux

package clean

import "testing"

func TestLinuxSystemStats(t *testing.T) {
	usedGB, availGB, availKB, totalKB, pctUsed := linuxDiskStats("/")
	if totalKB == 0 || availKB == 0 || availGB <= 0 || usedGB < 0 {
		t.Fatalf("unexpected linuxDiskStats: used=%.2f avail=%.2f availKB=%d totalKB=%d", usedGB, availGB, availKB, totalKB)
	}
	if pctUsed < 0 || pctUsed > 100 {
		t.Fatalf("unexpected pctUsed: %.2f", pctUsed)
	}
	if !sameDevice("/", "/") {
		t.Fatal("sameDevice('/', '/') must be true")
	}

	usedMB, freeMB, availMB, swapUsedMB := linuxMemStats()
	if availMB <= 0 || usedMB < 0 || freeMB < 0 || swapUsedMB < 0 {
		t.Fatalf("unexpected linuxMemStats: used=%d free=%d avail=%d swap=%d", usedMB, freeMB, availMB, swapUsedMB)
	}
}
