package blockchain

import (
	"math/big"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

func TestCompetingChainSelectsGreatestWorkAndUsesUndo(t *testing.T) {
	chain := New()
	activeMiner := testMinerAddress(t)
	sideMiner := testMinerAddress(t)

	active1, err := chain.Append(
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
	activeWork := chain.Chainwork()

	genesis, ok := chain.BlockAt(0)
	if !ok {
		t.Fatal("missing genesis")
	}
	side1 := mineBranchBlock(t, genesis, sideMiner, config.GenesisTimestamp+60)
	if err := chain.AddBlock(side1); err != nil {
		t.Fatal(err)
	}
	if chain.Tip().BlockHash != active2.BlockHash {
		t.Fatal("shorter side branch unexpectedly became active")
	}

	side2 := mineBranchBlock(t, side1, sideMiner, config.GenesisTimestamp+120)
	if err := chain.AddBlock(side2); err != nil {
		t.Fatal(err)
	}
	if chain.Tip().BlockHash != active2.BlockHash {
		t.Fatal("equal-chainwork branch replaced current tip")
	}
	if _, ok := chain.BlockByHash(side2.BlockHash); !ok {
		t.Fatal("side-branch block was not retained")
	}

	side3 := mineBranchBlock(t, side2, sideMiner, config.GenesisTimestamp+180)
	if err := chain.AddBlock(side3); err != nil {
		t.Fatal(err)
	}
	if chain.Tip().BlockHash != side3.BlockHash || chain.Height() != 3 {
		t.Fatalf("tip after reorg=%s height=%d", chain.Tip().BlockHash, chain.Height())
	}
	currentWork := new(big.Int)
	if _, ok := currentWork.SetString(chain.Chainwork(), 16); !ok {
		t.Fatalf("invalid chainwork %q", chain.Chainwork())
	}
	previousWork := new(big.Int)
	if _, ok := previousWork.SetString(activeWork, 16); !ok {
		t.Fatalf("invalid old chainwork %q", activeWork)
	}
	if currentWork.Cmp(previousWork) <= 0 {
		t.Fatalf("new chainwork=%s old=%s", currentWork, previousWork)
	}

	activeBalance, err := chain.Balance(activeMiner)
	if err != nil {
		t.Fatal(err)
	}
	if activeBalance != 0 {
		t.Fatalf("disconnected active miner balance=%d want=0", activeBalance)
	}
	sideBalance, err := chain.Balance(sideMiner)
	if err != nil {
		t.Fatal(err)
	}
	if sideBalance != 3*config.InitialMiningReward {
		t.Fatalf("side miner balance=%d want=%d", sideBalance, 3*config.InitialMiningReward)
	}

	for _, old := range []*block.Block{active1, active2} {
		if stored, ok := chain.BlockByHash(old.BlockHash); !ok || stored.BlockHash != old.BlockHash {
			t.Fatalf("disconnected block %s not retained as side branch", old.BlockHash)
		}
	}
	if _, ok := chain.TransactionByID(active1.Transactions[0].TransactionID); ok {
		t.Fatal("disconnected coinbase still appears as confirmed")
	}
	if got, ok := chain.BlockAt(2); !ok || got.BlockHash != side2.BlockHash {
		t.Fatalf("active height 2=%v", got)
	}
}

func mineBranchBlock(t *testing.T, parent *block.Block, miner string, timestamp int64) *block.Block {
	t.Helper()
	height := parent.Height + 1
	difficulty := consensus.NextDifficulty(parent.Difficulty, parent.Timestamp, timestamp)
	coinbase := testCoinbase(t, height, miner, timestamp)
	candidate := block.New(
		height,
		parent.BlockHash,
		timestamp,
		difficulty,
		0,
		[]*transaction.Transaction{coinbase},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}
	return candidate
}
