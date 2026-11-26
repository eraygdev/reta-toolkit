//go:build darwin

package clean

import "testing"

func TestDarwinSystemStats(t *testing.T) {
	usedGB, availGB, availKB, totalKB, pctUsed := darwinDiskStats("/")
	if totalKB == 0 || availKB == 0 || availGB <= 0 || usedGB < 0 {
		t.Fatalf("unexpected darwinDiskStats: used=%.2f avail=%.2f availKB=%d totalKB=%d", usedGB, availGB, availKB, totalKB)
	}
	if pctUsed < 0 || pctUsed > 100 {
		t.Fatalf("unexpected pctUsed: %.2f", pctUsed)
	}
}
