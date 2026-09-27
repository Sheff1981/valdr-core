package blockchain

import (
	"math/big"
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


func TestTestnet2DeeperLowerWorkBranchDoesNotActivate(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}

	active, err := NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	lowWork, err := NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	activeMiner := testMinerAddress(t)
	lowWorkMiner := testMinerAddress(t)

	// Four normal-timing Testnet2 blocks retain the harder non-special target.
	activeBlocks := appendV2Blocks(t, active, profile, activeMiner, 4)

	// Five delayed blocks trigger Testnet2 min-difficulty escape. The branch is
	// taller, but each block contributes less work, so cumulative chainwork must
	// remain below the shorter active branch.
	lowWorkBlocks := make([]*block.Block, 0, 5)
	for height := 1; height <= 5; height++ {
		timestamp := profile.GenesisTimestamp +
			int64(height)*profile.MinDifficultyAfterSeconds
		coinbase, err := transaction.NewCoinbaseForChain(
			profile.ChainID,
			uint64(height),
			lowWorkMiner,
			profile.InitialSubsidyVDR*config.AtomicUnitsPerVDR,
			timestamp,
		)
		if err != nil {
			t.Fatalf("low-work coinbase height %d: %v", height, err)
		}
		candidate, err := lowWork.Append(
			timestamp,
			[]*transaction.Transaction{coinbase},
		)
		if err != nil {
			t.Fatalf("append low-work height %d: %v", height, err)
		}
		lowWorkBlocks = append(lowWorkBlocks, candidate)
	}

	activeWork, ok := new(big.Int).SetString(active.Chainwork(), 16)
	if !ok {
		t.Fatalf("invalid active chainwork %q", active.Chainwork())
	}
	lowWorkTotal, ok := new(big.Int).SetString(lowWork.Chainwork(), 16)
	if !ok {
		t.Fatalf("invalid low-work chainwork %q", lowWork.Chainwork())
	}
	if lowWork.Height() <= active.Height() {
		t.Fatalf("test setup: low-work height=%d active=%d", lowWork.Height(), active.Height())
	}
	if lowWorkTotal.Cmp(activeWork) >= 0 {
		t.Fatalf(
			"test setup: deeper branch work=%s must be lower than active=%s",
			lowWorkTotal,
			activeWork,
		)
	}

	originalTip := active.Tip().BlockHash
	for i, candidate := range lowWorkBlocks {
		update, err := active.AddBlockWithUpdate(candidate)
		if err != nil {
			t.Fatalf("import low-work block %d: %v", i+1, err)
		}
		if update.Activated {
			t.Fatalf("deeper lower-work branch activated at height %d", candidate.Height)
		}
	}
	if active.Tip().BlockHash != originalTip || active.Height() != uint64(len(activeBlocks)) {
		t.Fatalf(
			"active tip/height changed to %s/%d want %s/%d",
			active.Tip().BlockHash,
			active.Height(),
			originalTip,
			len(activeBlocks),
		)
	}
	if _, ok := active.BlockByHash(lowWorkBlocks[len(lowWorkBlocks)-1].BlockHash); !ok {
		t.Fatal("deeper lower-work side branch was not retained")
	}

	activeBalance, err := active.Balance(activeMiner)
	if err != nil {
		t.Fatal(err)
	}
	wantActive := uint64(len(activeBlocks)) *
		profile.InitialSubsidyVDR * config.AtomicUnitsPerVDR
	if activeBalance != wantActive {
		t.Fatalf("active miner balance=%d want=%d", activeBalance, wantActive)
	}
	lowWorkBalance, err := active.Balance(lowWorkMiner)
	if err != nil {
		t.Fatal(err)
	}
	if lowWorkBalance != 0 {
		t.Fatalf("side-branch miner balance=%d want=0", lowWorkBalance)
	}
}
