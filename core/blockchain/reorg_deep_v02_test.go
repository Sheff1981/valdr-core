package blockchain

import (
	"math/big"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
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


func TestDeeperLowerWorkBranchDoesNotActivate(t *testing.T) {
	active := New()
	activeMiner := testMinerAddress(t)
	sideMiner := testMinerAddress(t)

	// Mine a short, high-work active branch by using the fastest clamped
	// legacy intervals. Difficulty rises 1 -> 4 -> 16.
	_, err := active.Append(
		config.GenesisTimestamp+15,
		[]*transaction.Transaction{
			testCoinbase(t, 1, activeMiner, config.GenesisTimestamp+15),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	active2, err := active.Append(
		config.GenesisTimestamp+30,
		[]*transaction.Transaction{
			testCoinbase(t, 2, activeMiner, config.GenesisTimestamp+30),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	originalTip := active2.BlockHash

	genesis, ok := active.BlockAt(0)
	if !ok {
		t.Fatal("missing genesis")
	}

	// Build a taller branch at minimum difficulty. Height alone must never win
	// against the shorter branch with greater cumulative work.
	side1 := mineBranchBlock(t, genesis, sideMiner, config.GenesisTimestamp+240)
	side2 := mineBranchBlock(t, side1, sideMiner, config.GenesisTimestamp+480)
	side3 := mineBranchBlock(t, side2, sideMiner, config.GenesisTimestamp+720)

	for i, candidate := range []*block.Block{side1, side2, side3} {
		update, err := active.AddBlockWithUpdate(candidate)
		if err != nil {
			t.Fatalf("import lower-work side block %d: %v", i+1, err)
		}
		if update.Activated {
			t.Fatalf("deeper lower-work branch activated at height %d", candidate.Height)
		}
	}

	if active.Height() != active2.Height || active.Tip().BlockHash != originalTip {
		t.Fatalf(
			"active tip/height changed to %s/%d want %s/%d",
			active.Tip().BlockHash,
			active.Height(),
			originalTip,
			active2.Height,
		)
	}
	if _, ok := active.BlockByHash(side3.BlockHash); !ok {
		t.Fatal("deeper lower-work side branch was not retained")
	}

	activeWork, ok := new(big.Int).SetString(active.Chainwork(), 16)
	if !ok {
		t.Fatalf("invalid active chainwork %q", active.Chainwork())
	}
	sideWork, err := consensus.AddWork(nil, genesis.Difficulty)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []*block.Block{side1, side2, side3} {
		sideWork, err = consensus.AddWork(sideWork, candidate.Difficulty)
		if err != nil {
			t.Fatal(err)
		}
	}
	if sideWork.Cmp(activeWork) >= 0 {
		t.Fatalf("test setup: side work=%s active work=%s", sideWork, activeWork)
	}

	activeBalance, err := active.Balance(activeMiner)
	if err != nil {
		t.Fatal(err)
	}
	if activeBalance != 2*config.InitialMiningReward {
		t.Fatalf(
			"active miner balance=%d want=%d",
			activeBalance,
			2*config.InitialMiningReward,
		)
	}
	sideBalance, err := active.Balance(sideMiner)
	if err != nil {
		t.Fatal(err)
	}
	if sideBalance != 0 {
		t.Fatalf("side-branch miner balance=%d want=0", sideBalance)
	}
}

