package blockchain

import (
	"errors"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
)

func TestLocalChainGenesisBlock1Block2(t *testing.T) {
	chain := New()
	if chain.Len() != 1 {
		t.Fatalf("initial chain length = %d, want 1", chain.Len())
	}

	genesis := chain.Tip()
	if genesis == nil || genesis.Height != 0 {
		t.Fatal("chain did not start with genesis")
	}

	block1, err := chain.Append(config.GenesisTimestamp+60, nil)
	if err != nil {
		t.Fatalf("append block 1: %v", err)
	}
	block2, err := chain.Append(config.GenesisTimestamp+120, nil)
	if err != nil {
		t.Fatalf("append block 2: %v", err)
	}

	if chain.Len() != 3 {
		t.Fatalf("chain length = %d, want 3", chain.Len())
	}
	if block1.Height != 1 || block2.Height != 2 {
		t.Fatalf("heights = %d, %d; want 1, 2", block1.Height, block2.Height)
	}
	if block1.PreviousBlockHash != genesis.BlockHash {
		t.Fatal("block 1 does not link to genesis")
	}
	if block2.PreviousBlockHash != block1.BlockHash {
		t.Fatal("block 2 does not link to block 1")
	}
}

func TestRejectsBrokenPreviousHash(t *testing.T) {
	chain := New()
	candidate := block.New(1, "not-the-tip", config.GenesisTimestamp+60, 1, 0, nil, "")
	err := chain.AddBlock(candidate)
	if !errors.Is(err, ErrPreviousHash) {
		t.Fatalf("AddBlock error = %v, want ErrPreviousHash", err)
	}
}

func TestRejectsTamperedBlockHash(t *testing.T) {
	chain := New()
	candidate := block.New(1, chain.Tip().BlockHash, config.GenesisTimestamp+60, 1, 0, nil, "")
	candidate.BlockHash = "tampered"
	err := chain.AddBlock(candidate)
	if !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("AddBlock error = %v, want ErrInvalidHash", err)
	}
}
