package blockchain

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func testMinerAddress(t *testing.T) string {
	t.Helper()
	key, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	address, err := valdrcrypto.AddressFromPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return address
}

func testCoinbase(t *testing.T, height uint64, address string, timestamp int64) *transaction.Transaction {
	t.Helper()
	tx, err := transaction.NewCoinbase(
		height,
		address,
		consensus.BlockReward(height),
		timestamp,
	)
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestLocalChainGenesisBlock1Block2(t *testing.T) {
	chain := New()
	if chain.Len() != 1 {
		t.Fatalf("initial chain length = %d, want 1", chain.Len())
	}

	genesis := chain.Tip()
	if genesis == nil || genesis.Height != 0 {
		t.Fatal("chain did not start with genesis")
	}

	miner := testMinerAddress(t)
	block1, err := chain.Append(
		config.GenesisTimestamp+60,
		[]*transaction.Transaction{
			testCoinbase(t, 1, miner, config.GenesisTimestamp+60),
		},
	)
	if err != nil {
		t.Fatalf("append block 1: %v", err)
	}
	block2, err := chain.Append(
		config.GenesisTimestamp+120,
		[]*transaction.Transaction{
			testCoinbase(t, 2, miner, config.GenesisTimestamp+120),
		},
	)
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

	balance, err := chain.Balance(miner)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 2*config.InitialMiningReward {
		t.Fatalf("miner balance = %d, want %d", balance, 2*config.InitialMiningReward)
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

func TestRejectsMissingCoinbase(t *testing.T) {
	chain := New()
	candidate := block.New(
		1,
		chain.Tip().BlockHash,
		config.GenesisTimestamp+60,
		1,
		0,
		nil,
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}

	err := chain.AddBlock(candidate)
	if !errors.Is(err, utxo.ErrMissingCoinbase) {
		t.Fatalf("AddBlock error = %v, want ErrMissingCoinbase", err)
	}
}

func TestRejectsWrongCoinbaseReward(t *testing.T) {
	chain := New()
	miner := testMinerAddress(t)
	coinbase, err := transaction.NewCoinbase(
		1,
		miner,
		config.InitialMiningReward+1,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}
	candidate := block.New(
		1,
		chain.Tip().BlockHash,
		config.GenesisTimestamp+60,
		1,
		0,
		[]*transaction.Transaction{coinbase},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}

	err = chain.AddBlock(candidate)
	if !errors.Is(err, transaction.ErrInvalidCoinbaseReward) {
		t.Fatalf("AddBlock error = %v, want ErrInvalidCoinbaseReward", err)
	}
}

func TestRejectsTransactionWithMissingUTXO(t *testing.T) {
	chain := New()
	miner := testMinerAddress(t)

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

	coinbase := testCoinbase(t, 1, miner, config.GenesisTimestamp+60)
	candidate := block.New(
		1,
		chain.Tip().BlockHash,
		config.GenesisTimestamp+60,
		1,
		0,
		[]*transaction.Transaction{coinbase, tx},
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

func TestDifficultyRetargetBoundary(t *testing.T) {
	chain := New()
	miner := testMinerAddress(t)

	for height := uint64(1); height < config.DifficultyWindowBlocks; height++ {
		timestamp := config.GenesisTimestamp + int64(height)*15
		candidate, err := chain.Append(
			timestamp,
			[]*transaction.Transaction{
				testCoinbase(t, height, miner, timestamp),
			},
		)
		if err != nil {
			t.Fatalf("append block %d: %v", height, err)
		}
		if candidate.Difficulty != config.GenesisDifficulty {
			t.Fatalf(
				"block %d difficulty = %d, want %d before retarget boundary",
				height,
				candidate.Difficulty,
				config.GenesisDifficulty,
			)
		}
	}

	if got := consensus.NextDifficulty(chain.blocks); got != 4 {
		t.Fatalf("height 10 expected difficulty = %d, want 4", got)
	}

	height := config.DifficultyWindowBlocks
	timestamp := config.GenesisTimestamp + int64(height)*15
	candidate, err := chain.Append(
		timestamp,
		[]*transaction.Transaction{
			testCoinbase(t, height, miner, timestamp),
		},
	)
	if err != nil {
		t.Fatalf("append retarget block: %v", err)
	}
	if candidate.Height != height || candidate.Difficulty != 4 {
		t.Fatalf(
			"retarget block height=%d difficulty=%d, want height=%d difficulty=4",
			candidate.Height,
			candidate.Difficulty,
			height,
		)
	}
	if err := consensus.ValidatePoW(candidate); err != nil {
		t.Fatalf("retarget block PoW: %v", err)
	}
}

func TestRejectsOldDifficultyAtRetargetBoundary(t *testing.T) {
	chain := New()
	miner := testMinerAddress(t)

	for height := uint64(1); height < config.DifficultyWindowBlocks; height++ {
		timestamp := config.GenesisTimestamp + int64(height)*15
		if _, err := chain.Append(
			timestamp,
			[]*transaction.Transaction{
				testCoinbase(t, height, miner, timestamp),
			},
		); err != nil {
			t.Fatalf("append block %d: %v", height, err)
		}
	}

	height := config.DifficultyWindowBlocks
	timestamp := config.GenesisTimestamp + int64(height)*15
	candidate := block.New(
		height,
		chain.Tip().BlockHash,
		timestamp,
		config.GenesisDifficulty,
		0,
		[]*transaction.Transaction{
			testCoinbase(t, height, miner, timestamp),
		},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}

	err := chain.AddBlock(candidate)
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("retarget boundary error = %v, want ErrInvalidDifficulty", err)
	}
	if chain.Height() != height-1 {
		t.Fatalf("height = %d after rejected boundary block, want %d", chain.Height(), height-1)
	}
}
