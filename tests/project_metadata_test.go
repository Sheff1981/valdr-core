package tests

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestProjectMetadata(t *testing.T) {
	if config.ProjectName != "VALDR" {
		t.Fatalf("ProjectName = %q, want VALDR", config.ProjectName)
	}
	if config.Ticker != "VDR" {
		t.Fatalf("Ticker = %q, want VDR", config.Ticker)
	}
	if config.AtomicUnit != "val" {
		t.Fatalf("AtomicUnit = %q, want val", config.AtomicUnit)
	}
	if config.AtomicUnitsPerVDR != 100_000_000 {
		t.Fatalf("AtomicUnitsPerVDR = %d, want 100000000", config.AtomicUnitsPerVDR)
	}
}
