package blockchain

import (
	"errors"
	"math/big"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

func TestDevnetV02AppendUsesExactTargetConsensus(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	miner := testMinerAddress(t)
	timestamp := profile.GenesisTimestamp + 60
	coinbase := testCoinbase(t, 1, miner, timestamp)

	candidate, err := chain.Append(
		timestamp,
		[]*transaction.Transaction{coinbase},
	)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Version != block.VersionV2 ||
		candidate.Difficulty != 0 ||
		candidate.Target != profile.GenesisTarget ||
		candidate.ChainID != profile.ChainID {
		t.Fatalf("unexpected v2 block: %+v", candidate)
	}
	target, err := consensus.ParseTargetHexV2(candidate.Target)
	if err != nil {
		t.Fatal(err)
	}
	if err := consensus.ValidatePoWTarget(candidate.BlockHash, target); err != nil {
		t.Fatalf("v2 PoW invalid: %v", err)
	}
	if chain.Height() != 1 || chain.Tip().BlockHash != candidate.BlockHash {
		t.Fatalf("tip height/hash=%d/%s", chain.Height(), chain.Tip().BlockHash)
	}
}

func TestDevnetV02RejectsUnexpectedExactTarget(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	chain, err := NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	miner := testMinerAddress(t)
	timestamp := profile.GenesisTimestamp + 60
	coinbase := testCoinbase(t, 1, miner, timestamp)

	powLimit, err := consensus.PowLimitForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	wrongTarget := new(big.Int).Div(new(big.Int).Set(powLimit), big.NewInt(2))
	wrongHex, err := consensus.TargetHexV2(wrongTarget)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := block.NewV2(
		1,
		chain.Tip().BlockHash,
		timestamp,
		wrongHex,
		0,
		[]*transaction.Transaction{coinbase},
		profile.ChainID,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := consensus.MineTarget(candidate, wrongTarget); err != nil {
		t.Fatal(err)
	}

	err = chain.AddBlock(candidate)
	if !errors.Is(err, consensus.ErrInvalidDifficultyHistory) {
		t.Fatalf("AddBlock error=%v want ErrInvalidDifficultyHistory", err)
	}
}
