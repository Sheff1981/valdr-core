package blockchain

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
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
	if err := consensus.ValidatePoW(block1); err != nil {
		t.Fatalf("block 1 PoW: %v", err)
	}
	if err := consensus.ValidatePoW(block2); err != nil {
		t.Fatalf("block 2 PoW: %v", err)
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

func TestRejectsTamperedMerkleRoot(t *testing.T) {
	chain := New()
	candidate := block.New(1, chain.Tip().BlockHash, config.GenesisTimestamp+60, 1, 0, nil, "")
	candidate.MerkleRoot = "tampered"

	err := chain.AddBlock(candidate)
	if !errors.Is(err, ErrInvalidMerkleRoot) {
		t.Fatalf("AddBlock error = %v, want ErrInvalidMerkleRoot", err)
	}
}

func TestRejectsWrongDifficulty(t *testing.T) {
	chain := New()
	candidate := block.New(1, chain.Tip().BlockHash, config.GenesisTimestamp+60, 2, 0, nil, "")

	err := chain.AddBlock(candidate)
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("AddBlock error = %v, want ErrInvalidDifficulty", err)
	}
}

func TestRejectsInvalidPoW(t *testing.T) {
	chain := New()
	candidate := block.New(1, chain.Tip().BlockHash, config.GenesisTimestamp+60, 1, 0, nil, "")

	for {
		candidate.BlockHash = candidate.CalculateHash()
		if errors.Is(consensus.ValidatePoW(candidate), consensus.ErrInvalidPoW) {
			break
		}
		candidate.Nonce++
	}

	err := chain.AddBlock(candidate)
	if !errors.Is(err, ErrInvalidPoW) {
		t.Fatalf("AddBlock error = %v, want ErrInvalidPoW", err)
	}
}

func TestRejectsTransactionWithMissingUTXO(t *testing.T) {
	chain := New()

	signer, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	recipientKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := valdrcrypto.AddressFromPublicKey(&recipientKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: strings.Repeat("88", 32),
			OutputIndex:           0,
		}},
		[]transaction.Output{{
			Amount:    10,
			Recipient: recipient,
		}},
		config.GenesisTimestamp+60,
	)
	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}

	candidate := block.New(
		1,
		chain.Tip().BlockHash,
		config.GenesisTimestamp+60,
		1,
		0,
		[]*transaction.Transaction{tx},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}

	err = chain.AddBlock(candidate)
	if !errors.Is(err, ErrInvalidTransaction) {
		t.Fatalf("AddBlock error = %v, want ErrInvalidTransaction", err)
	}
	if chain.Len() != 1 {
		t.Fatalf("chain length = %d after rejected block, want 1", chain.Len())
	}
}
