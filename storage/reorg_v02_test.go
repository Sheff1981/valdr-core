package storage_test

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/storage"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestBadgerPersistsSideBranchesAndAtomicReorg(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewBadgerStore(dir, config.ChainID, config.GenesisBlockHash)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := blockchain.NewPersistent(store)
	if err != nil {
		t.Fatal(err)
	}

	activeMiner, err := wallet.New("active-miner")
	if err != nil {
		t.Fatal(err)
	}
	sideMiner, err := wallet.New("side-miner")
	if err != nil {
		t.Fatal(err)
	}

	active1, err := mining.MineBlock(
		chain,
		activeMiner.Address,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	active2, err := mining.MineBlock(
		chain,
		activeMiner.Address,
		config.GenesisTimestamp+120,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	genesis, ok := chain.BlockAt(0)
	if !ok {
		t.Fatal("missing genesis")
	}
	side1 := mineStoredBranchBlock(t, genesis, sideMiner.Address, config.GenesisTimestamp+60)
	if err := chain.AddBlock(side1); err != nil {
		t.Fatal(err)
	}
	side2 := mineStoredBranchBlock(t, side1, sideMiner.Address, config.GenesisTimestamp+120)
	if err := chain.AddBlock(side2); err != nil {
		t.Fatal(err)
	}
	if chain.Tip().BlockHash != active2.BlockHash {
		t.Fatal("equal-work stored branch unexpectedly became active")
	}

	side3 := mineStoredBranchBlock(t, side2, sideMiner.Address, config.GenesisTimestamp+180)
	if err := chain.AddBlock(side3); err != nil {
		t.Fatal(err)
	}
	if chain.Tip().BlockHash != side3.BlockHash {
		t.Fatal("heavier branch did not become active")
	}

	info, err := store.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info.Height != 3 ||
		info.ActiveTip != side3.BlockHash ||
		info.Chainwork != chain.Chainwork() ||
		info.UTXOHash != storage.HashUTXOSet(chain.UTXOSnapshot()) {
		t.Fatalf("storage info after reorg=%+v chainwork=%s", info, chain.Chainwork())
	}

	for prefix, want := range map[string]int{
		"block/":  6,
		"header/": 6,
		"undo/":   5,
		"height/": 4,
		"tx/":     3,
		"utxo/":   3,
	} {
		got, err := store.KeyCount(prefix)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s key count=%d want=%d", prefix, got, want)
		}
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopenedStore, err := storage.NewBadgerStore(dir, config.ChainID, config.GenesisBlockHash)
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedStore.Close()
	reopened, err := blockchain.NewPersistent(reopenedStore)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Height() != 3 ||
		reopened.Tip().BlockHash != side3.BlockHash ||
		reopened.Chainwork() != chain.Chainwork() {
		t.Fatalf(
			"reopened tip/height/work=%s/%d/%s",
			reopened.Tip().BlockHash,
			reopened.Height(),
			reopened.Chainwork(),
		)
	}

	activeBalance, err := reopened.Balance(activeMiner.Address)
	if err != nil {
		t.Fatal(err)
	}
	if activeBalance != 0 {
		t.Fatalf("old active miner balance=%d want=0", activeBalance)
	}
	sideBalance, err := reopened.Balance(sideMiner.Address)
	if err != nil {
		t.Fatal(err)
	}
	if sideBalance != 3*config.InitialMiningReward {
		t.Fatalf("side miner balance=%d want=%d", sideBalance, 3*config.InitialMiningReward)
	}
	for _, old := range []*block.Block{active1, active2} {
		if _, ok := reopened.BlockByHash(old.BlockHash); !ok {
			t.Fatalf("disconnected branch block %s missing after restart", old.BlockHash)
		}
	}
}

func mineStoredBranchBlock(
	t *testing.T,
	parent *block.Block,
	miner string,
	timestamp int64,
) *block.Block {
	t.Helper()
	height := parent.Height + 1
	coinbase, err := transaction.NewCoinbase(
		height,
		miner,
		consensus.BlockReward(height),
		timestamp,
	)
	if err != nil {
		t.Fatal(err)
	}
	candidate := block.New(
		height,
		parent.BlockHash,
		timestamp,
		consensus.NextDifficulty(parent.Difficulty, parent.Timestamp, timestamp),
		0,
		[]*transaction.Transaction{coinbase},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}
	return candidate
}
