package blockchain

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

func TestAddBlockWithUpdateReportsReorgBranches(t *testing.T) {
	chain := New()
	activeMiner := testMinerAddress(t)
	sideMiner := testMinerAddress(t)

	block1, err := chain.Append(
		config.GenesisTimestamp+60,
		[]*transaction.Transaction{
			testCoinbase(t, 1, activeMiner, config.GenesisTimestamp+60),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	active2, err := chain.Append(
		config.GenesisTimestamp+120,
		[]*transaction.Transaction{
			testCoinbase(t, 2, activeMiner, config.GenesisTimestamp+120),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	side2 := mineBranchBlock(t, block1, sideMiner, config.GenesisTimestamp+120)
	update, err := chain.AddBlockWithUpdate(side2)
	if err != nil {
		t.Fatal(err)
	}
	if update.Activated {
		t.Fatal("equal-work side block unexpectedly activated")
	}

	side3 := mineBranchBlock(t, side2, sideMiner, config.GenesisTimestamp+180)
	update, err = chain.AddBlockWithUpdate(side3)
	if err != nil {
		t.Fatal(err)
	}
	if !update.Activated {
		t.Fatal("heavier side branch did not report activation")
	}
	if len(update.Disconnected) != 1 ||
		update.Disconnected[0].BlockHash != active2.BlockHash {
		t.Fatalf("disconnected=%v want active block 2", blockHashes(update.Disconnected))
	}
	if len(update.Connected) != 2 ||
		update.Connected[0].BlockHash != side2.BlockHash ||
		update.Connected[1].BlockHash != side3.BlockHash {
		t.Fatalf("connected=%v want side2,side3", blockHashes(update.Connected))
	}
}

func blockHashes(blocks []*block.Block) []string {
	hashes := make([]string, 0, len(blocks))
	for _, candidate := range blocks {
		if candidate == nil {
			hashes = append(hashes, "<nil>")
			continue
		}
		hashes = append(hashes, candidate.BlockHash)
	}
	return hashes
}
