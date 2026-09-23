package blockchain

import (
	"errors"
	"fmt"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
)

var (
	ErrNilBlock          = errors.New("block is nil")
	ErrInvalidHeight     = errors.New("invalid block height")
	ErrPreviousHash      = errors.New("previous block hash mismatch")
	ErrInvalidHash       = errors.New("invalid block hash")
	ErrWrongChainID      = errors.New("wrong chain id")
	ErrInvalidDifficulty = errors.New("invalid difficulty")
	ErrInvalidPoW        = errors.New("invalid proof of work")
)

// Blockchain is the in-memory local chain used by VALDR during the early MVP.
// Persistent storage is intentionally deferred to the storage stage in the master plan.
type Blockchain struct {
	blocks []*block.Block
}

// New creates a local VALDR chain containing the fixed genesis block.
func New() *Blockchain {
	return &Blockchain{blocks: []*block.Block{block.NewGenesis()}}
}

// Len returns the number of blocks currently in the chain, including genesis.
func (bc *Blockchain) Len() int {
	return len(bc.blocks)
}

// Tip returns the current chain tip.
func (bc *Blockchain) Tip() *block.Block {
	if len(bc.blocks) == 0 {
		return nil
	}
	return bc.blocks[len(bc.blocks)-1]
}

// BlockAt returns the block at a height when it exists.
func (bc *Blockchain) BlockAt(height uint64) (*block.Block, bool) {
	if height >= uint64(len(bc.blocks)) {
		return nil, false
	}
	return bc.blocks[height], true
}

// Append creates, mines and appends the next local block using the simplified
// v0.1 difficulty rule. Coinbase/reward handling is introduced later in the plan.
func (bc *Blockchain) Append(timestamp int64, transactions []string) (*block.Block, error) {
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

// AddBlock validates chain linkage, expected difficulty, block hash and PoW
// before appending a non-genesis block.
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
	if candidate.BlockHash != candidate.CalculateHash() {
		return ErrInvalidHash
	}
	if err := consensus.ValidatePoW(candidate); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPoW, err)
	}

	bc.blocks = append(bc.blocks, candidate)
	return nil
}
