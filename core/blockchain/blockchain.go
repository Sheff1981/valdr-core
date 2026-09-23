package blockchain

import (
	"errors"
	"fmt"
	"sync"

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
	ErrInvalidTransaction   = errors.New("invalid block transaction")
	ErrDuplicateBlock       = errors.New("duplicate block")
	ErrDuplicateTransaction = errors.New("duplicate confirmed transaction")
)

type Blockchain struct {
	mu     sync.RWMutex
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
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.blocks)
}

func (bc *Blockchain) Height() uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if len(bc.blocks) == 0 {
		return 0
	}
	return bc.blocks[len(bc.blocks)-1].Height
}

func (bc *Blockchain) Tip() *block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if len(bc.blocks) == 0 {
		return nil
	}
	return bc.blocks[len(bc.blocks)-1]
}

func (bc *Blockchain) BlockAt(height uint64) (*block.Block, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if height >= uint64(len(bc.blocks)) {
		return nil, false
	}
	return bc.blocks[height], true
}

func (bc *Blockchain) Balance(address string) (uint64, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.utxos.Balance(address)
}

func (bc *Blockchain) UTXOs(address string) ([]utxo.UTXO, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.utxos.List(address)
}

func (bc *Blockchain) UTXOSnapshot() []utxo.UTXO {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.utxos.Snapshot()
}

// ValidateTransaction verifies a normal transaction against the current
// confirmed UTXO state without mutating the blockchain.
func (bc *Blockchain) ValidateTransaction(tx *transaction.Transaction) error {
	bc.mu.RLock()
	snapshot := bc.utxos.Snapshot()
	bc.mu.RUnlock()

	working, err := utxo.New(snapshot)
	if err != nil {
		return err
	}
	return working.ApplyTransaction(tx)
}

// Append mines and appends a candidate whose transaction list already contains
// its coinbase transaction at index zero. The mining package constructs it.
func (bc *Blockchain) Append(
	timestamp int64,
	transactions []*transaction.Transaction,
) (*block.Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(bc.blocks) == 0 {
		return nil, errors.New("blockchain has no genesis block")
	}
	tip := bc.blocks[len(bc.blocks)-1]

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
	if err := bc.addBlockLocked(candidate); err != nil {
		return nil, err
	}

	return candidate, nil
}

func (bc *Blockchain) AddBlock(candidate *block.Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.addBlockLocked(candidate)
}

func (bc *Blockchain) addBlockLocked(candidate *block.Block) error {
	if candidate == nil {
		return ErrNilBlock
	}
	if candidate.ChainID != config.ChainID {
		return fmt.Errorf("%w: got %q want %q", ErrWrongChainID, candidate.ChainID, config.ChainID)
	}

	if candidate.BlockHash != "" &&
		candidate.BlockHash == candidate.CalculateHash() &&
		bc.hasBlockHashLocked(candidate.BlockHash) {
		return fmt.Errorf("%w: %s", ErrDuplicateBlock, candidate.BlockHash)
	}

	if len(bc.blocks) == 0 {
		return errors.New("blockchain has no genesis block")
	}
	tip := bc.blocks[len(bc.blocks)-1]

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

	if err := bc.validateTransactionUniquenessLocked(candidate.Transactions); err != nil {
		return err
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


func (bc *Blockchain) hasBlockHashLocked(hash string) bool {
	for _, existing := range bc.blocks {
		if existing != nil && existing.BlockHash == hash {
			return true
		}
	}
	return false
}

func (bc *Blockchain) validateTransactionUniquenessLocked(
	transactions []*transaction.Transaction,
) error {
	seen := make(map[string]struct{}, len(transactions))
	for _, tx := range transactions {
		if tx == nil || tx.TransactionID == "" {
			continue
		}
		if _, exists := seen[tx.TransactionID]; exists {
			return fmt.Errorf(
				"%w in candidate block: %s",
				ErrDuplicateTransaction,
				tx.TransactionID,
			)
		}
		seen[tx.TransactionID] = struct{}{}

		for _, existingBlock := range bc.blocks {
			if existingBlock == nil {
				continue
			}
			for _, existingTx := range existingBlock.Transactions {
				if existingTx != nil && existingTx.TransactionID == tx.TransactionID {
					return fmt.Errorf(
						"%w: %s",
						ErrDuplicateTransaction,
						tx.TransactionID,
					)
				}
			}
		}
	}
	return nil
}
