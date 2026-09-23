package blockchain

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
)

var (
	ErrNilBlock                   = errors.New("block is nil")
	ErrInvalidHeight              = errors.New("invalid block height")
	ErrPreviousHash               = errors.New("previous block hash mismatch")
	ErrInvalidMerkleRoot          = errors.New("invalid merkle root")
	ErrInvalidHash                = errors.New("invalid block hash")
	ErrWrongChainID               = errors.New("wrong chain id")
	ErrInvalidDifficulty          = errors.New("invalid difficulty")
	ErrInvalidPoW                 = errors.New("invalid proof of work")
	ErrInvalidTransaction         = errors.New("invalid block transaction")
	ErrDuplicateBlock             = errors.New("duplicate block")
	ErrDuplicateTransaction       = errors.New("duplicate confirmed transaction")
	ErrInvalidStoredChain         = errors.New("invalid stored blockchain")
	ErrPersistence                = errors.New("blockchain persistence failed")
	ErrNilBlockStore              = errors.New("block store is nil")
	ErrForkPersistenceUnsupported = errors.New("block store does not support side branches")
)

type BlockStore interface {
	Load() ([]*block.Block, error)
	Save([]*block.Block) error
}

type IndexedBlockStore interface {
	BlockStore
	SaveBlock(*block.Block, []utxo.UTXO, []utxo.UTXO) error
}

// ForkAwareBlockStore is the v0.2 persistence contract for competing branches.
// CommitCandidate must persist the candidate and, when activeChain is non-nil,
// atomically switch all active-chain indexes to that chain.
type ForkAwareBlockStore interface {
	BlockStore
	LoadAllBlocks() ([]*block.Block, error)
	CommitCandidate(
		candidate *block.Block,
		before []utxo.UTXO,
		after []utxo.UTXO,
		chainwork string,
		activeChain []*block.Block,
		activeUTXO []utxo.UTXO,
	) error
}

type chainNode struct {
	block     *block.Block
	parent    *chainNode
	chainwork *big.Int
	utxos     []utxo.UTXO
}

type Blockchain struct {
	mu     sync.RWMutex
	blocks []*block.Block
	utxos  *utxo.Set
	nodes  map[string]*chainNode
	tip    *chainNode
	store  BlockStore
}

func New() *Blockchain {
	genesis := block.NewGenesis()
	work, err := consensus.AddWork(nil, genesis.Difficulty)
	if err != nil {
		panic(fmt.Sprintf("VALDR genesis chainwork: %v", err))
	}
	node := &chainNode{
		block:     genesis,
		chainwork: work,
		utxos:     nil,
	}
	return &Blockchain{
		blocks: []*block.Block{genesis},
		utxos:  utxo.NewEmpty(),
		nodes:  map[string]*chainNode{genesis.BlockHash: node},
		tip:    node,
	}
}

func NewPersistent(store BlockStore) (*Blockchain, error) {
	if store == nil {
		return nil, ErrNilBlockStore
	}
	loaded, err := store.Load()
	if err != nil {
		return nil, err
	}

	bc := New()
	if len(loaded) == 0 {
		if err := store.Save(bc.blocks); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPersistence, err)
		}
		bc.store = store
		return bc, nil
	}

	expectedGenesis := block.NewGenesis()
	storedGenesis := loaded[0]
	if storedGenesis == nil ||
		storedGenesis.Height != 0 ||
		storedGenesis.ChainID != config.ChainID ||
		storedGenesis.BlockHash != expectedGenesis.BlockHash ||
		storedGenesis.CalculateHash() != expectedGenesis.BlockHash {
		return nil, ErrInvalidStoredChain
	}

	for i := 1; i < len(loaded); i++ {
		if err := bc.addBlockLocked(loaded[i]); err != nil {
			return nil, fmt.Errorf(
				"%w at height %d: %v",
				ErrInvalidStoredChain,
				i,
				err,
			)
		}
	}
	persistedTip := loaded[len(loaded)-1].BlockHash

	if forkStore, ok := store.(ForkAwareBlockStore); ok {
		all, err := forkStore.LoadAllBlocks()
		if err != nil {
			return nil, err
		}
		sort.Slice(all, func(i, j int) bool {
			if all[i].Height == all[j].Height {
				return all[i].BlockHash < all[j].BlockHash
			}
			return all[i].Height < all[j].Height
		})
		for _, candidate := range all {
			if candidate == nil {
				return nil, ErrInvalidStoredChain
			}
			if _, exists := bc.nodes[candidate.BlockHash]; exists {
				continue
			}
			if err := bc.addBlockLocked(candidate); err != nil {
				return nil, fmt.Errorf(
					"%w loading branch block %s: %v",
					ErrInvalidStoredChain,
					candidate.BlockHash,
					err,
				)
			}
		}
		if bc.tip == nil || bc.tip.block.BlockHash != persistedTip {
			return nil, fmt.Errorf(
				"%w: stored active tip %s is not greatest-chainwork tip %s",
				ErrInvalidStoredChain,
				persistedTip,
				bc.tip.block.BlockHash,
			)
		}
	}

	bc.store = store
	return bc, nil
}

func (bc *Blockchain) Len() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.blocks)
}

func (bc *Blockchain) Height() uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if bc.tip == nil {
		return 0
	}
	return bc.tip.block.Height
}

func (bc *Blockchain) Tip() *block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if bc.tip == nil {
		return nil
	}
	return bc.tip.block
}

func (bc *Blockchain) Chainwork() string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if bc.tip == nil {
		return ""
	}
	return consensus.ChainworkHex(bc.tip.chainwork)
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
// active-chain UTXO state without mutating the blockchain.
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

// Append mines and appends a candidate on the current active tip.
func (bc *Blockchain) Append(
	timestamp int64,
	transactions []*transaction.Transaction,
) (*block.Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.tip == nil {
		return nil, errors.New("blockchain has no genesis block")
	}
	tip := bc.tip.block
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
		candidate.BlockHash == candidate.CalculateHash() {
		if _, exists := bc.nodes[candidate.BlockHash]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateBlock, candidate.BlockHash)
		}
	}

	parent, exists := bc.nodes[candidate.PreviousBlockHash]
	if !exists {
		return fmt.Errorf("%w: unknown parent %q", ErrPreviousHash, candidate.PreviousBlockHash)
	}
	if candidate.Height != parent.block.Height+1 {
		return fmt.Errorf(
			"%w: got %d want %d",
			ErrInvalidHeight,
			candidate.Height,
			parent.block.Height+1,
		)
	}

	afterSet, err := bc.validateCandidateLocked(candidate, parent)
	if err != nil {
		return err
	}
	after := afterSet.Snapshot()
	chainwork, err := consensus.AddWork(parent.chainwork, candidate.Difficulty)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDifficulty, err)
	}

	newNode := &chainNode{
		block:     candidate,
		parent:    parent,
		chainwork: chainwork,
		utxos:     after,
	}
	becomesActive := bc.tip == nil || chainwork.Cmp(bc.tip.chainwork) > 0
	var activeChain []*block.Block
	var activeUTXO []utxo.UTXO
	if becomesActive {
		activeChain = activePath(newNode)
		activeUTXO = after
	}

	if bc.store != nil {
		if forkStore, ok := bc.store.(ForkAwareBlockStore); ok {
			if err := forkStore.CommitCandidate(
				candidate,
				parent.utxos,
				after,
				consensus.ChainworkHex(chainwork),
				activeChain,
				activeUTXO,
			); err != nil {
				return fmt.Errorf("%w: %v", ErrPersistence, err)
			}
		} else {
			if parent != bc.tip {
				return ErrForkPersistenceUnsupported
			}
			if indexed, ok := bc.store.(IndexedBlockStore); ok {
				if err := indexed.SaveBlock(candidate, parent.utxos, after); err != nil {
					return fmt.Errorf("%w: %v", ErrPersistence, err)
				}
			} else {
				next := append(append([]*block.Block(nil), bc.blocks...), candidate)
				if err := bc.store.Save(next); err != nil {
					return fmt.Errorf("%w: %v", ErrPersistence, err)
				}
			}
		}
	}

	bc.nodes[candidate.BlockHash] = newNode
	if becomesActive {
		bc.blocks = activeChain
		bc.tip = newNode
		bc.utxos = afterSet
	}
	return nil
}

func (bc *Blockchain) validateCandidateLocked(
	candidate *block.Block,
	parent *chainNode,
) (*utxo.Set, error) {
	expectedDifficulty := consensus.NextDifficulty(
		parent.block.Difficulty,
		parent.block.Timestamp,
		candidate.Timestamp,
	)
	if candidate.Difficulty != expectedDifficulty {
		return nil, fmt.Errorf(
			"%w: got %d want %d",
			ErrInvalidDifficulty,
			candidate.Difficulty,
			expectedDifficulty,
		)
	}

	if err := validateTransactionUniquenessOnBranch(candidate.Transactions, parent); err != nil {
		return nil, err
	}

	expectedMerkleRoot := block.CalculateMerkleRoot(candidate.Transactions)
	if candidate.MerkleRoot != expectedMerkleRoot {
		return nil, fmt.Errorf(
			"%w: got %q want %q",
			ErrInvalidMerkleRoot,
			candidate.MerkleRoot,
			expectedMerkleRoot,
		)
	}
	if candidate.BlockHash != candidate.CalculateHash() {
		return nil, ErrInvalidHash
	}
	if err := consensus.ValidatePoW(candidate); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPoW, err)
	}

	working, err := utxo.New(parent.utxos)
	if err != nil {
		return nil, err
	}
	reward := consensus.BlockReward(candidate.Height)
	if err := working.ApplyBlockTransactions(
		candidate.Height,
		candidate.Transactions,
		reward,
	); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidTransaction, err)
	}
	return working, nil
}

func validateTransactionUniquenessOnBranch(
	transactions []*transaction.Transaction,
	parent *chainNode,
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

		for node := parent; node != nil; node = node.parent {
			for _, existingTx := range node.block.Transactions {
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

func activePath(tip *chainNode) []*block.Block {
	if tip == nil {
		return nil
	}
	reversed := make([]*block.Block, 0, tip.block.Height+1)
	for node := tip; node != nil; node = node.parent {
		reversed = append(reversed, node.block)
	}
	path := make([]*block.Block, len(reversed))
	for i := range reversed {
		path[len(reversed)-1-i] = reversed[i]
	}
	return path
}
