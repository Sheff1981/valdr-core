package blockchain

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

func TestV2RepeatedDeepReorgPreservesBranchesAndUTXO(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	active, err := NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	branchB, err := NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	branchC, err := NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	minerA := testMinerAddress(t)
	minerB := testMinerAddress(t)
	minerC := testMinerAddress(t)

	blocksA := appendV2Blocks(t, active, profile, minerA, 4)
	blocksB := appendV2Blocks(t, branchB, profile, minerB, 5)
	blocksC := appendV2Blocks(t, branchC, profile, minerC, 6)

	if active.Height() != 4 {
		t.Fatalf("initial active height=%d want=4", active.Height())
	}

	var firstReorg ChainUpdate
	for i, candidate := range blocksB {
		update, err := active.AddBlockWithUpdate(candidate)
		if err != nil {
			t.Fatalf("import branch B block %d: %v", i+1, err)
		}
		if i < len(blocksB)-1 && update.Activated {
			t.Fatalf("branch B activated too early at height %d", candidate.Height)
		}
		if i == len(blocksB)-1 {
			firstReorg = update
		}
	}

	if !firstReorg.Activated {
		t.Fatal("first deeper branch did not activate")
	}
	if len(firstReorg.Disconnected) != 4 || len(firstReorg.Connected) != 5 {
		t.Fatalf(
			"first reorg disconnected/connected=%d/%d want=4/5",
			len(firstReorg.Disconnected),
			len(firstReorg.Connected),
		)
	}
	if active.Tip().BlockHash != blocksB[len(blocksB)-1].BlockHash {
		t.Fatalf("first reorg tip=%s want=%s", active.Tip().BlockHash, blocksB[len(blocksB)-1].BlockHash)
	}

	var secondReorg ChainUpdate
	for i, candidate := range blocksC {
		update, err := active.AddBlockWithUpdate(candidate)
		if err != nil {
			t.Fatalf("import branch C block %d: %v", i+1, err)
		}
		if i < len(blocksC)-1 && update.Activated {
			t.Fatalf("branch C activated too early at height %d", candidate.Height)
		}
		if i == len(blocksC)-1 {
			secondReorg = update
		}
	}

	if !secondReorg.Activated {
		t.Fatal("second deeper branch did not activate")
	}
	if len(secondReorg.Disconnected) != 5 || len(secondReorg.Connected) != 6 {
		t.Fatalf(
			"second reorg disconnected/connected=%d/%d want=5/6",
			len(secondReorg.Disconnected),
			len(secondReorg.Connected),
		)
	}
	if active.Height() != 6 ||
		active.Tip().BlockHash != blocksC[len(blocksC)-1].BlockHash ||
		active.Chainwork() != branchC.Chainwork() {
		t.Fatalf(
			"final height/tip/work=%d/%s/%s want=%d/%s/%s",
			active.Height(),
			active.Tip().BlockHash,
			active.Chainwork(),
			branchC.Height(),
			branchC.Tip().BlockHash,
			branchC.Chainwork(),
		)
	}

	reward := profile.InitialSubsidyVDR * config.AtomicUnitsPerVDR
	for name, miner := range map[string]string{
		"A": minerA,
		"B": minerB,
	} {
		balance, err := active.Balance(miner)
		if err != nil {
			t.Fatal(err)
		}
		if balance != 0 {
			t.Fatalf("disconnected miner %s balance=%d want=0", name, balance)
		}
	}
	balanceC, err := active.Balance(minerC)
	if err != nil {
		t.Fatal(err)
	}
	if balanceC != uint64(len(blocksC))*reward {
		t.Fatalf("active miner C balance=%d want=%d", balanceC, uint64(len(blocksC))*reward)
	}

	for _, disconnected := range append(blocksA, blocksB...) {
		stored, ok := active.BlockByHash(disconnected.BlockHash)
		if !ok || stored.BlockHash != disconnected.BlockHash {
			t.Fatalf("disconnected block %s was not retained", disconnected.BlockHash)
		}
	}
}

func appendV2Blocks(
	t *testing.T,
	chain *Blockchain,
	profile config.NetworkProfile,
	miner string,
	count int,
) []*block.Block {
	t.Helper()

	result := make([]*block.Block, 0, count)
	for height := 1; height <= count; height++ {
		timestamp := profile.GenesisTimestamp +
			int64(height)*profile.TargetBlockTimeSeconds
		coinbase, err := transaction.NewCoinbaseForChain(
			profile.ChainID,
			uint64(height),
			miner,
			profile.InitialSubsidyVDR*config.AtomicUnitsPerVDR,
			timestamp,
		)
		if err != nil {
			t.Fatalf("coinbase height %d: %v", height, err)
		}
		candidate, err := chain.Append(
			timestamp,
			[]*transaction.Transaction{coinbase},
		)
		if err != nil {
			t.Fatalf("append height %d: %v", height, err)
		}
		result = append(result, candidate)
	}
	return result
}
