package blockchain

import (
	"errors"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

type TransactionLocation struct {
	Transaction *transaction.Transaction
	BlockHeight uint64
	BlockHash   string
}

func (bc *Blockchain) BlockByHash(hash string) (*block.Block, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	node, ok := bc.nodes[hash]
	if !ok || node == nil {
		return nil, false
	}
	return node.block, true
}

func (bc *Blockchain) TransactionByID(transactionID string) (TransactionLocation, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for _, candidate := range bc.blocks {
		for _, tx := range candidate.Transactions {
			if tx != nil && tx.TransactionID == transactionID {
				return TransactionLocation{
					Transaction: tx,
					BlockHeight: candidate.Height,
					BlockHash:   candidate.BlockHash,
				}, true
			}
		}
	}
	return TransactionLocation{}, false
}

const (
	LocatorRecentCount = 10
	MaxHeadersPerBatch = 2000
)

var ErrNoCommonAncestor = errors.New("no common ancestor found")

// BlockLocator returns active-chain hashes from newest toward Genesis. The
// first entries are recent consecutive blocks, then the step doubles until
// Genesis is included.
func (bc *Blockchain) BlockLocator() []string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if len(bc.blocks) == 0 {
		return nil
	}

	locator := make([]string, 0, LocatorRecentCount+16)
	height := len(bc.blocks) - 1
	step := 1
	for {
		locator = append(locator, bc.blocks[height].BlockHash)
		if height == 0 {
			break
		}
		if len(locator) > LocatorRecentCount {
			step *= 2
		}
		if step > height {
			height = 0
		} else {
			height -= step
		}
	}
	return locator
}

// HeadersAfterLocator finds the first locator hash known on this active chain
// and returns the following active headers, bounded to MaxHeadersPerBatch.
func (bc *Blockchain) HeadersAfterLocator(
	locator []string,
	limit int,
) ([]block.Header, string, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if len(locator) == 0 || len(bc.blocks) == 0 {
		return nil, "", ErrNoCommonAncestor
	}
	if limit <= 0 || limit > MaxHeadersPerBatch {
		limit = MaxHeadersPerBatch
	}

	activeHeight := make(map[string]int, len(bc.blocks))
	for height, candidate := range bc.blocks {
		activeHeight[candidate.BlockHash] = height
	}

	ancestorHeight := -1
	ancestorHash := ""
	for _, hash := range locator {
		if height, ok := activeHeight[hash]; ok {
			ancestorHeight = height
			ancestorHash = hash
			break
		}
	}
	if ancestorHeight < 0 {
		return nil, "", ErrNoCommonAncestor
	}

	start := ancestorHeight + 1
	end := start + limit
	if end > len(bc.blocks) {
		end = len(bc.blocks)
	}
	headers := make([]block.Header, 0, end-start)
	for _, candidate := range bc.blocks[start:end] {
		headers = append(headers, candidate.Header())
	}
	return headers, ancestorHash, nil
}

func (bc *Blockchain) HeaderHistoryByHash(
	hash string,
) ([]consensus.V2DifficultyHeader, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	node, ok := bc.nodes[hash]
	if !ok || node == nil {
		return nil, false
	}
	return v2DifficultyHistory(node), true
}

func (bc *Blockchain) HasBlock(hash string) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	_, ok := bc.nodes[hash]
	return ok
}
