package rpc

import (
	"testing"

	"github.com/Sheff1981/valdr-core/p2p"
)

func TestCalculateSyncProgress(t *testing.T) {
	best, progress := calculateSyncProgress(
		50,
		[]p2p.Peer{
			{Height: 40},
			{Height: 100},
			{Height: 75},
		},
	)
	if best != 100 || progress != 0.5 {
		t.Fatalf("best=%d progress=%f want 100/0.5", best, progress)
	}

	best, progress = calculateSyncProgress(12, nil)
	if best != 12 || progress != 1 {
		t.Fatalf("no-peer best=%d progress=%f want 12/1", best, progress)
	}

	best, progress = calculateSyncProgress(0, nil)
	if best != 0 || progress != 1 {
		t.Fatalf("genesis best=%d progress=%f want 0/1", best, progress)
	}
}
