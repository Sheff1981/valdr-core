package blockchain

import (
	"errors"
	"fmt"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
)

var (
	ErrNilBlock           = errors.New("block is nil")
	ErrInvalidHeight      = errors.New("invalid block height")
	ErrPreviousHash       = errors.New("previous block hash mismatch")
	ErrInvalidMerkleRoot  = errors.New("invalid merkle root")
	ErrInvalidHash        = errors.New("invalid block hash")
	ErrWrongChainID       = errors.New("wrong chain id")
	ErrInvalidDifficulty  = errors.New("invalid difficulty")
	ErrInvalidPoW         = errors.New("invalid proof of work")
	ErrInvalidTransaction = errors.New("invalid block transaction")
)

type Blockchain struct {
	blocks []*block.Block
	utxos  *utxo.Set
}

func New() *Blockchain {
	return &Blockchain{
		blocks: []*block.Block{block.NewGenesis()},
		utxos:  utxo.NewEmpty(),
	}
}

func (bc *Blockchain) Len() int {
	return len(bc.blocks)
}

func (bc *Blockchain) Tip() *block.Block {
	if len(bc.blocks) == 0 {
		return nil
	}
	return bc.blocks[len(bc.blocks)-1]
}

func (bc *Blockchain) BlockAt(height uint64) (*block.Block, bool) {
	if height >= uint64(len(bc.blocks)) {
		return nil, false
	}
	return bc.blocks[height], true
}

func (bc *Blockchain) Balance(address string) (uint64, error) {
	return bc.utxos.Balance(address)
}

func (bc *Blockchain) UTXOs(address string) ([]utxo.UTXO, error) {
	return bc.utxos.List(address)
}

func (bc *Blockchain) UTXOSnapshot() []utxo.UTXO {
	return bc.utxos.Snapshot()
}

// Append mines and appends a candidate whose transaction list already contains
// its coinbase transaction at index zero. The mining package constructs it.
func (bc *Blockchain) Append(
	timestamp int64,
	transactions []*transaction.Transaction,
) (*block.Block, error) {
	tip := bc.Tip()
	if tip == nil {
		return nil, errors.New("blockchain has no genesis block")
	}

	difficulty := consensus.NextDifficulty(tip.Difficulty, tip.Timestamp, timestamp)
	candidate := block.New(
		tip.Height+1,
		tip.BlockHash,
		timestamp,
		difficulty,
		0,
		transactions,
		"",
	)

	if err := consensus.Mine(candidate); err != nil {
		return nil, err
	}
	if err := bc.AddBlock(candidate); err != nil {
		return nil, err
	}

	return candidate, nil
}

func (bc *Blockchain) AddBlock(candidate *block.Block) error {
	if candidate == nil {
		return ErrNilBlock
	}
	if candidate.ChainID != config.ChainID {
		return fmt.Errorf("%w: got %q want %q", ErrWrongChainID, candidate.ChainID, config.ChainID)
	}

	tip := bc.Tip()
	if tip == nil {
		return errors.New("blockchain has no genesis block")
	}
	if candidate.Height != tip.Height+1 {
		return fmt.Errorf("%w: got %d want %d", ErrInvalidHeight, candidate.Height, tip.Height+1)
	}
	if candidate.PreviousBlockHash != tip.BlockHash {
		return fmt.Errorf("%w: got %q want %q", ErrPreviousHash, candidate.PreviousBlockHash, tip.BlockHash)
	}

	expectedDifficulty := consensus.NextDifficulty(tip.Difficulty, tip.Timestamp, candidate.Timestamp)
	if candidate.Difficulty != expectedDifficulty {
		return fmt.Errorf(
			"%w: got %d want %d",
			ErrInvalidDifficulty,
			candidate.Difficulty,
			expectedDifficulty,
		)
	}

	expectedMerkleRoot := block.CalculateMerkleRoot(candidate.Transactions)
	if candidate.MerkleRoot != expectedMerkleRoot {
		return fmt.Errorf(
			"%w: got %q want %q",
			ErrInvalidMerkleRoot,
			candidate.MerkleRoot,
			expectedMerkleRoot,
		)
	}
	if candidate.BlockHash != candidate.CalculateHash() {
		return ErrInvalidHash
	}
	if err := consensus.ValidatePoW(candidate); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPoW, err)
	}

	reward := consensus.BlockReward(candidate.Height)
	if err := bc.utxos.ApplyBlockTransactions(
		candidate.Height,
		candidate.Transactions,
		reward,
	); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTransaction, err)
	}

	bc.blocks = append(bc.blocks, candidate)
	return nil
}
